# User preferences

Account preferences are stored in `user_preferences`, with one row per user.
The normal API startup migration creates the table; the equivalent standalone
DDL is in `migrations/002_user_preferences.sql`.

## HTTP API

Both routes use the existing authentication middleware. The account comes from
the authenticated session or API key, never from the request body.

- `GET /v1/users/me/preferences`: returns the saved preferences, or defaults if
  this account has not saved any yet. A read does not create a row.
- `PATCH /v1/users/me/preferences`: saves only the supplied fields and returns
  the complete preferences. The first save creates the row. An empty patch reads
  the current preferences. An atomic upsert preserves omitted fields when
  multiple devices write concurrently; the latest write wins for the same field.

Response and example PATCH body:

```json
{
  "theme": "system",
  "confetti_enabled": true
}
```

`theme` accepts `light`, `dark`, or `system` (default). `confetti_enabled` accepts
a boolean and defaults to `true`; an explicit `false` is saved. Unknown fields,
invalid values/types, or malformed JSON return 400. Missing authentication returns
401; storage failures return 500. Timestamps are maintained internally.

## Frontend behavior

The profile Preferences section, header, and user menu share the Angular
`PreferencesService`. Changes save automatically. Saves are serialized, older
load responses cannot overwrite pending edits, and failed saves restore the
confirmed values with a retry action. Cache entries are scoped by user ID.
Preferences refresh after authentication and when the tab becomes visible;
a preferences failure does not terminate the session.

`system` means follow each device's own appearance setting. Confetti is a lazy
loaded native Canvas effect lasting about three seconds. It runs only after a
successful transition into `ACCEPTED` through Kanban or application editing,
respects reduced motion, and never intercepts pointer events. Loading, creating,
or editing an already accepted application does not celebrate. There is no live
push synchronization between devices.
