-- Correct the direction of the all-day event date migration.
-- Migration 019 moved affected all-day dates one day earlier. The observed import
-- offset was already one day early, so existing rows need to move two days later.
-- On a fresh database, migrations 019 and 020 together result in a net +1 day.
UPDATE events
SET
  starts_at = strftime('%Y-%m-%dT%H:%M:%SZ', starts_at, '+2 days'),
  ends_at = strftime('%Y-%m-%dT%H:%M:%SZ', ends_at, '+2 days'),
  updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE all_day = 1
  AND starts_at IS NOT NULL
  AND starts_at != ''
  AND ends_at IS NOT NULL
  AND ends_at != '';

UPDATE event_recurrence
SET until_at = strftime('%Y-%m-%dT%H:%M:%SZ', until_at, '+2 days')
WHERE until_at IS NOT NULL
  AND until_at != ''
  AND event_id IN (SELECT id FROM events WHERE all_day = 1);
