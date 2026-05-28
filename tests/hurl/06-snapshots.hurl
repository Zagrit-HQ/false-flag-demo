# snapshots.hurl — empty + populated snapshot compilation.

POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-snaps","display_name":"Snapshots","config_strategy":"json"}
HTTP 201

# Empty project compiles to {flags: {}} at version 1.
POST http://localhost:8080/v1/projects/hurl-snaps/snapshots
Content-Type: application/json
X-Actor: hurl
HTTP 201
[Asserts]
jsonpath "$.version"  == 1
jsonpath "$.compiled" exists

# Add a flag + version, then recompile.
POST http://localhost:8080/v1/projects/hurl-snaps/flags
Content-Type: application/json
X-Actor: hurl
{"key":"feature-a","name":"Feature A","value_type":"boolean","default_value":false}
HTTP 201

PUT http://localhost:8080/v1/projects/hurl-snaps/flags/feature-a
Content-Type: application/json
X-Actor: hurl
{"strategy":"json","source":{"value_type":"boolean","default":false,"rules":[{"id":"always","when":{"kind":"always"},"value":true}]}}
HTTP 200

POST http://localhost:8080/v1/projects/hurl-snaps/snapshots
Content-Type: application/json
X-Actor: hurl
HTTP 201
[Asserts]
jsonpath "$.version"                                       == 2
jsonpath "$.compiled.flags['feature-a'].default"            == false
jsonpath "$.compiled.flags['feature-a'].rules[0].id"        == "always"

GET http://localhost:8080/v1/projects/hurl-snaps/snapshots/latest
HTTP 200
[Asserts]
jsonpath "$.version" == 2

GET http://localhost:8080/v1/projects/hurl-snaps/snapshots
HTTP 200
[Asserts]
jsonpath "$.items"           count == 2
jsonpath "$.items[0].version" == 2
jsonpath "$.items[1].version" == 1
