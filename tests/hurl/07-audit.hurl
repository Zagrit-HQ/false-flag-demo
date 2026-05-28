# audit.hurl — audit search reads from prior Hurl files' writes.

# Default-paged list: every audit event ever, newest first.
GET http://localhost:8080/v1/projects/hurl-snaps/audit-events
HTTP 200
[Asserts]
# Multiple events are present (compile_snapshot ran twice in 06, plus
# the create_flag + publish_version emitted there).
jsonpath "$.items" count >= 3

# Filter by action.
GET http://localhost:8080/v1/projects/hurl-snaps/audit-events?action=publish_version
HTTP 200
[Asserts]
jsonpath "$.items"           count == 1
jsonpath "$.items[0].action"    == "publish_version"
jsonpath "$.items[0].actor"     == "hurl"

# Filter by actor.
GET http://localhost:8080/v1/projects/hurl-snaps/audit-events?actor=hurl
HTTP 200
[Asserts]
jsonpath "$.items"           count >= 1
jsonpath "$.items[0].actor"     == "hurl"

# Filter that matches nothing.
GET http://localhost:8080/v1/projects/hurl-snaps/audit-events?action=does_not_exist
HTTP 200
[Asserts]
jsonpath "$.items" count == 0

# Tight limit + cursor round-trip: limit=1 returns 1 item plus next_cursor.
GET http://localhost:8080/v1/projects/hurl-snaps/audit-events?limit=1
HTTP 200
[Asserts]
jsonpath "$.items"      count == 1
jsonpath "$.next_cursor" exists
