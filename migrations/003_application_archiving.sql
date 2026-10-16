ALTER TABLE applications ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
ALTER TABLE events ADD COLUMN IF NOT EXISTS action VARCHAR(20);
CREATE INDEX IF NOT EXISTS idx_applications_archived_page ON applications (user_id, archived_at DESC, id DESC) WHERE archived_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_applications_active_activity ON applications (updated_at, user_id) WHERE archived_at IS NULL;
UPDATE applications a SET updated_at = recent.last_event
FROM (SELECT application_id, MAX(created_at) AS last_event FROM events WHERE COALESCE(action, '') = '' GROUP BY application_id) recent
WHERE a.id = recent.application_id AND a.archived_at IS NULL AND a.updated_at < recent.last_event;
