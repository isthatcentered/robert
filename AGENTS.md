- Commit once a change is done and all checks pass for the whole project

## Design decisions
- Each command must return json
- Json errors should have this shape `{error: string, context?: any}` where context can be the repository causing issue or anything relevant to better understanding the issue
- All errors returned by the cli must provide enough context for the user to understand what the issue is and how to resolve it

## Guides
- [vertical slice](/home/isthatcentered/Test/robert2/.agents/guides/vertical-slice.md)
