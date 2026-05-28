# flags.hurl — exercises the flag lifecycle for JSON, CEL, and
# TypeScript strategies.

# Seed: project per strategy.
POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-flags-json","display_name":"JSON","config_strategy":"json"}
HTTP 201

POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-flags-cel","display_name":"CEL","config_strategy":"cel"}
HTTP 201

POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-flags-ts","display_name":"TypeScript","config_strategy":"typescript"}
HTTP 201

# ---- JSON strategy --------------------------------------------------
POST http://localhost:8080/v1/projects/hurl-flags-json/flags
Content-Type: application/json
{"key":"checkout","name":"Checkout","value_type":"boolean","default_value":false}
HTTP 201
[Asserts]
jsonpath "$.key"        == "checkout"
jsonpath "$.value_type" == "boolean"

PUT http://localhost:8080/v1/projects/hurl-flags-json/flags/checkout
Content-Type: application/json
{
  "strategy": "json",
  "source": {
    "value_type": "boolean",
    "default": false,
    "rules": [
      {"id":"r1","when":{"kind":"eq","attr":"user.plan","value":"pro"},"value":true}
    ]
  }
}
HTTP 200
[Asserts]
jsonpath "$.version"  == 1
jsonpath "$.strategy" == "json"

GET http://localhost:8080/v1/projects/hurl-flags-json/flags/checkout
HTTP 200
[Asserts]
jsonpath "$.flag.key"                == "checkout"
jsonpath "$.latest_version.version"  == 1
jsonpath "$.latest_version.strategy" == "json"

# Republish bumps version to 2.
PUT http://localhost:8080/v1/projects/hurl-flags-json/flags/checkout
Content-Type: application/json
{
  "strategy": "json",
  "source": {
    "value_type": "boolean",
    "default": true,
    "rules": []
  }
}
HTTP 200
[Asserts]
jsonpath "$.version" == 2

GET http://localhost:8080/v1/projects/hurl-flags-json/flags/checkout/versions
HTTP 200
[Asserts]
jsonpath "$.items"           count == 2
jsonpath "$.items[0].version" == 2
jsonpath "$.items[1].version" == 1

# Bad JSON source → 422 (invalid value_type; slice 8 phase 3 moved
# compile failures off the 400 bucket onto the 422 CompileError path).
PUT http://localhost:8080/v1/projects/hurl-flags-json/flags/checkout
Content-Type: application/json
{"strategy":"json","source":{"value_type":"date","default":false,"rules":[]}}
HTTP 422
[Asserts]
jsonpath "$.message" exists

# ---- CEL strategy ---------------------------------------------------
POST http://localhost:8080/v1/projects/hurl-flags-cel/flags
Content-Type: application/json
{"key":"premium","name":"Premium","value_type":"string","default_value":"basic"}
HTTP 201

PUT http://localhost:8080/v1/projects/hurl-flags-cel/flags/premium
Content-Type: application/json
{
  "strategy": "cel",
  "source": {
    "value_type": "string",
    "default": "basic",
    "rules": [
      {"id":"r1","when":{"kind":"cel","source":"ctx.user.age >= 21 && ctx.user.country == 'US'"},"value":"premium"}
    ]
  }
}
HTTP 200
[Asserts]
jsonpath "$.strategy" == "cel"

# Bad CEL → 422 (compile failure; see note above for line 79).
PUT http://localhost:8080/v1/projects/hurl-flags-cel/flags/premium
Content-Type: application/json
{
  "strategy": "cel",
  "source": {
    "value_type": "string",
    "default": "basic",
    "rules": [{"id":"r1","when":{"kind":"cel","source":"this is not cel ((("},"value":"premium"}]
  }
}
HTTP 422
[Asserts]
jsonpath "$.message" exists

# ---- TypeScript strategy (source_text, server-side compile) ---------
# Since slice 8 phase 3 the TS pipeline compiles source_text via
# esbuild + goja; the IR-as-JSON back-compat path is gone (the
# typescript_publish.hurl file covers the structured cases).
POST http://localhost:8080/v1/projects/hurl-flags-ts/flags
Content-Type: application/json
{"key":"theme","name":"Theme","value_type":"object","default_value":{"mode":"light"}}
HTTP 201

PUT http://localhost:8080/v1/projects/hurl-flags-ts/flags/theme
Content-Type: application/json
{
  "strategy": "typescript",
  "source": {},
  "source_text": "import { FalseFlag as ff } from \"@falseflag/config\";\n\nexport default ff.flag({\n  value_type: \"object\",\n  default: {mode: \"light\"},\n  rules: [\n    ff.rule(\"r-dark\", ff.eq(\"user.prefers\", \"dark\"), {mode: \"dark\"}),\n  ],\n});\n"
}
HTTP 200
[Asserts]
jsonpath "$.strategy" == "typescript"
