# Config Strategies

Project-scoped configuration strategy implementations.

Expected strategies:

- `json`: simple JSON configuration
- `cel`: CEL targeting rules
- `typescript`: TypeScript config-as-code compiled into static release snapshots

Only one strategy needs to be active per project. All strategies must compile into the same normalized release model.
