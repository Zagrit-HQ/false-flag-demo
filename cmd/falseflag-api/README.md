# falseflag-api

Control-plane API binary.

Expected responsibilities:

- serve REST/OpenAPI endpoints
- serve ConnectRPC or gRPC admin/internal APIs
- own project, environment, flag, segment, config, snapshot, and audit workflows
- use `internal/db` for SQLC-backed persistence
- expose health and metrics endpoints
