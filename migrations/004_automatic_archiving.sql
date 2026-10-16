ALTER TABLE user_preferences ADD COLUMN IF NOT EXISTS auto_archive_days INTEGER NOT NULL DEFAULT 60;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'user_preferences_auto_archive_days_check' AND conrelid = 'user_preferences'::regclass) THEN
        ALTER TABLE user_preferences ADD CONSTRAINT user_preferences_auto_archive_days_check CHECK (auto_archive_days IN (0,30,60,90));
    END IF;
END $$;
