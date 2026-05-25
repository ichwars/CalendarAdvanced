package sqlite

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"calendaradvanced/internal/domain"
)

type Store struct {
	DB *DB
}

type EventFilter struct {
	UserID           int64
	From             time.Time
	To               time.Time
	CalendarID       int64
	Query            string
	Limit            int
	Offset           int
	IncludeRecurring bool
}

type TaskFilter struct {
	UserID    int64
	Query     string
	Completed *bool
	Limit     int
	Offset    int
}

type ContactFilter struct {
	UserID int64
	Query  string
	Limit  int
	Offset int
}

func OpenStore(dataDir, migrationsDir string) (*Store, error) {
	db, err := Open(filepath.Join(dataDir, "calendaradvanced.db"))
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(migrationsDir); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) UserCount() (int, error) {
	rows, err := s.DB.Query(`SELECT COUNT(*) AS count FROM users`)
	if err != nil || len(rows) == 0 {
		return 0, err
	}
	return Int(rows[0]["count"]), nil
}

func (s *Store) SetupRequired() (bool, error) {
	count, err := s.UserCount()
	return count == 0, err
}

func (s *Store) CreateUser(email, username, displayName, passwordHash string, active bool, roles []domain.RoleName) (domain.User, error) {
	now := time.Now().UTC()
	if err := s.DB.Exec(`INSERT INTO users(email, username, display_name, password_hash, active, created_at, updated_at) VALUES(?,?,?,?,?,?,?)`, domain.NormalizeEmail(email), domain.NormalizeUsername(username), strings.TrimSpace(displayName), passwordHash, active, now, now); err != nil {
		return domain.User{}, err
	}
	id := s.DB.LastInsertID()
	for _, role := range roles {
		if err := s.DB.Exec(`INSERT OR IGNORE INTO user_roles(user_id, role_id) SELECT ?, id FROM roles WHERE name = ?`, id, string(role)); err != nil {
			return domain.User{}, err
		}
	}
	return s.FindUserByID(id)
}

func (s *Store) FindUserByEmail(email string) (domain.User, error) {
	rows, err := s.DB.Query(`SELECT u.*, COALESCE(t.enabled,0) AS two_factor_enabled FROM users u LEFT JOIN two_factor_secrets t ON t.user_id = u.id WHERE u.email = ?`, domain.NormalizeEmail(email))
	if err != nil {
		return domain.User{}, err
	}
	if len(rows) == 0 {
		return domain.User{}, ErrNotFound
	}
	return s.userFromRow(rows[0])
}

func (s *Store) FindUserByLogin(identifier string) (domain.User, error) {
	normalized := domain.NormalizeUsername(identifier)
	rows, err := s.DB.Query(`SELECT u.*, COALESCE(t.enabled,0) AS two_factor_enabled FROM users u LEFT JOIN two_factor_secrets t ON t.user_id = u.id WHERE u.email = ? OR u.username = ?`, domain.NormalizeEmail(identifier), normalized)
	if err != nil {
		return domain.User{}, err
	}
	if len(rows) == 0 {
		return domain.User{}, ErrNotFound
	}
	return s.userFromRow(rows[0])
}

func (s *Store) FindUserByID(id int64) (domain.User, error) {
	rows, err := s.DB.Query(`SELECT u.*, COALESCE(t.enabled,0) AS two_factor_enabled FROM users u LEFT JOIN two_factor_secrets t ON t.user_id = u.id WHERE u.id = ?`, id)
	if err != nil {
		return domain.User{}, err
	}
	if len(rows) == 0 {
		return domain.User{}, ErrNotFound
	}
	return s.userFromRow(rows[0])
}

func (s *Store) ListUsers() ([]domain.User, error) {
	rows, err := s.DB.Query(`SELECT u.*, COALESCE(t.enabled,0) AS two_factor_enabled FROM users u LEFT JOIN two_factor_secrets t ON t.user_id = u.id ORDER BY u.email`)
	if err != nil {
		return nil, err
	}
	users := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		user, err := s.userFromRow(row)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *Store) userFromRow(row Row) (domain.User, error) {
	id := Int64(row["id"])
	roleRows, err := s.DB.Query(`SELECT r.name FROM roles r JOIN user_roles ur ON ur.role_id = r.id WHERE ur.user_id = ? ORDER BY r.name`, id)
	if err != nil {
		return domain.User{}, err
	}
	roles := make([]domain.RoleName, 0, len(roleRows))
	for _, rr := range roleRows {
		roles = append(roles, domain.RoleName(rr["name"]))
	}
	return domain.User{
		ID:               id,
		Email:            row["email"],
		Username:         row["username"],
		DisplayName:      row["display_name"],
		PasswordHash:     row["password_hash"],
		Active:           Bool(row["active"]),
		Roles:            roles,
		TwoFactorEnabled: Bool(row["two_factor_enabled"]),
		CreatedAt:        ParseTime(row["created_at"]),
		UpdatedAt:        ParseTime(row["updated_at"]),
	}, nil
}

func (s *Store) SetUserRoles(userID int64, roles []domain.RoleName) error {
	if err := s.DB.Exec(`DELETE FROM user_roles WHERE user_id = ?`, userID); err != nil {
		return err
	}
	for _, role := range roles {
		if err := s.DB.Exec(`INSERT OR IGNORE INTO user_roles(user_id, role_id) SELECT ?, id FROM roles WHERE name = ?`, userID, string(role)); err != nil {
			return err
		}
	}
	return s.DB.Exec(`UPDATE users SET updated_at = ? WHERE id = ?`, time.Now().UTC(), userID)
}

func (s *Store) UpdateUserPassword(userID int64, passwordHash string) error {
	if err := s.DB.Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, passwordHash, time.Now().UTC(), userID); err != nil {
		return err
	}
	return s.RevokeUserSessions(userID)
}

func (s *Store) GetUserPreferences(userID int64) (domain.GeneralPreferences, error) {
	rows, err := s.DB.Query(`SELECT preferences_json FROM user_preferences WHERE user_id = ?`, userID)
	if err != nil {
		return domain.GeneralPreferences{}, err
	}
	if len(rows) == 0 {
		return domain.GeneralPreferences{}, ErrNotFound
	}
	var preferences domain.GeneralPreferences
	if err := json.Unmarshal([]byte(rows[0]["preferences_json"]), &preferences); err != nil {
		return domain.GeneralPreferences{}, err
	}
	return preferences, nil
}

func (s *Store) UpsertUserPreferences(userID int64, preferences domain.GeneralPreferences) error {
	body, err := json.Marshal(preferences)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return s.DB.Exec(`INSERT INTO user_preferences(user_id, preferences_json, created_at, updated_at) VALUES(?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET preferences_json = excluded.preferences_json, updated_at = excluded.updated_at`, userID, string(body), now, now)
}

func (s *Store) CreateSession(userID int64, tokenHash, csrfHash string, expiresAt time.Time, userAgent, ip string) error {
	return s.DB.Exec(`INSERT INTO sessions(user_id, token_hash, csrf_hash, expires_at, user_agent, ip, created_at) VALUES(?,?,?,?,?,?,?)`, userID, tokenHash, csrfHash, expiresAt, sanitizeLogField(userAgent, 300), sanitizeLogField(ip, 80), time.Now().UTC())
}

func (s *Store) FindSession(tokenHash string) (domain.Session, domain.User, error) {
	rows, err := s.DB.Query(`SELECT * FROM sessions WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > ?`, tokenHash, time.Now().UTC())
	if err != nil {
		return domain.Session{}, domain.User{}, err
	}
	if len(rows) == 0 {
		return domain.Session{}, domain.User{}, ErrNotFound
	}
	row := rows[0]
	session := domain.Session{ID: Int64(row["id"]), UserID: Int64(row["user_id"]), TokenHash: row["token_hash"], CSRFHash: row["csrf_hash"], ExpiresAt: ParseTime(row["expires_at"]), CreatedAt: ParseTime(row["created_at"]), UserAgent: row["user_agent"], IP: row["ip"]}
	user, err := s.FindUserByID(session.UserID)
	return session, user, err
}

func (s *Store) RevokeSession(tokenHash string) error {
	return s.DB.Exec(`UPDATE sessions SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`, time.Now().UTC(), tokenHash)
}

func (s *Store) RevokeUserSessions(userID int64) error {
	return s.DB.Exec(`UPDATE sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, time.Now().UTC(), userID)
}

func (s *Store) InsertPasswordResetToken(userID int64, tokenHash string, expiresAt time.Time) error {
	return s.DB.Exec(`INSERT INTO password_reset_tokens(user_id, token_hash, expires_at, created_at) VALUES(?,?,?,?)`, userID, tokenHash, expiresAt, time.Now().UTC())
}

func (s *Store) FindPasswordResetToken(tokenHash string) (int64, error) {
	rows, err := s.DB.Query(`SELECT user_id FROM password_reset_tokens WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`, tokenHash, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, ErrNotFound
	}
	return Int64(rows[0]["user_id"]), nil
}

func (s *Store) MarkPasswordResetUsed(tokenHash string) error {
	return s.DB.Exec(`UPDATE password_reset_tokens SET used_at = ? WHERE token_hash = ?`, time.Now().UTC(), tokenHash)
}

func (s *Store) UpsertTwoFactorSecret(userID int64, secret string, enabled bool) error {
	now := time.Now().UTC()
	return s.DB.Exec(`INSERT INTO two_factor_secrets(user_id, secret_encrypted, enabled, created_at, updated_at) VALUES(?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET secret_encrypted = excluded.secret_encrypted, enabled = excluded.enabled, updated_at = excluded.updated_at`, userID, secret, enabled, now, now)
}

func (s *Store) GetTwoFactorSecret(userID int64) (secret string, enabled bool, err error) {
	rows, err := s.DB.Query(`SELECT secret_encrypted, enabled FROM two_factor_secrets WHERE user_id = ?`, userID)
	if err != nil {
		return "", false, err
	}
	if len(rows) == 0 {
		return "", false, ErrNotFound
	}
	return rows[0]["secret_encrypted"], Bool(rows[0]["enabled"]), nil
}

func (s *Store) SetTwoFactorEnabled(userID int64, enabled bool) error {
	return s.DB.Exec(`UPDATE two_factor_secrets SET enabled = ?, updated_at = ? WHERE user_id = ?`, enabled, time.Now().UTC(), userID)
}

func (s *Store) ReplaceBackupCodes(userID int64, hashes []string) error {
	if err := s.DB.Exec(`DELETE FROM two_factor_backup_codes WHERE user_id = ?`, userID); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, hash := range hashes {
		if err := s.DB.Exec(`INSERT INTO two_factor_backup_codes(user_id, code_hash, created_at) VALUES(?,?,?)`, userID, hash, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UseBackupCode(userID int64, hash string) (bool, error) {
	rows, err := s.DB.Query(`SELECT id FROM two_factor_backup_codes WHERE user_id = ? AND code_hash = ? AND used_at IS NULL`, userID, hash)
	if err != nil || len(rows) == 0 {
		return false, err
	}
	if err := s.DB.Exec(`UPDATE two_factor_backup_codes SET used_at = ? WHERE id = ?`, time.Now().UTC(), Int64(rows[0]["id"])); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) Audit(actorID int64, action domain.AuditAction, entityType, entityID, ip, userAgent string, metadata any) error {
	var meta string
	if metadata != nil {
		encoded, _ := json.Marshal(metadata)
		meta = string(encoded)
	}
	return s.DB.Exec(`INSERT INTO audit_log(actor_id, action, entity_type, entity_id, ip, user_agent, metadata, created_at) VALUES(?,?,?,?,?,?,?,?)`, nullZero(actorID), string(action), entityType, entityID, sanitizeLogField(ip, 80), sanitizeLogField(userAgent, 300), meta, time.Now().UTC())
}

func (s *Store) ListAudit(limit, offset int) ([]domain.AuditEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.DB.Query(`SELECT * FROM audit_log ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AuditEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AuditEntry{ID: Int64(row["id"]), ActorID: Int64(row["actor_id"]), Action: domain.AuditAction(row["action"]), EntityType: row["entity_type"], EntityID: row["entity_id"], IP: row["ip"], UserAgent: row["user_agent"], Metadata: row["metadata"], CreatedAt: ParseTime(row["created_at"])})
	}
	return out, nil
}

func (s *Store) ListDAVSyncAudit(userID int64, limit int) ([]domain.AuditEntry, error) {
	if limit <= 0 || limit > 50 {
		limit = 8
	}
	rows, err := s.DB.Query(`SELECT * FROM audit_log WHERE actor_id = ? AND action = ? AND entity_type = ? ORDER BY id DESC LIMIT ?`, userID, string(domain.AuditIntegrationChanged), "dav_sync", limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AuditEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AuditEntry{ID: Int64(row["id"]), ActorID: Int64(row["actor_id"]), Action: domain.AuditAction(row["action"]), EntityType: row["entity_type"], EntityID: row["entity_id"], IP: row["ip"], UserAgent: row["user_agent"], Metadata: row["metadata"], CreatedAt: ParseTime(row["created_at"])})
	}
	return out, nil
}

func (s *Store) CreateCalendar(calendar domain.Calendar) (domain.Calendar, error) {
	now := time.Now().UTC()
	if calendar.Color == "" {
		calendar.Color = "#6d8cff"
	}
	if calendar.Timezone == "" {
		calendar.Timezone = "UTC"
	}
	if calendar.ReminderTime == "" {
		calendar.ReminderTime = "09:00"
	}
	if calendar.SameDayReminderTime == "" {
		calendar.SameDayReminderTime = "09:00"
	}
	if err := s.DB.Exec(`INSERT INTO calendars(owner_user_id, name, description, color, timezone, visible, reminder_enabled, reminder_days_before, reminder_time, same_day_reminder_time, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, calendar.OwnerUserID, strings.TrimSpace(calendar.Name), calendar.Description, calendar.Color, calendar.Timezone, true, calendar.ReminderEnabled, calendar.ReminderDaysBefore, calendar.ReminderTime, calendar.SameDayReminderTime, now, now); err != nil {
		return domain.Calendar{}, err
	}
	return s.FindCalendarByID(s.DB.LastInsertID(), calendar.OwnerUserID)
}

func (s *Store) FindCalendarByID(id, userID int64) (domain.Calendar, error) {
	rows, err := s.DB.Query(`SELECT c.* FROM calendars c WHERE c.id = ? AND (c.owner_user_id = ? OR EXISTS(SELECT 1 FROM calendar_shares cs WHERE cs.calendar_id = c.id AND cs.user_id = ?))`, id, userID, userID)
	if err != nil {
		return domain.Calendar{}, err
	}
	if len(rows) == 0 {
		return domain.Calendar{}, ErrNotFound
	}
	return calendarFromRow(rows[0]), nil
}

func (s *Store) ListCalendars(userID int64) ([]domain.Calendar, error) {
	rows, err := s.DB.Query(`SELECT DISTINCT c.* FROM calendars c LEFT JOIN calendar_shares cs ON cs.calendar_id = c.id WHERE c.owner_user_id = ? OR cs.user_id = ? ORDER BY c.name`, userID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Calendar, 0, len(rows))
	for _, row := range rows {
		out = append(out, calendarFromRow(row))
	}
	return out, nil
}

func (s *Store) UpdateCalendar(calendar domain.Calendar, userID int64) (domain.Calendar, error) {
	if _, err := s.FindCalendarByID(calendar.ID, userID); err != nil {
		return domain.Calendar{}, err
	}
	if calendar.ReminderTime == "" {
		calendar.ReminderTime = "09:00"
	}
	if calendar.SameDayReminderTime == "" {
		calendar.SameDayReminderTime = "09:00"
	}
	if err := s.DB.Exec(`UPDATE calendars SET name = ?, description = ?, color = ?, timezone = ?, visible = ?, reminder_enabled = ?, reminder_days_before = ?, reminder_time = ?, same_day_reminder_time = ?, updated_at = ? WHERE id = ?`, strings.TrimSpace(calendar.Name), calendar.Description, calendar.Color, calendar.Timezone, calendar.Visible, calendar.ReminderEnabled, calendar.ReminderDaysBefore, calendar.ReminderTime, calendar.SameDayReminderTime, time.Now().UTC(), calendar.ID); err != nil {
		return domain.Calendar{}, err
	}
	return s.FindCalendarByID(calendar.ID, userID)
}

func (s *Store) DeleteCalendar(id, userID int64) error {
	return s.DB.Exec(`UPDATE calendars SET deleted_at = ?, visible = 0 WHERE id = ? AND owner_user_id = ?`, time.Now().UTC(), id, userID)
}

func calendarFromRow(row Row) domain.Calendar {
	return domain.Calendar{ID: Int64(row["id"]), OwnerUserID: Int64(row["owner_user_id"]), Name: row["name"], Description: row["description"], Color: row["color"], Timezone: row["timezone"], Visible: Bool(row["visible"]), ReminderEnabled: Bool(row["reminder_enabled"]), ReminderDaysBefore: Int(row["reminder_days_before"]), ReminderTime: valueOrDefault(row["reminder_time"], "09:00"), SameDayReminderTime: valueOrDefault(row["same_day_reminder_time"], "09:00"), CreatedAt: ParseTime(row["created_at"]), UpdatedAt: ParseTime(row["updated_at"])}
}

func (s *Store) CreateEvent(event domain.Event) (domain.Event, error) {
	now := time.Now().UTC()
	if event.UID == "" {
		event.UID = fmt.Sprintf("ck-%d-%d@calendaradvanced", event.CalendarID, now.UnixNano())
	}
	if event.Status == "" {
		event.Status = domain.EventStatusConfirmed
	}
	event.ETag = fmt.Sprintf(`"%x"`, now.UnixNano())
	if err := s.DB.Exec(`INSERT INTO events(calendar_id, uid, title, description, location, starts_at, ends_at, timezone, all_day, private, completed, birthday_year, status, etag, created_by, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, event.CalendarID, event.UID, strings.TrimSpace(event.Title), event.Description, event.Location, event.StartsAt, event.EndsAt, event.Timezone, event.AllDay, event.Private, event.Completed, event.BirthdayYear, string(event.Status), event.ETag, event.CreatedBy, now, now); err != nil {
		return domain.Event{}, err
	}
	id := s.DB.LastInsertID()
	if event.Recurrence != nil && event.Recurrence.Frequency != "" {
		r := *event.Recurrence
		r.EventID = id
		r.RRule = domain.RRULE(r)
		if err := s.DB.Exec(`INSERT INTO event_recurrence(event_id, frequency, interval, count, until_at, by_day, rrule, created_at) VALUES(?,?,?,?,?,?,?,?)`, r.EventID, string(r.Frequency), r.Interval, r.Count, r.Until, r.ByDay, r.RRule, now); err != nil {
			return domain.Event{}, err
		}
	}
	for _, attendee := range event.Attendees {
		if attendee.Status == "" {
			attendee.Status = domain.AttendeeNeedsAction
		}
		_ = s.DB.Exec(`INSERT INTO event_attendees(event_id, email, display_name, status, created_at) VALUES(?,?,?,?,?)`, id, domain.NormalizeEmail(attendee.Email), attendee.DisplayName, string(attendee.Status), now)
	}
	for _, reminder := range event.Reminders {
		_ = s.DB.Exec(`INSERT INTO event_reminders(event_id, minutes_before, created_at) VALUES(?,?,?)`, id, reminder.MinutesBefore, now)
	}
	return s.FindEventByID(id, event.CreatedBy)
}

func (s *Store) FindEventByID(id, userID int64) (domain.Event, error) {
	rows, err := s.DB.Query(`SELECT e.* FROM events e JOIN calendars c ON c.id = e.calendar_id LEFT JOIN calendar_shares cs ON cs.calendar_id = c.id WHERE e.id = ? AND e.deleted_at IS NULL AND (c.owner_user_id = ? OR cs.user_id = ?)`, id, userID, userID)
	if err != nil {
		return domain.Event{}, err
	}
	if len(rows) == 0 {
		return domain.Event{}, ErrNotFound
	}
	return s.eventFromRow(rows[0])
}

func (s *Store) FindEventByUID(uid string, userID int64) (domain.Event, error) {
	rows, err := s.DB.Query(`SELECT e.* FROM events e JOIN calendars c ON c.id = e.calendar_id LEFT JOIN calendar_shares cs ON cs.calendar_id = c.id WHERE e.uid = ? AND e.deleted_at IS NULL AND (c.owner_user_id = ? OR cs.user_id = ?)`, uid, userID, userID)
	if err != nil {
		return domain.Event{}, err
	}
	if len(rows) == 0 {
		return domain.Event{}, ErrNotFound
	}
	return s.eventFromRow(rows[0])
}

func (s *Store) FindEventByUIDIncludingDeleted(uid string, userID int64) (domain.Event, error) {
	rows, err := s.DB.Query(`SELECT e.* FROM events e JOIN calendars c ON c.id = e.calendar_id LEFT JOIN calendar_shares cs ON cs.calendar_id = c.id WHERE e.uid = ? AND (c.owner_user_id = ? OR cs.user_id = ?)`, uid, userID, userID)
	if err != nil {
		return domain.Event{}, err
	}
	if len(rows) == 0 {
		return domain.Event{}, ErrNotFound
	}
	return s.eventFromRow(rows[0])
}

func (s *Store) ListEvents(filter EventFilter) ([]domain.Event, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query := `SELECT DISTINCT e.* FROM events e JOIN calendars c ON c.id = e.calendar_id LEFT JOIN calendar_shares cs ON cs.calendar_id = c.id LEFT JOIN event_attendees ea ON ea.event_id = e.id`
	if filter.IncludeRecurring {
		query += ` LEFT JOIN event_recurrence er ON er.event_id = e.id`
	}
	query += ` WHERE e.deleted_at IS NULL AND (c.owner_user_id = ? OR cs.user_id = ?)`
	args := []any{filter.UserID, filter.UserID}
	if !filter.From.IsZero() {
		if filter.IncludeRecurring {
			query += ` AND (e.ends_at >= ? OR er.event_id IS NOT NULL)`
		} else {
			query += ` AND e.ends_at >= ?`
		}
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		query += ` AND e.starts_at <= ?`
		args = append(args, filter.To)
	}
	if filter.CalendarID > 0 {
		query += ` AND e.calendar_id = ?`
		args = append(args, filter.CalendarID)
	}
	if filter.Query != "" {
		like := "%" + strings.ToLower(filter.Query) + "%"
		query += ` AND (lower(e.title) LIKE ? OR lower(e.description) LIKE ? OR lower(e.location) LIKE ? OR lower(ea.email) LIKE ?)`
		args = append(args, like, like, like, like)
	}
	query += ` ORDER BY e.starts_at ASC LIMIT ? OFFSET ?`
	args = append(args, limit, filter.Offset)
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	events := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		event, err := s.eventFromRow(row)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (s *Store) UpdateEvent(event domain.Event, userID int64) (domain.Event, error) {
	if _, err := s.FindEventByID(event.ID, userID); err != nil {
		return domain.Event{}, err
	}
	now := time.Now().UTC()
	event.ETag = fmt.Sprintf(`"%x"`, now.UnixNano())
	if err := s.DB.Exec(`UPDATE events SET title = ?, description = ?, location = ?, starts_at = ?, ends_at = ?, timezone = ?, all_day = ?, private = ?, completed = ?, birthday_year = ?, status = ?, etag = ?, updated_at = ? WHERE id = ?`, strings.TrimSpace(event.Title), event.Description, event.Location, event.StartsAt, event.EndsAt, event.Timezone, event.AllDay, event.Private, event.Completed, event.BirthdayYear, string(event.Status), event.ETag, now, event.ID); err != nil {
		return domain.Event{}, err
	}
	_ = s.DB.Exec(`DELETE FROM event_recurrence WHERE event_id = ?`, event.ID)
	if event.Recurrence != nil && event.Recurrence.Frequency != "" {
		r := *event.Recurrence
		r.EventID = event.ID
		r.RRule = domain.RRULE(r)
		if err := s.DB.Exec(`INSERT INTO event_recurrence(event_id, frequency, interval, count, until_at, by_day, rrule, created_at) VALUES(?,?,?,?,?,?,?,?)`, r.EventID, string(r.Frequency), r.Interval, r.Count, r.Until, r.ByDay, r.RRule, now); err != nil {
			return domain.Event{}, err
		}
	}
	return s.FindEventByID(event.ID, userID)
}

func (s *Store) RestoreEvent(event domain.Event, userID int64) (domain.Event, error) {
	rows, err := s.DB.Query(`SELECT e.id FROM events e JOIN calendars c ON c.id = e.calendar_id LEFT JOIN calendar_shares cs ON cs.calendar_id = c.id WHERE e.id = ? AND (c.owner_user_id = ? OR cs.user_id = ?)`, event.ID, userID, userID)
	if err != nil {
		return domain.Event{}, err
	}
	if len(rows) == 0 {
		return domain.Event{}, ErrNotFound
	}
	now := time.Now().UTC()
	event.ETag = fmt.Sprintf(`"%x"`, now.UnixNano())
	if err := s.DB.Exec(`UPDATE events SET calendar_id = ?, title = ?, description = ?, location = ?, starts_at = ?, ends_at = ?, timezone = ?, all_day = ?, private = ?, completed = ?, birthday_year = ?, status = ?, etag = ?, deleted_at = NULL, updated_at = ? WHERE id = ?`, event.CalendarID, strings.TrimSpace(event.Title), event.Description, event.Location, event.StartsAt, event.EndsAt, event.Timezone, event.AllDay, event.Private, event.Completed, event.BirthdayYear, string(event.Status), event.ETag, now, event.ID); err != nil {
		return domain.Event{}, err
	}
	_ = s.DB.Exec(`DELETE FROM event_recurrence WHERE event_id = ?`, event.ID)
	if event.Recurrence != nil && event.Recurrence.Frequency != "" {
		r := *event.Recurrence
		r.EventID = event.ID
		r.RRule = domain.RRULE(r)
		if err := s.DB.Exec(`INSERT INTO event_recurrence(event_id, frequency, interval, count, until_at, by_day, rrule, created_at) VALUES(?,?,?,?,?,?,?,?)`, r.EventID, string(r.Frequency), r.Interval, r.Count, r.Until, r.ByDay, r.RRule, now); err != nil {
			return domain.Event{}, err
		}
	}
	return s.FindEventByID(event.ID, userID)
}

func (s *Store) DeleteEvent(id, userID int64) error {
	if _, err := s.FindEventByID(id, userID); err != nil {
		return err
	}
	return s.DB.Exec(`UPDATE events SET deleted_at = ?, updated_at = ? WHERE id = ?`, time.Now().UTC(), time.Now().UTC(), id)
}

func (s *Store) ListDueReminders(userID int64, now time.Time, limit int) ([]domain.DueReminder, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.DB.Query(`
		SELECT DISTINCT er.id AS reminder_id, er.event_id, er.minutes_before, e.title, e.starts_at, c.name AS calendar_name
		FROM event_reminders er
		JOIN events e ON e.id = er.event_id
		JOIN calendars c ON c.id = e.calendar_id
		LEFT JOIN calendar_shares cs ON cs.calendar_id = c.id
		WHERE er.delivered_at IS NULL
			AND e.deleted_at IS NULL
			AND e.starts_at >= ?
			AND e.starts_at <= ?
			AND (c.owner_user_id = ? OR cs.user_id = ?)
		ORDER BY e.starts_at ASC
		LIMIT ?`, now.Add(-24*time.Hour), now.AddDate(0, 0, 370), userID, userID, limit*4)
	if err != nil {
		return nil, err
	}
	out := make([]domain.DueReminder, 0, len(rows))
	for _, row := range rows {
		startsAt := ParseTime(row["starts_at"])
		minutesBefore := Int(row["minutes_before"])
		dueAt := startsAt.Add(-time.Duration(minutesBefore) * time.Minute)
		if dueAt.After(now) {
			continue
		}
		out = append(out, domain.DueReminder{
			ID:            Int64(row["reminder_id"]),
			EventID:       Int64(row["event_id"]),
			Kind:          "event",
			Title:         row["title"],
			CalendarName:  row["calendar_name"],
			StartsAt:      startsAt,
			MinutesBefore: minutesBefore,
			DueAt:         dueAt,
		})
		if len(out) >= limit {
			break
		}
	}
	taskRows, err := s.DB.Query(`
		SELECT id, title, due_at, reminder_at
		FROM tasks
		WHERE user_id = ?
			AND completed = 0
			AND reminder_at IS NOT NULL
			AND reminder_at != ''
			AND reminder_delivered_at IS NULL
			AND reminder_at <= ?
		ORDER BY reminder_at ASC
		LIMIT ?`, userID, now, limit)
	if err != nil {
		return nil, err
	}
	for _, row := range taskRows {
		reminderAt := ParseTime(row["reminder_at"])
		startsAt := ParseTime(row["due_at"])
		if startsAt.IsZero() {
			startsAt = reminderAt
		}
		out = append(out, domain.DueReminder{
			ID:           Int64(row["id"]),
			TaskID:       Int64(row["id"]),
			Kind:         "task",
			Title:        row["title"],
			CalendarName: "Tasks",
			StartsAt:     startsAt,
			DueAt:        reminderAt,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Store) MarkReminderDelivered(id, userID int64, deliveredAt time.Time) error {
	return s.DB.Exec(`
		UPDATE event_reminders
		SET delivered_at = ?
		WHERE id = ?
			AND event_id IN (
				SELECT e.id
				FROM events e
				JOIN calendars c ON c.id = e.calendar_id
				LEFT JOIN calendar_shares cs ON cs.calendar_id = c.id
				WHERE e.deleted_at IS NULL
					AND (c.owner_user_id = ? OR cs.user_id = ?)
			)`, deliveredAt, id, userID, userID)
}

func (s *Store) MarkTaskReminderDelivered(id, userID int64, deliveredAt time.Time) error {
	return s.DB.Exec(`UPDATE tasks SET reminder_delivered_at = ?, updated_at = ? WHERE id = ? AND user_id = ?`, deliveredAt, deliveredAt, id, userID)
}

func (s *Store) eventFromRow(row Row) (domain.Event, error) {
	id := Int64(row["id"])
	event := domain.Event{ID: id, CalendarID: Int64(row["calendar_id"]), UID: row["uid"], Title: row["title"], Description: row["description"], Location: row["location"], StartsAt: ParseTime(row["starts_at"]), EndsAt: ParseTime(row["ends_at"]), Timezone: row["timezone"], AllDay: Bool(row["all_day"]), Private: Bool(row["private"]), Completed: Bool(row["completed"]), BirthdayYear: Int(row["birthday_year"]), Status: domain.EventStatus(row["status"]), ETag: row["etag"], CreatedBy: Int64(row["created_by"]), CreatedAt: ParseTime(row["created_at"]), UpdatedAt: ParseTime(row["updated_at"])}
	rec, _ := s.DB.Query(`SELECT * FROM event_recurrence WHERE event_id = ?`, id)
	if len(rec) > 0 {
		r := rec[0]
		event.Recurrence = &domain.Recurrence{ID: Int64(r["id"]), EventID: id, Frequency: domain.RecurrenceFrequency(r["frequency"]), Interval: Int(r["interval"]), Count: Int(r["count"]), Until: ParseTime(r["until_at"]), ByDay: r["by_day"], RRule: r["rrule"], CreatedAt: ParseTime(r["created_at"])}
	}
	attendeeRows, _ := s.DB.Query(`SELECT * FROM event_attendees WHERE event_id = ? ORDER BY email`, id)
	for _, a := range attendeeRows {
		event.Attendees = append(event.Attendees, domain.Attendee{ID: Int64(a["id"]), EventID: id, Email: a["email"], DisplayName: a["display_name"], Status: domain.AttendeeStatus(a["status"]), CreatedAt: ParseTime(a["created_at"])})
	}
	reminderRows, _ := s.DB.Query(`SELECT * FROM event_reminders WHERE event_id = ? ORDER BY minutes_before`, id)
	for _, r := range reminderRows {
		event.Reminders = append(event.Reminders, domain.Reminder{ID: Int64(r["id"]), EventID: id, MinutesBefore: Int(r["minutes_before"]), DeliveredAt: ParseTime(r["delivered_at"]), CreatedAt: ParseTime(r["created_at"])})
	}
	return event, nil
}

func (s *Store) FindConflicts(userID int64, event domain.Event) ([]domain.EventConflict, error) {
	rows, err := s.DB.Query(`SELECT e.id, e.title FROM events e JOIN calendars c ON c.id = e.calendar_id LEFT JOIN calendar_shares cs ON cs.calendar_id = c.id WHERE e.deleted_at IS NULL AND e.id != ? AND e.calendar_id = ? AND e.starts_at < ? AND e.ends_at > ? AND (c.owner_user_id = ? OR cs.user_id = ?)`, event.ID, event.CalendarID, event.EndsAt, event.StartsAt, userID, userID)
	if err != nil {
		return nil, err
	}
	conflicts := make([]domain.EventConflict, 0, len(rows))
	for _, row := range rows {
		conflicts = append(conflicts, domain.EventConflict{EventID: Int64(row["id"]), Title: row["title"]})
	}
	return conflicts, nil
}

func (s *Store) CreateTask(task domain.Task) (domain.Task, error) {
	now := time.Now().UTC()
	if task.Priority == "" {
		task.Priority = domain.TaskPriorityNormal
	}
	if err := s.DB.Exec(`INSERT INTO tasks(user_id, title, description, due_at, reminder_at, priority, completed, show_in_calendar, completed_at, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, task.UserID, strings.TrimSpace(task.Title), task.Description, task.DueAt, task.ReminderAt, string(task.Priority), task.Completed, task.ShowInCalendar, task.CompletedAt, now, now); err != nil {
		return domain.Task{}, err
	}
	return s.FindTaskByID(s.DB.LastInsertID(), task.UserID)
}

func (s *Store) FindTaskByID(id, userID int64) (domain.Task, error) {
	rows, err := s.DB.Query(`SELECT * FROM tasks WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return domain.Task{}, err
	}
	if len(rows) == 0 {
		return domain.Task{}, ErrNotFound
	}
	return taskFromRow(rows[0]), nil
}

func (s *Store) ListTasks(filter TaskFilter) ([]domain.Task, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query := `SELECT * FROM tasks WHERE user_id = ?`
	args := []any{filter.UserID}
	if filter.Completed != nil {
		query += ` AND completed = ?`
		args = append(args, *filter.Completed)
	}
	if filter.Query != "" {
		like := "%" + strings.ToLower(filter.Query) + "%"
		query += ` AND (lower(title) LIKE ? OR lower(description) LIKE ?)`
		args = append(args, like, like)
	}
	query += ` ORDER BY completed ASC, COALESCE(due_at, '9999-12-31T23:59:59Z') ASC, updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, filter.Offset)
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, taskFromRow(row))
	}
	return tasks, nil
}

func (s *Store) UpdateTask(task domain.Task, userID int64) (domain.Task, error) {
	existing, err := s.FindTaskByID(task.ID, userID)
	if err != nil {
		return domain.Task{}, err
	}
	if task.Priority == "" {
		task.Priority = domain.TaskPriorityNormal
	}
	completedAt := existing.CompletedAt
	if task.Completed && completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	if !task.Completed {
		completedAt = time.Time{}
	}
	reminderDeliveredAt := existing.ReminderDeliveredAt
	if !task.ReminderAt.Equal(existing.ReminderAt) {
		reminderDeliveredAt = time.Time{}
	}
	if task.Completed || task.ReminderAt.IsZero() {
		reminderDeliveredAt = time.Time{}
	}
	if err := s.DB.Exec(`UPDATE tasks SET title = ?, description = ?, due_at = ?, reminder_at = ?, priority = ?, completed = ?, show_in_calendar = ?, completed_at = ?, reminder_delivered_at = ?, updated_at = ? WHERE id = ? AND user_id = ?`, strings.TrimSpace(task.Title), task.Description, task.DueAt, task.ReminderAt, string(task.Priority), task.Completed, task.ShowInCalendar, completedAt, reminderDeliveredAt, time.Now().UTC(), task.ID, userID); err != nil {
		return domain.Task{}, err
	}
	return s.FindTaskByID(task.ID, userID)
}

func (s *Store) DeleteTask(id, userID int64) error {
	if _, err := s.FindTaskByID(id, userID); err != nil {
		return err
	}
	return s.DB.Exec(`DELETE FROM tasks WHERE id = ? AND user_id = ?`, id, userID)
}

func taskFromRow(row Row) domain.Task {
	return domain.Task{
		ID:                  Int64(row["id"]),
		UserID:              Int64(row["user_id"]),
		Title:               row["title"],
		Description:         row["description"],
		DueAt:               ParseTime(row["due_at"]),
		ReminderAt:          ParseTime(row["reminder_at"]),
		Priority:            domain.TaskPriority(row["priority"]),
		Completed:           Bool(row["completed"]),
		ShowInCalendar:      Bool(row["show_in_calendar"]),
		CompletedAt:         ParseTime(row["completed_at"]),
		ReminderDeliveredAt: ParseTime(row["reminder_delivered_at"]),
		CreatedAt:           ParseTime(row["created_at"]),
		UpdatedAt:           ParseTime(row["updated_at"]),
	}
}

func (s *Store) CreateContact(contact domain.Contact) (domain.Contact, error) {
	now := time.Now().UTC()
	if err := s.DB.Exec(`INSERT INTO contacts(user_id, first_name, last_name, company, company_email, company_phone, company_mobile, email, phone, mobile, address, birthday, notes, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, contact.UserID, strings.TrimSpace(contact.FirstName), strings.TrimSpace(contact.LastName), strings.TrimSpace(contact.Company), strings.TrimSpace(contact.CompanyEmail), strings.TrimSpace(contact.CompanyPhone), strings.TrimSpace(contact.CompanyMobile), strings.TrimSpace(contact.Email), strings.TrimSpace(contact.Phone), strings.TrimSpace(contact.Mobile), strings.TrimSpace(contact.Address), strings.TrimSpace(contact.Birthday), contact.Notes, now, now); err != nil {
		return domain.Contact{}, err
	}
	return s.FindContactByID(s.DB.LastInsertID(), contact.UserID)
}

func (s *Store) FindContactByID(id, userID int64) (domain.Contact, error) {
	rows, err := s.DB.Query(`SELECT * FROM contacts WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return domain.Contact{}, err
	}
	if len(rows) == 0 {
		return domain.Contact{}, ErrNotFound
	}
	return contactFromRow(rows[0]), nil
}

func (s *Store) ListContacts(filter ContactFilter) ([]domain.Contact, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query := `SELECT * FROM contacts WHERE user_id = ?`
	args := []any{filter.UserID}
	if filter.Query != "" {
		like := "%" + strings.ToLower(filter.Query) + "%"
		query += ` AND (lower(first_name) LIKE ? OR lower(last_name) LIKE ? OR lower(company) LIKE ? OR lower(company_email) LIKE ? OR lower(company_phone) LIKE ? OR lower(company_mobile) LIKE ? OR lower(email) LIKE ? OR lower(phone) LIKE ? OR lower(mobile) LIKE ? OR lower(address) LIKE ? OR lower(notes) LIKE ?)`
		args = append(args, like, like, like, like, like, like, like, like, like, like, like)
	}
	query += ` ORDER BY lower(last_name) ASC, lower(first_name) ASC, lower(company) ASC, updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, filter.Offset)
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	contacts := make([]domain.Contact, 0, len(rows))
	for _, row := range rows {
		contacts = append(contacts, contactFromRow(row))
	}
	return contacts, nil
}

func (s *Store) UpdateContact(contact domain.Contact, userID int64) (domain.Contact, error) {
	if _, err := s.FindContactByID(contact.ID, userID); err != nil {
		return domain.Contact{}, err
	}
	if err := s.DB.Exec(`UPDATE contacts SET first_name = ?, last_name = ?, company = ?, company_email = ?, company_phone = ?, company_mobile = ?, email = ?, phone = ?, mobile = ?, address = ?, birthday = ?, notes = ?, updated_at = ? WHERE id = ? AND user_id = ?`, strings.TrimSpace(contact.FirstName), strings.TrimSpace(contact.LastName), strings.TrimSpace(contact.Company), strings.TrimSpace(contact.CompanyEmail), strings.TrimSpace(contact.CompanyPhone), strings.TrimSpace(contact.CompanyMobile), strings.TrimSpace(contact.Email), strings.TrimSpace(contact.Phone), strings.TrimSpace(contact.Mobile), strings.TrimSpace(contact.Address), strings.TrimSpace(contact.Birthday), contact.Notes, time.Now().UTC(), contact.ID, userID); err != nil {
		return domain.Contact{}, err
	}
	return s.FindContactByID(contact.ID, userID)
}

func (s *Store) DeleteContact(id, userID int64) error {
	if _, err := s.FindContactByID(id, userID); err != nil {
		return err
	}
	return s.DB.Exec(`DELETE FROM contacts WHERE id = ? AND user_id = ?`, id, userID)
}

func contactFromRow(row Row) domain.Contact {
	return domain.Contact{
		ID:            Int64(row["id"]),
		UserID:        Int64(row["user_id"]),
		FirstName:     row["first_name"],
		LastName:      row["last_name"],
		Company:       row["company"],
		CompanyEmail:  row["company_email"],
		CompanyPhone:  row["company_phone"],
		CompanyMobile: row["company_mobile"],
		Email:         row["email"],
		Phone:         row["phone"],
		Mobile:        row["mobile"],
		Address:       row["address"],
		Birthday:      row["birthday"],
		Notes:         row["notes"],
		CreatedAt:     ParseTime(row["created_at"]),
		UpdatedAt:     ParseTime(row["updated_at"]),
	}
}

func (s *Store) CreateCalDAVToken(userID int64, name, tokenHash, hint string) (domain.CalDAVAccount, error) {
	now := time.Now().UTC()
	if err := s.DB.Exec(`INSERT INTO caldav_accounts(user_id, name, token_hash, token_hint, created_at) VALUES(?,?,?,?,?)`, userID, strings.TrimSpace(name), tokenHash, hint, now); err != nil {
		return domain.CalDAVAccount{}, err
	}
	return domain.CalDAVAccount{ID: s.DB.LastInsertID(), UserID: userID, Name: name, TokenHint: hint, CreatedAt: now}, nil
}

func (s *Store) FindCalDAVUser(email, tokenHash string) (domain.User, error) {
	rows, err := s.DB.Query(`SELECT u.id FROM users u JOIN caldav_accounts ca ON ca.user_id = u.id WHERE u.email = ? AND ca.token_hash = ? AND ca.revoked_at IS NULL AND u.active = 1`, domain.NormalizeEmail(email), tokenHash)
	if err != nil {
		return domain.User{}, err
	}
	if len(rows) == 0 {
		return domain.User{}, ErrNotFound
	}
	_ = s.DB.Exec(`UPDATE caldav_accounts SET last_used_at = ? WHERE token_hash = ?`, time.Now().UTC(), tokenHash)
	return s.FindUserByID(Int64(rows[0]["id"]))
}

func (s *Store) ListCalDAVTokens(userID int64) ([]domain.CalDAVAccount, error) {
	rows, err := s.DB.Query(`SELECT id, user_id, name, token_hint, last_used_at, revoked_at, created_at FROM caldav_accounts WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CalDAVAccount, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CalDAVAccount{ID: Int64(row["id"]), UserID: Int64(row["user_id"]), Name: row["name"], TokenHint: row["token_hint"], LastUsedAt: ParseTime(row["last_used_at"]), RevokedAt: ParseTime(row["revoked_at"]), CreatedAt: ParseTime(row["created_at"])})
	}
	return out, nil
}

func (s *Store) FindCalDAVConnection(userID int64) (domain.CalDAVConnection, error) {
	rows, err := s.DB.Query(`SELECT * FROM caldav_connections WHERE user_id = ?`, userID)
	if err != nil {
		return domain.CalDAVConnection{}, err
	}
	if len(rows) == 0 {
		return domain.CalDAVConnection{}, ErrNotFound
	}
	return calDAVConnectionFromRow(rows[0]), nil
}

func (s *Store) ListEnabledCalDAVConnections() ([]domain.CalDAVConnection, error) {
	rows, err := s.DB.Query(`SELECT * FROM caldav_connections WHERE sync_enabled = 1 ORDER BY updated_at ASC`)
	if err != nil {
		return nil, err
	}
	connections := make([]domain.CalDAVConnection, 0, len(rows))
	for _, row := range rows {
		connections = append(connections, calDAVConnectionFromRow(row))
	}
	return connections, nil
}

func (s *Store) UpsertCalDAVConnection(connection domain.CalDAVConnection) (domain.CalDAVConnection, error) {
	now := time.Now().UTC()
	if connection.SyncDirection == "" {
		connection.SyncDirection = "pull"
	}
	if connection.SyncIntervalMinutes == 0 {
		connection.SyncIntervalMinutes = 60
	}
	if connection.SyncWindowPastDays == 0 {
		connection.SyncWindowPastDays = 30
	}
	if connection.SyncWindowFutureDays == 0 {
		connection.SyncWindowFutureDays = 365
	}
	existing, err := s.FindCalDAVConnection(connection.UserID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return domain.CalDAVConnection{}, err
	}
	if errors.Is(err, ErrNotFound) {
		if err := s.DB.Exec(`INSERT INTO caldav_connections(user_id, display_name, base_url, username, password_encrypted, sync_enabled, sync_direction, sync_events, sync_tasks, sync_contacts, sync_interval_minutes, sync_window_past_days, sync_window_future_days, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, connection.UserID, strings.TrimSpace(connection.DisplayName), strings.TrimSpace(connection.BaseURL), strings.TrimSpace(connection.Username), connection.PasswordEncrypted, connection.SyncEnabled, connection.SyncDirection, connection.SyncEvents, connection.SyncTasks, connection.SyncContacts, connection.SyncIntervalMinutes, connection.SyncWindowPastDays, connection.SyncWindowFutureDays, now, now); err != nil {
			return domain.CalDAVConnection{}, err
		}
		return s.FindCalDAVConnection(connection.UserID)
	}
	if strings.TrimSpace(connection.PasswordEncrypted) == "" {
		connection.PasswordEncrypted = existing.PasswordEncrypted
	}
	if err := s.DB.Exec(`UPDATE caldav_connections SET display_name = ?, base_url = ?, username = ?, password_encrypted = ?, sync_enabled = ?, sync_direction = ?, sync_events = ?, sync_tasks = ?, sync_contacts = ?, sync_interval_minutes = ?, sync_window_past_days = ?, sync_window_future_days = ?, updated_at = ? WHERE user_id = ?`, strings.TrimSpace(connection.DisplayName), strings.TrimSpace(connection.BaseURL), strings.TrimSpace(connection.Username), connection.PasswordEncrypted, connection.SyncEnabled, connection.SyncDirection, connection.SyncEvents, connection.SyncTasks, connection.SyncContacts, connection.SyncIntervalMinutes, connection.SyncWindowPastDays, connection.SyncWindowFutureDays, now, connection.UserID); err != nil {
		return domain.CalDAVConnection{}, err
	}
	return s.FindCalDAVConnection(connection.UserID)
}

func (s *Store) UpdateCalDAVConnectionTest(userID int64, status, message string) error {
	return s.DB.Exec(`UPDATE caldav_connections SET last_test_at = ?, last_test_status = ?, last_test_message = ?, updated_at = ? WHERE user_id = ?`, time.Now().UTC(), status, message, time.Now().UTC(), userID)
}

func (s *Store) UpdateCalDAVConnectionSync(userID int64, status, message string) error {
	now := time.Now().UTC()
	return s.DB.Exec(`UPDATE caldav_connections SET last_sync_at = ?, last_sync_status = ?, last_sync_message = ?, updated_at = ? WHERE user_id = ?`, now, status, message, now, userID)
}

func (s *Store) ListDAVCollections(userID int64) ([]domain.DAVCollection, error) {
	rows, err := s.DB.Query(`SELECT * FROM dav_collections WHERE user_id = ? ORDER BY kind, display_name`, userID)
	if err != nil {
		return nil, err
	}
	items := make([]domain.DAVCollection, 0, len(rows))
	for _, row := range rows {
		items = append(items, davCollectionFromRow(row))
	}
	return items, nil
}

func (s *Store) UpsertDAVCollections(userID int64, collections []domain.DAVCollection) ([]domain.DAVCollection, error) {
	now := time.Now().UTC()
	for _, collection := range collections {
		if strings.TrimSpace(collection.URL) == "" {
			continue
		}
		selected := collection.Selected
		rows, err := s.DB.Query(`SELECT selected FROM dav_collections WHERE user_id = ? AND url = ?`, userID, collection.URL)
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			selected = Bool(rows[0]["selected"])
		}
		if err := s.DB.Exec(`INSERT INTO dav_collections(user_id, kind, display_name, url, selected, supports_events, supports_tasks, ctag, sync_token, last_seen_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(user_id, url) DO UPDATE SET kind = excluded.kind, display_name = excluded.display_name, supports_events = excluded.supports_events, supports_tasks = excluded.supports_tasks, ctag = excluded.ctag, sync_token = excluded.sync_token, last_seen_at = excluded.last_seen_at, updated_at = excluded.updated_at`, userID, collection.Kind, strings.TrimSpace(collection.DisplayName), collection.URL, selected, collection.SupportsEvents, collection.SupportsTasks, collection.CTag, collection.SyncToken, now, now); err != nil {
			return nil, err
		}
	}
	return s.ListDAVCollections(userID)
}

func (s *Store) UpdateDAVCollectionSelections(userID int64, selections map[string]bool) ([]domain.DAVCollection, error) {
	now := time.Now().UTC()
	for rawURL, selected := range selections {
		if err := s.DB.Exec(`UPDATE dav_collections SET selected = ?, updated_at = ? WHERE user_id = ? AND url = ?`, selected, now, userID, rawURL); err != nil {
			return nil, err
		}
	}
	return s.ListDAVCollections(userID)
}

func (s *Store) FindDAVSyncItem(userID int64, resourceURL, kind string) (domain.DAVSyncItem, error) {
	rows, err := s.DB.Query(`SELECT * FROM dav_sync_items WHERE user_id = ? AND resource_url = ? AND kind = ?`, userID, resourceURL, kind)
	if err != nil {
		return domain.DAVSyncItem{}, err
	}
	if len(rows) == 0 {
		return domain.DAVSyncItem{}, ErrNotFound
	}
	return davSyncItemFromRow(rows[0]), nil
}

func (s *Store) FindDAVSyncItemByLocalID(userID, localID int64, kind string) (domain.DAVSyncItem, error) {
	rows, err := s.DB.Query(`SELECT * FROM dav_sync_items WHERE user_id = ? AND local_id = ? AND kind = ? ORDER BY updated_at DESC LIMIT 1`, userID, localID, kind)
	if err != nil {
		return domain.DAVSyncItem{}, err
	}
	if len(rows) == 0 {
		return domain.DAVSyncItem{}, ErrNotFound
	}
	return davSyncItemFromRow(rows[0]), nil
}

func (s *Store) ListDAVSyncItemsForCollection(userID int64, collectionURL, kind string) ([]domain.DAVSyncItem, error) {
	rows, err := s.DB.Query(`SELECT * FROM dav_sync_items WHERE user_id = ? AND collection_url = ? AND kind = ?`, userID, collectionURL, kind)
	if err != nil {
		return nil, err
	}
	items := make([]domain.DAVSyncItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, davSyncItemFromRow(row))
	}
	return items, nil
}

func (s *Store) CreateDAVSyncRun(run domain.DAVSyncRun) error {
	if run.Mode == "" {
		run.Mode = "manual"
	}
	if run.Status == "" {
		run.Status = "ok"
	}
	if run.Warnings == "" {
		run.Warnings = "[]"
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = time.Now().UTC()
	}
	return s.DB.Exec(`INSERT INTO dav_sync_runs(user_id, mode, status, message, events, tasks, contacts, skipped, warnings, created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, run.UserID, run.Mode, run.Status, run.Message, run.Events, run.Tasks, run.Contacts, run.Skipped, run.Warnings, run.CreatedAt)
}

func (s *Store) ListDAVSyncRuns(userID int64, limit int) ([]domain.DAVSyncRun, error) {
	if limit <= 0 || limit > 50 {
		limit = 8
	}
	rows, err := s.DB.Query(`SELECT * FROM dav_sync_runs WHERE user_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	items := make([]domain.DAVSyncRun, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.DAVSyncRun{
			ID:        Int64(row["id"]),
			UserID:    Int64(row["user_id"]),
			Mode:      row["mode"],
			Status:    row["status"],
			Message:   row["message"],
			Events:    Int(row["events"]),
			Tasks:     Int(row["tasks"]),
			Contacts:  Int(row["contacts"]),
			Skipped:   Int(row["skipped"]),
			Warnings:  row["warnings"],
			CreatedAt: ParseTime(row["created_at"]),
		})
	}
	return items, nil
}

func (s *Store) ListDAVSyncedLocalIDs(userID int64, kind string) (map[int64]bool, error) {
	rows, err := s.DB.Query(`SELECT local_id FROM dav_sync_items WHERE user_id = ? AND kind = ?`, userID, kind)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]bool, len(rows))
	for _, row := range rows {
		localID := Int64(row["local_id"])
		if localID > 0 {
			out[localID] = true
		}
	}
	return out, nil
}

func (s *Store) UpsertDAVSyncItem(item domain.DAVSyncItem) error {
	return s.DB.Exec(`INSERT INTO dav_sync_items(user_id, collection_url, resource_url, kind, local_id, uid, etag, updated_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(user_id, resource_url, kind) DO UPDATE SET collection_url = excluded.collection_url, local_id = excluded.local_id, uid = excluded.uid, etag = excluded.etag, updated_at = excluded.updated_at`, item.UserID, item.CollectionURL, item.ResourceURL, item.Kind, item.LocalID, item.UID, item.ETag, time.Now().UTC())
}

func (s *Store) DeleteDAVSyncItem(userID, id int64) error {
	return s.DB.Exec(`DELETE FROM dav_sync_items WHERE user_id = ? AND id = ?`, userID, id)
}

func davSyncItemFromRow(row Row) domain.DAVSyncItem {
	return domain.DAVSyncItem{
		ID:            Int64(row["id"]),
		UserID:        Int64(row["user_id"]),
		CollectionURL: row["collection_url"],
		ResourceURL:   row["resource_url"],
		Kind:          row["kind"],
		LocalID:       Int64(row["local_id"]),
		UID:           row["uid"],
		ETag:          row["etag"],
		UpdatedAt:     ParseTime(row["updated_at"]),
	}
}

func davCollectionFromRow(row Row) domain.DAVCollection {
	return domain.DAVCollection{
		ID:             Int64(row["id"]),
		UserID:         Int64(row["user_id"]),
		Kind:           row["kind"],
		DisplayName:    row["display_name"],
		URL:            row["url"],
		Selected:       Bool(row["selected"]),
		SupportsEvents: Bool(row["supports_events"]),
		SupportsTasks:  Bool(row["supports_tasks"]),
		CTag:           row["ctag"],
		SyncToken:      row["sync_token"],
		LastSeenAt:     ParseTime(row["last_seen_at"]),
		UpdatedAt:      ParseTime(row["updated_at"]),
	}
}

func calDAVConnectionFromRow(row Row) domain.CalDAVConnection {
	password := row["password_encrypted"]
	return domain.CalDAVConnection{
		ID:                   Int64(row["id"]),
		UserID:               Int64(row["user_id"]),
		DisplayName:          row["display_name"],
		BaseURL:              row["base_url"],
		Username:             row["username"],
		PasswordEncrypted:    password,
		PasswordConfigured:   strings.TrimSpace(password) != "",
		SyncEnabled:          Bool(row["sync_enabled"]),
		SyncDirection:        row["sync_direction"],
		SyncEvents:           Bool(row["sync_events"]),
		SyncTasks:            Bool(row["sync_tasks"]),
		SyncContacts:         Bool(row["sync_contacts"]),
		SyncIntervalMinutes:  Int(row["sync_interval_minutes"]),
		SyncWindowPastDays:   Int(row["sync_window_past_days"]),
		SyncWindowFutureDays: Int(row["sync_window_future_days"]),
		LastTestAt:           ParseTime(row["last_test_at"]),
		LastTestStatus:       row["last_test_status"],
		LastTestMessage:      row["last_test_message"],
		LastSyncAt:           ParseTime(row["last_sync_at"]),
		LastSyncStatus:       row["last_sync_status"],
		LastSyncMessage:      row["last_sync_message"],
		CreatedAt:            ParseTime(row["created_at"]),
		UpdatedAt:            ParseTime(row["updated_at"]),
	}
}

func (s *Store) RecordExcelExport(export domain.ExcelExport) error {
	return s.DB.Exec(`INSERT INTO excel_exports(user_id, kind, format, range_start, range_end, created_at) VALUES(?,?,?,?,?,?)`, export.UserID, export.Kind, export.Format, export.RangeStart, export.RangeEnd, time.Now().UTC())
}

func (s *Store) ListAppDataForBackup() (map[string]any, error) {
	data := map[string]any{"version": 2, "createdAt": FormatTime(time.Now().UTC())}
	for _, table := range backupTables {
		rows, err := s.DB.Query(fmt.Sprintf(`SELECT * FROM %s ORDER BY id`, table))
		if err != nil {
			return nil, err
		}
		data[table] = rows
	}
	return data, nil
}

func (s *Store) RestoreAppDataFromBackup(data map[string]any) error {
	if err := s.DB.ExecScript(`PRAGMA foreign_keys = OFF;`); err != nil {
		return err
	}
	defer func() { _ = s.DB.ExecScript(`PRAGMA foreign_keys = ON;`) }()
	for i := len(backupTables) - 1; i >= 0; i-- {
		if err := s.DB.Exec(fmt.Sprintf(`DELETE FROM %s`, backupTables[i])); err != nil {
			return err
		}
	}
	for _, table := range backupTables {
		rows, ok := backupRows(data[table])
		if !ok {
			continue
		}
		allowed := backupTableColumns[table]
		for _, row := range rows {
			columns := make([]string, 0, len(row))
			args := make([]any, 0, len(row))
			placeholders := make([]string, 0, len(row))
			for _, column := range allowed {
				value, exists := row[column]
				if !exists {
					continue
				}
				columns = append(columns, column)
				placeholders = append(placeholders, "?")
				args = append(args, restoreValue(value))
			}
			if len(columns) == 0 {
				continue
			}
			query := fmt.Sprintf(`INSERT OR REPLACE INTO %s(%s) VALUES(%s)`, table, strings.Join(columns, ","), strings.Join(placeholders, ","))
			if err := s.DB.Exec(query, args...); err != nil {
				return err
			}
		}
	}
	return nil
}

var ErrNotFound = errors.New("not found")

var backupTables = []string{
	"calendars",
	"events",
	"event_recurrence",
	"event_attendees",
	"event_reminders",
	"tasks",
	"contacts",
	"dav_collections",
	"dav_sync_items",
	"dav_sync_runs",
}

var backupTableColumns = map[string][]string{
	"calendars":        {"id", "owner_user_id", "name", "description", "color", "timezone", "visible", "reminder_enabled", "reminder_days_before", "reminder_time", "same_day_reminder_time", "deleted_at", "created_at", "updated_at"},
	"events":           {"id", "calendar_id", "uid", "title", "description", "location", "starts_at", "ends_at", "timezone", "all_day", "private", "status", "source", "external_id", "external_url", "etag", "microsoft_join_url", "created_by", "completed", "birthday_year", "deleted_at", "created_at", "updated_at"},
	"event_recurrence": {"id", "event_id", "frequency", "interval", "count", "until_at", "by_day", "rrule", "created_at"},
	"event_attendees":  {"id", "event_id", "email", "display_name", "status", "created_at"},
	"event_reminders":  {"id", "event_id", "minutes_before", "delivered_at", "created_at"},
	"tasks":            {"id", "user_id", "title", "description", "due_at", "priority", "completed", "show_in_calendar", "reminder_at", "reminder_delivered_at", "completed_at", "created_at", "updated_at"},
	"contacts":         {"id", "user_id", "first_name", "last_name", "company", "company_email", "company_phone", "company_mobile", "email", "phone", "mobile", "address", "birthday", "notes", "created_at", "updated_at"},
	"dav_collections":  {"id", "user_id", "kind", "display_name", "url", "selected", "supports_events", "supports_tasks", "ctag", "sync_token", "last_seen_at", "updated_at"},
	"dav_sync_items":   {"id", "user_id", "collection_url", "resource_url", "kind", "local_id", "uid", "etag", "updated_at"},
	"dav_sync_runs":    {"id", "user_id", "mode", "status", "message", "events", "tasks", "contacts", "skipped", "warnings", "created_at"},
}

func backupRows(value any) ([]map[string]any, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if ok {
			rows = append(rows, row)
		}
	}
	return rows, true
}

func restoreValue(value any) any {
	if text, ok := value.(string); ok && text == "" {
		return nil
	}
	return value
}

func sanitizeLogField(v string, max int) string {
	v = strings.ReplaceAll(v, "\n", " ")
	v = strings.ReplaceAll(v, "\r", " ")
	if len(v) > max {
		return v[:max]
	}
	return v
}

func nullZero(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
