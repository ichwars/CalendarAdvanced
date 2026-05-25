DELETE FROM event_attachments;
DELETE FROM event_reminders;
DELETE FROM event_attendees;
DELETE FROM event_recurrence;
DELETE FROM invitations;
DELETE FROM microsoft_sync_state;
DELETE FROM events;

DROP TABLE IF EXISTS microsoft_excel_sources;
DROP TABLE IF EXISTS microsoft_sync_state;
DROP TABLE IF EXISTS microsoft_token_store;
DROP TABLE IF EXISTS microsoft_accounts;

ALTER TABLE events DROP COLUMN source;
ALTER TABLE events DROP COLUMN external_id;
ALTER TABLE events DROP COLUMN external_url;
ALTER TABLE events DROP COLUMN microsoft_join_url;
ALTER TABLE excel_exports DROP COLUMN microsoft_file;
