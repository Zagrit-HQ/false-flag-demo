# falseflag-proxy

Evaluation proxy binary.

Expected responsibilities:

- serve flag evaluation requests
- expose OFREP-inspired remote evaluation endpoints
- cache compiled snapshots locally and through Redis
- stream updates to SDKs when that slice exists
- avoid executing user-authored TypeScript at runtime
