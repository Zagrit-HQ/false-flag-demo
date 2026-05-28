# typescript-publish.hurl — exercises server-side TypeScript
# compilation via the source_text field added in slice 8.

# Setup: typescript-strategy project + flag.
POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-ts-publish","display_name":"TS Publish","config_strategy":"typescript"}
HTTP 201

POST http://localhost:8080/v1/projects/hurl-ts-publish/flags
Content-Type: application/json
X-Actor: hurl
{"key":"theme","name":"Theme","value_type":"string","default_value":"light"}
HTTP 201

# Case 1: PUT with source_text only. Server compiles via esbuild+goja,
# persists both source_text and the produced IR.
PUT http://localhost:8080/v1/projects/hurl-ts-publish/flags/theme
Content-Type: application/json
X-Actor: hurl
{
  "strategy": "typescript",
  "source": {},
  "source_text": "import { FalseFlag as ff } from \"@falseflag/config\";\n\nexport default ff.flag({\n  value_type: \"string\",\n  default: \"light\",\n  rules: [\n    ff.rule(\"dark-mode\", ff.eq(\"user.prefers\", \"dark\"), \"dark\"),\n  ],\n});\n"
}
HTTP 200
[Asserts]
jsonpath "$.version"                          == 1
jsonpath "$.strategy"                         == "typescript"
jsonpath "$.source_text"                      contains "ff.flag"
jsonpath "$.compiled.value_type"              == "string"
jsonpath "$.compiled.default"                 == "light"
jsonpath "$.compiled.rules[0].id"             == "dark-mode"
jsonpath "$.compiled.rules[0].when.kind"      == "eq"
jsonpath "$.compiled.rules[0].when.attr"      == "user.prefers"
jsonpath "$.compiled.rules[0].when.value"    == "dark"
jsonpath "$.compiled.rules[0].value"          == "dark"

# Case 2: PUT with malformed TS — esbuild emits a syntax error on
# line 1, the API returns 422 with structured details.
PUT http://localhost:8080/v1/projects/hurl-ts-publish/flags/theme
Content-Type: application/json
X-Actor: hurl
{
  "strategy": "typescript",
  "source": {},
  "source_text": "const x: = 1;\n"
}
HTTP 422
[Asserts]
jsonpath "$.message"             exists
jsonpath "$.details"             isCollection
jsonpath "$.details[0].text"     exists
jsonpath "$.details[0].line"     == 1

# Case 3: legacy publish path — strategy=json, no source_text. The
# server accepts the IR directly; response carries a null source_text.
# Verifies that older CLI binaries (pre-slice-8, no source_text field)
# keep working against the new server. Uses a separate flag to avoid
# colliding with the typescript-strategy version stream above.
POST http://localhost:8080/v1/projects/hurl-ts-publish/flags
Content-Type: application/json
X-Actor: hurl
{"key":"legacy","name":"Legacy","value_type":"string","default_value":"old"}
HTTP 201

PUT http://localhost:8080/v1/projects/hurl-ts-publish/flags/legacy
Content-Type: application/json
X-Actor: hurl
{
  "strategy": "json",
  "source": {"value_type":"string","default":"legacy","rules":[]}
}
HTTP 200
[Asserts]
jsonpath "$.version"             == 1
jsonpath "$.compiled.default"    == "legacy"
jsonpath "$.source_text"         not exists

# Case 4: both source and source_text supplied. The server uses
# source_text as the authoritative compile input; the supplied IR is
# discarded in favor of what the server compiles.
PUT http://localhost:8080/v1/projects/hurl-ts-publish/flags/theme
Content-Type: application/json
X-Actor: hurl
{
  "strategy": "typescript",
  "source": {"value_type":"string","default":"stale","rules":[]},
  "source_text": "import { FalseFlag as ff } from \"@falseflag/config\";\n\nexport default ff.flag({\n  value_type: \"string\",\n  default: \"server-wins\",\n  rules: [],\n});\n"
}
HTTP 200
[Asserts]
jsonpath "$.version"             == 2
jsonpath "$.compiled.default"    == "server-wins"
jsonpath "$.source_text"         contains "server-wins"
