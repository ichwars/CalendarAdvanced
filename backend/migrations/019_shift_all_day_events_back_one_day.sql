-- Correct the v0.1.0 all-day event import/display offset.
-- All existing active events in affected installs are all-day values stored one calendar day too late.
UPDATE events
SET
  starts_at = strftime('%Y-%m-%dT%H:%M:%SZ', starts_at, '-1 day'),
  ends_at = strftime('%Y-%m-%dT%H:%M:%SZ', ends_at, '-1 day'),
  updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE all_day = 1
  AND starts_at IS NOT NULL
  AND starts_at != ''
  AND ends_at IS NOT NULL
  AND ends_at != '';

UPDATE event_recurrence
SET until_at = strftime('%Y-%m-%dT%H:%M:%SZ', until_at, '-1 day')
WHERE until_at IS NOT NULL
  AND until_at != ''
  AND event_id IN (SELECT id FROM events WHERE all_day = 1);
