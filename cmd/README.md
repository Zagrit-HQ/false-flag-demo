# Go Binaries

This directory contains thin Go command entrypoints.

Keep business logic in `internal/**`; commands should parse configuration, initialize logging/observability, wire dependencies, and start the relevant service.
