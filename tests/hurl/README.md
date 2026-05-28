# Hurl API Tests

Hurl tests for HTTP API smoke and e2e validation.

Expected command shape:

```bash
hurl --test tests/hurl/*.hurl
```

These tests should start small with `/healthz`, then expand to project, flag, config validation, snapshot, evaluation, and audit paths.
