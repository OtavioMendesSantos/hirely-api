# Interview Rate

`kpis.interviews.count` counts each application once if its current status is
INTERVIEW, OFFER, or ACCEPTED, or if a remaining event records entering or leaving
any of those stages. Moving to REJECTED, OTHER, or back to TO_APPLY does not erase
that history. Archived applications are included.

`kpis.interviews.rate` divides that count by the applications that have ever
advanced beyond TO_APPLY, using their current status or remaining status events.
This includes applications moved back to TO_APPLY, keeping the historical
numerator and denominator consistent. Applications that never left TO_APPLY do
not contribute to the denominator; an empty denominator produces zero.

The existing date range selects applications by creation date. Their remaining
history is considered regardless of when a transition occurred. Repeated visits
to an interview stage count once. Notes that mention interviews do not count as
transitions. Deleting the last supporting event removes historical evidence,
while a current INTERVIEW/OFFER/ACCEPTED status remains sufficient evidence.

The current-status funnel, rejection and ghosting calculations are unchanged.
There are no new HTTP fields or additional frontend requests for this fix.
