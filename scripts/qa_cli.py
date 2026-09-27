#!/usr/bin/env python3
"""Black-box CLI QA with real Git, isolated homes, concurrency, and JSON timings."""

import argparse
import concurrent.futures
import json
import math
import os
from pathlib import Path
import platform
import re
import shutil
import statistics
import subprocess
import tempfile
import threading
import time
from datetime import datetime, timezone


PROJECT = Path(__file__).resolve().parents[1]
REMOTE = "https://qa.invalid/demo.git"


def require(condition, message):
    if not condition:
        raise AssertionError(message)


def execute(args, **kwargs):
    return subprocess.run(args, capture_output=True, text=True, timeout=60, **kwargs)


def checked(args, **kwargs):
    result = execute(args, **kwargs)
    require(result.returncode == 0, f"{args}: {result.stdout} {result.stderr}")
    return result.stdout.strip()


def distribution(values):
    ordered = sorted(values)
    return {
        "count": len(values),
        "median_ms": round(statistics.median(values), 3),
        "p95_ms": round(ordered[math.ceil(len(values) * .95) - 1], 3),
        "max_ms": round(max(values), 3),
    }


def parse_cli_text(args, output, failed):
    lines = output.rstrip("\n").split("\n")
    require(lines and lines[0], f"empty CLI output: {args}")
    if not failed and args[0] == "list":
        if lines[0] in ("No repositories found.", "No repositories match."):
            return []
        require(lines[0].startswith("REPOSITORY") and "REFERENCE" in lines[0]
                and "CHECKOUT" in lines[0], f"missing list header: {output}")
        entries = []
        for line in lines[1:]:
            columns = re.split(r"\s{2,}", line, maxsplit=2)
            require(len(columns) == 3, f"invalid list row: {line}")
            kind, value = columns[1].split(" ", 1)
            entries.append({"url": columns[0], "reference": {"type": kind, "value": value},
                            "path": columns[2]})
        return entries

    first = lines[0]
    context = {}
    if failed:
        require(first.startswith("error: "), f"missing text error: {output}")
        data = {"error": first.removeprefix("error: "), "context": context}
    elif first.startswith("Repository: "):
        data = {"status": "added", "url": first.removeprefix("Repository: ")}
    elif first.startswith("Removed from catalogue: "):
        data = {"status": "removed", "url": first.removeprefix("Removed from catalogue: ")}
    else:
        raise AssertionError(f"unexpected CLI output: {output}")
    fields = {"Repository": "url", "Checkout": "path", "Config": "path",
              "Checkout root": "installDir", "Lock file": "lockPath", "Hint": "hint",
              "Cause": "cause", "Git command": "gitCommand", "Added": "addedAt",
              "Usage": "usage", "Cleanup error": "cleanupError"}
    for line in lines[1:]:
        line = line.strip()
        if ": " not in line:
            continue
        label, value = line.split(": ", 1)
        value = value.strip()
        target = context if failed else data
        if label == "Reference":
            kind, reference = value.split(" ", 1)
            target["reference"] = {"type": kind, "value": reference}
        elif label in fields:
            target[fields[label]] = value
            if label == "Cause" and "clone" in data.get("error", ""):
                target["gitError"] = value
    return data


class QA:
    def __init__(self, root, report, workers):
        self.root, self.report, self.workers = root, report, workers
        self.binary = root / "robert"
        # HOME is changed only for child processes. Never touch the real catalogue.
        self.env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
        self.env.update(GIT_CONFIG_NOSYSTEM="1", GIT_TERMINAL_PROMPT="0")
        self.env["GIT_CONFIG_GLOBAL"] = str(root / "gitconfig")
        self.env["GIT_ALLOW_PROTOCOL"] = "file"
        self.env["LC_ALL"] = "C"
        self.records_lock = threading.Lock()

    def setup(self):
        started = time.perf_counter()
        checked(["go", "build", "-o", str(self.binary), "./cmd/robert"], cwd=PROJECT)
        self.report["build_ms"] = round((time.perf_counter() - started) * 1000, 3)
        source = self.root / "source"
        source.mkdir()
        git = lambda *args: checked(["git", *args], cwd=source, env=self.env)
        git("init", "--quiet", "--initial-branch=main", "--template=")
        git("config", "user.name", "Robert QA")
        git("config", "user.email", "qa@example.invalid")
        (source / "README.md").write_text("main fixture\n")
        git("add", "README.md")
        git("-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "main")
        self.main_id = git("rev-parse", "HEAD")
        git("tag", "v1.0.0")
        git("checkout", "--quiet", "-b", "develop")
        (source / "README.md").write_text("develop fixture\n")
        git("-c", "commit.gpgsign=false", "commit", "--quiet", "-am", "develop")
        self.develop_id = git("rev-parse", "HEAD")
        git("checkout", "--quiet", "main")
        remotes = self.root / "remotes"
        remotes.mkdir()
        git("clone", "--quiet", "--bare", str(source), str(remotes / "demo.git"))
        for i in range(self.workers * 2):
            (remotes / f"repo-{i}.git").symlink_to("demo.git")
        checked(["git", "config", "--file", self.env["GIT_CONFIG_GLOBAL"],
                 f"url.{remotes.as_uri()}/.insteadOf", "https://qa.invalid/"], env=self.env)

    def cli(self, *args, code=0):
        started = time.perf_counter()
        result = execute([str(self.binary), *args], env={**self.env, "HOME": str(self.home)})
        record = {"case": self.case, "args": args, "exit_code": result.returncode,
                  "ms": round((time.perf_counter() - started) * 1000, 3),
                  "stdout": result.stdout, "stderr": result.stderr}
        with self.records_lock:
            self.report["commands"].append(record)
        require(code is None or result.returncode == code, f"unexpected exit: {record}")
        require(result.returncode in (0, 1), f"invalid exit code: {record}")
        data = parse_cli_text(args, result.stdout if result.returncode == 0 else result.stderr,
                              result.returncode != 0)
        if result.returncode:
            require(not result.stdout, f"unexpected stdout on failure: {record}")
        elif result.stderr:
            require(args[0] == "remove" and result.stderr.startswith("warning: could not delete checkout: ")
                    and "  Checkout: " in result.stderr, f"unexpected warning: {record}")
            data["warning"] = result.stderr
        if result.returncode:
            require(isinstance(data, dict) and isinstance(data.get("error"), str)
                    and data["error"] and data.get("context"), f"error lacks context: {record}")
            require(set(data) <= {"error", "context"}, f"unexpected error shape: {data}")
        elif args[0] in ("add", "remove"):
            require(data.get("status") == {"add": "added", "remove": "removed"}[args[0]],
                    f"wrong success status: {record}")
            require(data.get("url") == args[1] and Path(data.get("path", "")).is_absolute()
                    and data.get("reference", {}).get("type") in ("branch", "tag", "commit"),
                    f"incomplete success response: {record}")
        return data

    def batch(self, jobs):
        barrier = threading.Barrier(len(jobs))

        def run(job):
            barrier.wait(timeout=30)
            args, code = job
            return self.cli(*args, code=code)

        started = time.perf_counter()
        with concurrent.futures.ThreadPoolExecutor(max_workers=len(jobs)) as pool:
            results = list(pool.map(run, jobs))
        elapsed = (time.perf_counter() - started) * 1000
        self.report["batches"].append({"case": self.case, "commands": len(jobs),
                                       "wall_ms": round(elapsed, 3),
                                       "commands_per_second": round(len(jobs) * 1000 / elapsed, 2)})
        return results

    def config(self):
        return self.home / ".robert"

    def verify(self, count):
        entries = json.loads(self.config().read_text())["repositories"] if self.config().exists() else []
        require(len(entries) == count, f"expected {count} entries, got {entries}")
        keys = {(e["url"], e["reference"]["type"], e["reference"]["value"]) for e in entries}
        require(len(keys) == count, "duplicate catalogue entries")
        install = self.home / ".agents" / "robert"
        paths = {str(p) for p in install.iterdir()} if install.exists() else set()
        require(paths == {e["path"] for e in entries}, f"orphan/missing checkout: {paths}, {entries}")
        listed = self.cli("list")
        expected = [{k: e[k] for k in ("url", "reference", "path")} for e in entries]
        require(sorted(listed, key=lambda e: e["path"]) == sorted(expected, key=lambda e: e["path"]),
                "list does not match saved catalogue")
        for entry in entries:
            path, ref = Path(entry["path"]), entry["reference"]
            require(path.is_relative_to(install), f"checkout outside isolated home: {path}")
            git = lambda *args: checked(["git", "-C", str(path), *args], env=self.env)
            expected_id = self.develop_id if ref["value"] == "develop" else self.main_id
            require(git("rev-parse", "HEAD") == expected_id, f"wrong checkout: {entry}")
            require(git("remote", "get-url", "--push", "origin") != "", "missing origin")
            require(git("config", "remote.origin.url") == entry["url"], "wrong origin URL")
            require(git("rev-parse", "--is-shallow-repository") == "true", "checkout is not shallow")
            require(git("status", "--porcelain") == "", "dirty checkout")
            content = "develop fixture\n" if expected_id == self.develop_id else "main fixture\n"
            require((path / "README.md").read_text() == content, "incorrect worktree content")
            if ref["type"] == "branch":
                require(git("rev-parse", "--abbrev-ref", "HEAD") == ref["value"], "wrong branch")
                require(git("rev-parse", "--abbrev-ref", "@{upstream}") == "origin/" + ref["value"],
                        "wrong tracking branch")
            else:
                require(git("rev-parse", "--abbrev-ref", "HEAD") == "HEAD", "HEAD not detached")
        require(not list(self.home.glob(".robert-*")), "temporary catalogue files leaked")
        return entries

    def lifecycle(self):
        require(self.cli("list") == [], "fresh catalogue not empty")
        require(not self.config().exists(), "read created configuration")
        self.cli("remove", REMOTE, code=1)
        require(not self.config().exists(), "failed remove created configuration")
        first = self.cli("add", REMOTE)
        before = self.config().read_bytes()
        duplicate = self.cli("add", REMOTE, "--branch", "main")
        require(first == duplicate and before == self.config().read_bytes(), "duplicate changed state")
        for flag, value in (("--branch", "develop"), ("--tag", "v1.0.0"), ("--commit", self.main_id.upper())):
            self.cli("add", REMOTE, flag, value)
        entries = self.verify(4)
        before = self.config().read_bytes()
        self.cli("remove", REMOTE, code=1)
        require(before == self.config().read_bytes(), "ambiguous removal changed state")
        for entry in entries:
            ref = entry["reference"]
            removed = self.cli("remove", REMOTE, "--" + ref["type"], ref["value"])
            require(removed["status"] == "removed" and removed["path"] == entry["path"], "wrong removal")
            require(not Path(entry["path"]).exists(), "removed checkout remains")
        self.verify(0)
        self.cli("remove", REMOTE, code=1)

    def failed_adds(self):
        for args in (("--branch", "missing"), ("--tag", "missing"), ("--commit", "f" * 40)):
            result = self.cli("add", REMOTE, *args, code=1)
            require(result["context"].get("gitError") and result["context"].get("url"), "missing Git context")
            require(not self.config().exists(), "failed clone created configuration")
            self.verify(0)
        self.cli("add", "https://qa.invalid/absent.git", code=1)
        self.verify(0)
        for args in (("add",), ("remove",), ("add", REMOTE, "--commit", "bad"),
                     ("remove", REMOTE, "--branch", "main", "--tag", "v1.0.0")):
            self.cli(*args, code=1)

    def serial_timing(self):
        for _ in range(20):
            self.cli("add", REMOTE, "--branch", "main")
            self.verify(1)
            self.cli("remove", REMOTE)
            self.verify(0)

    def concurrent_distinct(self):
        urls = [f"https://qa.invalid/repo-{i}.git" for i in range(self.workers)]
        results = self.batch([(("add", url, "--branch", "main"), 0) for url in urls])
        entries = self.verify(self.workers)
        require({e["url"] for e in entries} == set(urls), "concurrent additions lost")
        require({r["path"] for r in results} == {e["path"] for e in entries}, "wrong add results")
        self.batch([(("remove", url), 0) for url in urls])
        self.verify(0)

    def concurrent_duplicates(self):
        results = self.batch([(("add", REMOTE, "--branch", "main"), 0)] * self.workers)
        require(all(r == results[0] for r in results), "duplicate add results disagree")
        self.verify(1)
        results = self.batch([(("remove", REMOTE), None)] * self.workers)
        require(sum(r.get("status") == "removed" for r in results) == 1, "multiple remove winners")
        require(sum("not found" in r.get("error", "") for r in results) == self.workers - 1,
                "losing removes did not return not-found")
        self.verify(0)

    def mixed(self):
        old = [f"https://qa.invalid/repo-{i}.git" for i in range(self.workers)]
        new = [f"https://qa.invalid/repo-{i + self.workers}.git" for i in range(self.workers)]
        self.batch([(("add", url, "--branch", "main"), 0) for url in old])
        jobs = ([(("remove", url), 0) for url in old]
                + [(("add", url, "--branch", "main"), 0) for url in new]
                + [(("list",), 0)] * self.workers)
        self.batch(jobs)
        entries = self.verify(self.workers)
        require({e["url"] for e in entries} == set(new), "mixed workload lost updates")
        self.batch([(("remove", url), 0) for url in new])
        self.verify(0)

    def same_key_race(self):
        self.cli("add", REMOTE, "--branch", "main")
        jobs = ([(("add", REMOTE, "--branch", "main"), 0)] * self.workers
                + [(("remove", REMOTE), None)] * self.workers)
        self.batch(jobs)
        count = len(json.loads(self.config().read_text())["repositories"])
        require(count in (0, 1), "same-key race created duplicate entries")
        self.verify(count)
        if count:
            self.cli("remove", REMOTE)
        self.verify(0)

    def save_failure(self):
        self.cli("add", REMOTE, "--branch", "main")
        before = self.config().read_bytes()
        self.home.chmod(0o500)
        try:
            for args in (("remove", REMOTE), ("add", REMOTE, "--branch", "develop")):
                self.cli(*args, code=1)
                require(self.config().read_bytes() == before, "failed save changed catalogue")
                self.verify(1)
        finally:
            self.home.chmod(0o700)
        self.cli("remove", REMOTE)
        self.verify(0)

    def lock_failure(self):
        (self.home / ".robert.lock").mkdir()
        for args in (("list",), ("add", REMOTE, "--branch", "main"), ("remove", REMOTE)):
            result = self.cli(*args, code=1)
            require(result["context"].get("lockPath") and result["context"].get("hint"),
                    "lock error lacks recovery context")
        (self.home / ".robert.lock").rmdir()
        self.verify(0)

    def deletion_failure(self):
        added = self.cli("add", REMOTE, "--branch", "main")
        checkout = Path(added["path"])
        protected = checkout / "protected"
        protected.mkdir()
        (protected / "file").write_text("cannot unlink while parent is read-only\n")
        protected.chmod(0o500)
        try:
            result = self.cli("remove", REMOTE, code=0)
            remaining = (protected / "file").exists()
            require(result["status"] == "removed", "remove did not report catalogue removal")
            require(bool(result.get("warning")) == remaining,
                    f"checkout warning disagrees with directory state: {result}")
            require(self.cli("list") == [], "failed deletion retained entry unexpectedly")
            self.cli("remove", REMOTE, code=1)
        finally:
            if protected.exists():
                protected.chmod(0o700)
            if checkout.exists():
                shutil.rmtree(checkout)
        self.verify(0)

    def redundant_cleanup_failure(self):
        # Synchronize two real clones immediately before they publish their entries.
        # Both have a protected directory, so whichever loses cannot fully clean up.
        wrapper_dir = self.root / "git-wrapper"
        wrapper_dir.mkdir()
        gate = self.root / "clone-gate"
        gate.mkdir()
        wrapper = wrapper_dir / "git"
        wrapper.write_text(f"""#!{os.sys.executable}
import os
from pathlib import Path
import subprocess
import sys
import time

result = subprocess.run([{shutil.which('git')!r}, *sys.argv[1:]])
if result.returncode == 0 and sys.argv[1:] == ['config', 'branch.main.merge', 'refs/heads/main']:
    checkout = Path.cwd()
    protected = checkout / 'protected'
    protected.mkdir()
    (protected / 'file').write_text('QA cleanup failure probe')
    protected.chmod(0o500)
    gate = Path({str(gate)!r})
    (gate / checkout.name).touch()
    deadline = time.monotonic() + 20
    while len(list(gate.iterdir())) < 2:
        if time.monotonic() > deadline:
            sys.exit('QA clone barrier timed out')
        time.sleep(0.005)
sys.exit(result.returncode)
""")
        wrapper.chmod(0o700)
        old_path = self.env["PATH"]
        self.env["PATH"] = str(wrapper_dir) + os.pathsep + old_path
        install = self.home / ".agents" / "robert"
        try:
            results = self.batch([(("add", REMOTE, "--branch", "main"), 0)] * 2)
            require(results[0] == results[1], "duplicate race returned different winners")
            entries = json.loads(self.config().read_text())["repositories"]
            require(len(entries) == 1, "duplicate race saved multiple entries")
            orphans = [str(p) for p in install.iterdir() if str(p) != entries[0]["path"]]
            if orphans:
                self.report["findings"].append({
                    "id": "silent-redundant-clone-cleanup-failure",
                    "case": self.case,
                    "severity": "medium",
                    "summary": "Concurrent duplicate adds both exit 0 but failed redundant-clone cleanup leaves an unregistered directory",
                    "reproduction": "Synchronize two real clones before catalogue update and create a nonempty 0500 directory in each checkout.",
                    "impact": "The losing checkout is absent from list and cannot be removed through the CLI; manual cleanup is required.",
                    "design_note": "Explicitly accepted in ADR 0005; future automatic cleanup is mentioned but not implemented.",
                    "orphan_paths": orphans,
                })
        finally:
            self.env["PATH"] = old_path
            if install.exists():
                for checkout in install.iterdir():
                    protected = checkout / "protected"
                    if protected.exists():
                        protected.chmod(0o700)
        self.cli("remove", REMOTE)
        for checkout in install.iterdir():
            shutil.rmtree(checkout)
        self.verify(0)

    def run_case(self, name, action):
        self.case = name
        self.home = self.root / name
        self.home.mkdir()
        started = time.perf_counter()
        result = {"name": name}
        findings_before = len(self.report["findings"])
        try:
            action()
            result["status"] = "issues_found" if len(self.report["findings"]) > findings_before else "passed"
        except Exception as error:
            result.update(status="failed", error=f"{type(error).__name__}: {error}")
        result["wall_ms"] = round((time.perf_counter() - started) * 1000, 3)
        self.report["cases"].append(result)
        print(f"{name}: {result['status']} ({result['wall_ms']:.1f} ms)", file=os.sys.stderr)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workers", type=int, default=16)
    parser.add_argument("--rounds", type=int, default=5)
    parser.add_argument("--report", type=Path, default=Path("/tmp/robert-qa-results.json"))
    args = parser.parse_args()
    if args.workers < 2 or args.rounds < 1:
        parser.error("workers must be >= 2 and rounds >= 1")
    report = {"started_at": datetime.now(timezone.utc).isoformat(), "platform": platform.platform(),
              "workers": args.workers, "rounds": args.rounds, "cases": [], "commands": [],
              "batches": [], "findings": [], "scope": "Local real Git fixtures; no internet latency; Linux/macOS only"}
    report["revision"] = checked(["git", "rev-parse", "HEAD"], cwd=PROJECT)
    report["go_version"] = checked(["go", "version"])
    report["git_version"] = checked(["git", "--version"])
    started = time.perf_counter()
    root = None
    try:
        with tempfile.TemporaryDirectory(prefix="robert-qa-") as temp:
            root = Path(temp)
            report["temporary_root"] = str(root)
            qa = QA(root, report, args.workers)
            qa.setup()
            qa.run_case("lifecycle", qa.lifecycle)
            qa.run_case("serial-timing", qa.serial_timing)
            qa.run_case("failed-adds-and-arguments", qa.failed_adds)
            qa.run_case("lock-failure", qa.lock_failure)
            for i in range(args.rounds):
                for name, action in (("distinct", qa.concurrent_distinct), ("duplicates", qa.concurrent_duplicates),
                                     ("mixed", qa.mixed), ("same-key-race", qa.same_key_race)):
                    qa.run_case(f"{name}-{i + 1}", action)
            if os.geteuid() == 0:
                report["cases"].append({"name": "permission-probes", "status": "skipped", "reason": "requires non-root user"})
            else:
                qa.run_case("save-failure", qa.save_failure)
                qa.run_case("deletion-failure", qa.deletion_failure)
                qa.run_case("redundant-cleanup-failure", qa.redundant_cleanup_failure)
    except Exception as error:
        report["fatal_error"] = f"{type(error).__name__}: {error}"
    report["cleanup_verified"] = root is not None and not root.exists()
    report["total_wall_ms"] = round((time.perf_counter() - started) * 1000, 3)
    report["timings"] = {}
    for operation in ("add", "remove", "list"):
        for successful in (True, False):
            values = [r["ms"] for r in report["commands"]
                      if r["args"][0] == operation and (r["exit_code"] == 0) == successful]
            if values:
                report["timings"][operation + ("_success" if successful else "_error")] = distribution(values)
    report["serial_timings"] = {
        operation: distribution([r["ms"] for r in report["commands"]
                                 if r["case"] == "serial-timing" and r["args"][0] == operation])
        for operation in ("add", "remove", "list")
        if any(r["case"] == "serial-timing" and r["args"][0] == operation for r in report["commands"])
    }
    failed = report.get("fatal_error") or any(c["status"] == "failed" for c in report["cases"]) or not report["cleanup_verified"]
    report["status"] = "failed" if failed else "issues_found" if report["findings"] else "passed"
    args.report.write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({k: v for k, v in report.items() if k not in ("commands", "batches")}, indent=2))
    return 1 if failed else 2 if report["findings"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
