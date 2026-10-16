# Application list and metric drill-downs

`GET /users/:user_id/applications` keeps its existing search, status, tag, sorting,
archive-state and cursor filters. New optional parameters:

- `start_date`, `end_date`: filter application **creation** time, as in metrics.
  Accept `YYYY-MM-DD` (inclusive UTC calendar days) or RFC3339 timestamps.
  Malformed dates and inverted ranges return 400.
- `metric`: `interviews`, `rejections`, `direct_rejections`,
  `advanced_rejections`, or `ghosting`. Unknown values return 400.

Metric rules are shared with the statistics queries:

- Interviews: current INTERVIEW/OFFER/ACCEPTED or a recorded transition from/to
  one of these statuses. Each application appears once, including reverted,
  rejected and archived applications. Deleting the final historical evidence can
  remove membership when the current status no longer proves it.
- Rejections: current REJECTED. Advanced rejections additionally have a recorded
  transition from/to INTERVIEW/OFFER; direct rejections have no such transition.
- Ghosting: current APPLIED with `updated_at` at least 30 days old. Count and page
  filters use the same evaluation time within a request.

Current-status filters combine with metric filters using AND. A status-only query
always uses the application's current status, never its transition history.
Use `archive_state=all` for metric links because statistics include archives.
All queries remain scoped to the authenticated owner.

Responses add `total_count`, counting the entire filtered result before the
cursor/limit, alongside `applications` and `next_page_token`. Use `page_size=20`
for the Applications page. Existing unpaged clients remain supported.
Cursors are bound to the dates, metric, owner and other normalized filters;
changing any filter requires starting at the first page.

No schema migration or background worker is needed for these read filters.
