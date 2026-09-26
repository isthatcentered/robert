# Robert

Robert manages local checkouts of remote Git repositories through a saved configuration.

## Language

**Remote repository**:
A Git repository addressed by a URL. One remote repository can have multiple installations at different references.

**Configuration**:
Robert's saved set of installations. It is authoritative even when a checkout directory is missing.

**Installation**:
A saved association between a remote repository, a selected reference, and a local checkout directory. An installation exists while its configuration entry exists, even if the checkout directory is missing.
_Avoid_: Repository instance, clone

**Checkout directory**:
The local directory associated with an installation. A directory left behind after its configuration entry is removed is no longer an installation.
_Avoid_: Installation

**Reference**:
The branch, tag, or commit selected for an installation.
