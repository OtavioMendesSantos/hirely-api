# Application archiving

Archiving preserves the application's status and sets `archived_at`. Restoring
clears that field, keeps the saved status, and resets `updated_at` so inactivity
starts again. Lifecycle events use `action: ARCHIVED` or `UNARCHIVED`; their
`createdAt` records when the operation happened. Archive operations preserve the
last activity timestamp and do not remove applications from historical metrics.

## HTTP API

All routes require the existing authentication and user isolation checks.

- `POST /v1/users/:user_id/applications/:application_id/archive`
- `POST /v1/users/:user_id/applications/:application_id/restore`

Both return the complete application, including `archivedAt`, events and tags.
Repeating the current state returns it without creating an event. Changes and
their events are committed together while locking the application row. Archived
applications remain readable and deletable; updates and new notes return 409
until the user restores them.

`GET /v1/users/:user_id/applications` accepts `archive_state=active|archived|all`
(default `active`), the existing `search`, `status`, `tag_ids`, `order_by`, and
`order` filters, and `page_size`/`page_token`. Search matches company or job title;
statuses are comma-separated. Archived reads default to 20 rows ordered by
`archived_at DESC`. Page sizes must be 1 through 100. Other reads remain unpaged
unless pagination is requested. The grouped Kanban endpoint only returns active
applications. There is no new date-range filter.

A page returns `applications` and `next_page_token` (empty at the end). Tokens
carry a keyset cursor and a fingerprint of user, filters, page size and sort.
Changing filters or sort requires starting with an empty token. Invalid tokens
return 400. Cursors use the requested sort value and ID as a stable tie-breaker;
nullable date fields keep explicit null ordering. The archive page uses
`order_by=archived_at` with ascending or descending direction.

The MCP `read_applications` tool accepts the same archive-state filter, returns
`archivedAt`, and preserves its existing unpaged output. MCP edits go through the
same archived-state checks as HTTP.

The startup migrations add the archive field, event action, archive/activity
indexes, and account for existing notes newer than `updated_at`. Equivalent DDL
is provided in `migrations/003_application_archiving.sql`.

## Verification

Run `go test ./...`. To also run isolated PostgreSQL integration tests, set
`HIRELY_TEST_DATABASE_URL` to a disposable database where the test user may create
schemas, then run the same command. Tests create a unique schema and remove it
afterwards; they do not use the application's `.env` file.
