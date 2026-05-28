// @falseflag/generated-client re-exports the Orval-generated REST client
// and the matching Zod schemas. Slice 3 broadens the surface to every
// control-plane resource (projects, environments, segments, flags,
// snapshots, evaluation, audit). The dashboard and CLI consume these
// rather than calling fetch() themselves.
export * from "./generated/api.js";
export * as zod from "./generated/api.zod.js";
