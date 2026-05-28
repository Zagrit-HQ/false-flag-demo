# environments.hurl — environments lifecycle for a single project.

POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-envs","display_name":"Environments","config_strategy":"json"}
HTTP 201

POST http://localhost:8080/v1/projects/hurl-envs/environments
Content-Type: application/json
X-Actor: hurl
{"slug":"prod","name":"Production"}
HTTP 201
[Asserts]
jsonpath "$.slug" == "prod"
jsonpath "$.name" == "Production"

POST http://localhost:8080/v1/projects/hurl-envs/environments
Content-Type: application/json
X-Actor: hurl
{"slug":"staging","name":"Staging"}
HTTP 201

# Duplicate slug → 409.
POST http://localhost:8080/v1/projects/hurl-envs/environments
Content-Type: application/json
{"slug":"prod","name":"prod again"}
HTTP 409

GET http://localhost:8080/v1/projects/hurl-envs/environments
HTTP 200
[Asserts]
jsonpath "$.items"        count == 2
jsonpath "$.items[0].slug"     == "prod"

GET http://localhost:8080/v1/projects/hurl-envs/environments/prod
HTTP 200
[Asserts]
jsonpath "$.slug" == "prod"

GET http://localhost:8080/v1/projects/hurl-envs/environments/missing
HTTP 404
