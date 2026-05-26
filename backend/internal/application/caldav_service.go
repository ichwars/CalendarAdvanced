package application

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/crypto"
	"calendaradvanced/internal/infrastructure/ics"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type CalDAVService struct {
	Store      *sqlite.Store
	Audit      *AuditService
	Cipher     *crypto.TokenCipher
	HTTPClient *http.Client
	autoSyncMu sync.Mutex
}

type CalDAVTokenResult struct {
	Account domain.CalDAVAccount `json:"account"`
	Token   string               `json:"token"`
}

type CalDAVConnectionInput struct {
	DisplayName          string `json:"displayName"`
	BaseURL              string `json:"baseUrl"`
	Username             string `json:"username"`
	Password             string `json:"password"`
	SyncEnabled          bool   `json:"syncEnabled"`
	SyncDirection        string `json:"syncDirection"`
	SyncEvents           bool   `json:"syncEvents"`
	SyncTasks            bool   `json:"syncTasks"`
	SyncContacts         bool   `json:"syncContacts"`
	SyncIntervalMinutes  int    `json:"syncIntervalMinutes"`
	SyncWindowPastDays   int    `json:"syncWindowPastDays"`
	SyncWindowFutureDays int    `json:"syncWindowFutureDays"`
}

type CalDAVConnectionTestResult struct {
	OK          bool   `json:"ok"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	StatusCode  int    `json:"statusCode,omitempty"`
	CalendarURL string `json:"calendarUrl,omitempty"`
}

type DAVCollectionSelection struct {
	URL      string `json:"url"`
	Selected bool   `json:"selected"`
}

type DAVCollectionSelectionInput struct {
	Items []DAVCollectionSelection `json:"items"`
}

type DAVDiscoveryResult struct {
	Items   []domain.DAVCollection `json:"items"`
	Message string                 `json:"message"`
}

type DAVSyncResult struct {
	OK               bool     `json:"ok"`
	Status           string   `json:"status"`
	Message          string   `json:"message"`
	EventsImported   int      `json:"eventsImported"`
	EventsUpdated    int      `json:"eventsUpdated"`
	TasksImported    int      `json:"tasksImported"`
	TasksUpdated     int      `json:"tasksUpdated"`
	ContactsImported int      `json:"contactsImported"`
	ContactsUpdated  int      `json:"contactsUpdated"`
	EventsExported   int      `json:"eventsExported"`
	TasksExported    int      `json:"tasksExported"`
	ContactsExported int      `json:"contactsExported"`
	EventsDeleted    int      `json:"eventsDeleted"`
	TasksDeleted     int      `json:"tasksDeleted"`
	ContactsDeleted  int      `json:"contactsDeleted"`
	Skipped          int      `json:"skipped"`
	Warnings         []string `json:"warnings,omitempty"`
}

type DAVSyncInput struct {
	ConflictStrategy string `json:"conflictStrategy"`
}

type DAVSyncHistoryItem struct {
	ID        int64     `json:"id"`
	Mode      string    `json:"mode"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Events    int       `json:"events"`
	Tasks     int       `json:"tasks"`
	Contacts  int       `json:"contacts"`
	Skipped   int       `json:"skipped"`
	Warnings  []string  `json:"warnings,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type davPropfindResponse struct {
	Responses []davResponse `xml:"response"`
}

type davResponse struct {
	Href     string        `xml:"href"`
	Propstat []davPropstat `xml:"propstat"`
}

type davPropstat struct {
	Prop davProp `xml:"prop"`
}

type davProp struct {
	DisplayName  string      `xml:"displayname"`
	Resource     innerXMLTag `xml:"resourcetype"`
	Components   innerXMLTag `xml:"supported-calendar-component-set"`
	CTag         string      `xml:"getctag"`
	SyncToken    string      `xml:"sync-token"`
	ETag         string      `xml:"getetag"`
	CalendarData string      `xml:"calendar-data"`
	AddressData  string      `xml:"address-data"`
}

type innerXMLTag struct {
	Inner string `xml:",innerxml"`
}

func (s *CalDAVService) CreateToken(user domain.User, name, ip, userAgent string) (CalDAVTokenResult, error) {
	if strings.TrimSpace(name) == "" {
		name = "DAVx5"
	}
	token, err := crypto.RandomToken(32)
	if err != nil {
		return CalDAVTokenResult{}, err
	}
	hint := token
	if len(hint) > 8 {
		hint = hint[len(hint)-8:]
	}
	account, err := s.Store.CreateCalDAVToken(user.ID, name, crypto.HashToken(token), hint)
	if err != nil {
		return CalDAVTokenResult{}, err
	}
	s.Audit.Record(user.ID, domain.AuditIntegrationChanged, "caldav_account", fmt.Sprint(account.ID), ip, userAgent, map[string]any{"operation": "create_token"})
	return CalDAVTokenResult{Account: account, Token: token}, nil
}

func (s *CalDAVService) ListTokens(user domain.User) ([]domain.CalDAVAccount, error) {
	return s.Store.ListCalDAVTokens(user.ID)
}

func (s *CalDAVService) Authenticate(email, token string) (domain.User, error) {
	return s.Store.FindCalDAVUser(email, crypto.HashToken(token))
}

func (s *CalDAVService) GetConnection(user domain.User) (domain.CalDAVConnection, error) {
	connection, err := s.Store.FindCalDAVConnection(user.ID)
	if err != nil {
		if errors.Is(err, sqlite.ErrNotFound) {
			return defaultCalDAVConnection(user.ID), nil
		}
		return domain.CalDAVConnection{}, err
	}
	return connection, nil
}

func (s *CalDAVService) SaveConnection(user domain.User, input CalDAVConnectionInput, ip, userAgent string) (domain.CalDAVConnection, error) {
	connection, err := s.connectionFromInput(user, input, false)
	if err != nil {
		return domain.CalDAVConnection{}, err
	}
	saved, err := s.Store.UpsertCalDAVConnection(connection)
	if err != nil {
		return domain.CalDAVConnection{}, err
	}
	s.Audit.Record(user.ID, domain.AuditIntegrationChanged, "caldav_connection", fmt.Sprint(saved.ID), ip, userAgent, map[string]any{"operation": "save_connection"})
	return saved, nil
}

func (s *CalDAVService) TestConnection(user domain.User, input CalDAVConnectionInput) (CalDAVConnectionTestResult, error) {
	connection, err := s.connectionFromInput(user, input, true)
	if err != nil {
		return CalDAVConnectionTestResult{}, err
	}
	password := strings.TrimSpace(input.Password)
	if password == "" {
		stored, err := s.Store.FindCalDAVConnection(user.ID)
		if err != nil {
			return CalDAVConnectionTestResult{}, NewError("caldav_password_required", "Passwort ist für den Verbindungstest erforderlich.", nil)
		}
		password, err = s.decryptPassword(stored.PasswordEncrypted)
		if err != nil {
			return CalDAVConnectionTestResult{}, NewError("caldav_password_unavailable", "Gespeichertes Passwort konnte nicht gelesen werden.", nil)
		}
	}
	result := s.probeConnection(connection.BaseURL, connection.Username, password)
	if existing, err := s.Store.FindCalDAVConnection(user.ID); err == nil && existing.ID > 0 {
		_ = s.Store.UpdateCalDAVConnectionTest(user.ID, result.Status, result.Message)
	}
	return result, nil
}

func (s *CalDAVService) ListCollections(user domain.User) ([]domain.DAVCollection, error) {
	return s.Store.ListDAVCollections(user.ID)
}

func (s *CalDAVService) SaveCollectionSelections(user domain.User, input DAVCollectionSelectionInput) ([]domain.DAVCollection, error) {
	selections := make(map[string]bool, len(input.Items))
	for _, item := range input.Items {
		if strings.TrimSpace(item.URL) != "" {
			selections[item.URL] = item.Selected
		}
	}
	return s.Store.UpdateDAVCollectionSelections(user.ID, selections)
}

func (s *CalDAVService) DiscoverCollections(user domain.User, input CalDAVConnectionInput, ip, userAgent string) (DAVDiscoveryResult, error) {
	connection, err := s.connectionFromInput(user, input, true)
	if err != nil {
		return DAVDiscoveryResult{}, err
	}
	password := strings.TrimSpace(input.Password)
	if password == "" {
		stored, err := s.Store.FindCalDAVConnection(user.ID)
		if err != nil {
			return DAVDiscoveryResult{}, NewError("caldav_password_required", "Passwort ist für die Collection-Suche erforderlich.", nil)
		}
		password, err = s.decryptPassword(stored.PasswordEncrypted)
		if err != nil {
			return DAVDiscoveryResult{}, NewError("caldav_password_unavailable", "Gespeichertes Passwort konnte nicht gelesen werden.", nil)
		}
	}
	discovered, err := s.discoverDAVCollections(connection.BaseURL, connection.Username, password)
	if err != nil {
		return DAVDiscoveryResult{}, err
	}
	for i := range discovered {
		discovered[i].UserID = user.ID
	}
	items, err := s.Store.UpsertDAVCollections(user.ID, discovered)
	if err != nil {
		return DAVDiscoveryResult{}, err
	}
	s.Audit.Record(user.ID, domain.AuditIntegrationChanged, "dav_collections", "discovery", ip, userAgent, map[string]any{"count": len(discovered)})
	message := fmt.Sprintf("%d DAV-Collections gefunden.", len(discovered))
	if len(discovered) == 0 {
		message = "Keine Kalender oder Adressbücher gefunden. Bitte prüfe die DAV-Server-URL."
	}
	return DAVDiscoveryResult{Items: items, Message: message}, nil
}

func (s *CalDAVService) SyncNow(user domain.User, ip, userAgent string, input DAVSyncInput) (DAVSyncResult, error) {
	connection, err := s.Store.FindCalDAVConnection(user.ID)
	if err != nil {
		return DAVSyncResult{}, NewError("caldav_connection_missing", "Bitte speichere zuerst die DAV-Verbindung.", nil)
	}
	result, err := s.syncConnection(user, connection, input)
	if err != nil {
		s.recordDAVSyncAudit(user.ID, "manual", ip, userAgent, DAVSyncResult{OK: false, Status: "error", Message: err.Error()})
		return DAVSyncResult{}, err
	}
	s.recordDAVSyncAudit(user.ID, "manual", ip, userAgent, result)
	return result, nil
}

func (s *CalDAVService) ListSyncHistory(user domain.User) ([]DAVSyncHistoryItem, error) {
	runs, err := s.Store.ListDAVSyncRuns(user.ID, 8)
	if err != nil {
		return nil, err
	}
	if len(runs) > 0 {
		items := make([]DAVSyncHistoryItem, 0, len(runs))
		for _, run := range runs {
			var warnings []string
			_ = json.Unmarshal([]byte(run.Warnings), &warnings)
			items = append(items, DAVSyncHistoryItem{
				ID:        run.ID,
				Mode:      run.Mode,
				Status:    run.Status,
				Message:   run.Message,
				Events:    run.Events,
				Tasks:     run.Tasks,
				Contacts:  run.Contacts,
				Skipped:   run.Skipped,
				Warnings:  warnings,
				CreatedAt: run.CreatedAt,
			})
		}
		return items, nil
	}
	entries, err := s.Store.ListDAVSyncAudit(user.ID, 8)
	if err != nil {
		return nil, err
	}
	items := make([]DAVSyncHistoryItem, 0, len(entries))
	for _, entry := range entries {
		item := DAVSyncHistoryItem{ID: entry.ID, Mode: entry.EntityID, Status: "ok", CreatedAt: entry.CreatedAt}
		if item.Mode == "" {
			item.Mode = "manual"
		}
		if strings.TrimSpace(entry.Metadata) != "" {
			var meta struct {
				Status   string   `json:"status"`
				Message  string   `json:"message"`
				Events   int      `json:"events"`
				Tasks    int      `json:"tasks"`
				Contacts int      `json:"contacts"`
				Skipped  int      `json:"skipped"`
				Warnings []string `json:"warnings"`
			}
			if err := json.Unmarshal([]byte(entry.Metadata), &meta); err == nil {
				item.Status = meta.Status
				item.Message = meta.Message
				item.Events = meta.Events
				item.Tasks = meta.Tasks
				item.Contacts = meta.Contacts
				item.Skipped = meta.Skipped
				item.Warnings = meta.Warnings
			}
		}
		if item.Status == "" {
			item.Status = "ok"
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *CalDAVService) recordDAVSyncAudit(userID int64, mode, ip, userAgent string, result DAVSyncResult) {
	warnings, _ := json.Marshal(result.Warnings)
	_ = s.Store.CreateDAVSyncRun(domain.DAVSyncRun{
		UserID:    userID,
		Mode:      mode,
		Status:    result.Status,
		Message:   result.Message,
		Events:    result.EventsImported + result.EventsUpdated + result.EventsExported + result.EventsDeleted,
		Tasks:     result.TasksImported + result.TasksUpdated + result.TasksExported + result.TasksDeleted,
		Contacts:  result.ContactsImported + result.ContactsUpdated + result.ContactsExported + result.ContactsDeleted,
		Skipped:   result.Skipped,
		Warnings:  string(warnings),
		CreatedAt: time.Now().UTC(),
	})
	if s.Audit == nil {
		return
	}
	s.Audit.Record(userID, domain.AuditIntegrationChanged, "dav_sync", mode, ip, userAgent, davSyncAuditMetadata(result))
}

func davSyncAuditMetadata(result DAVSyncResult) map[string]any {
	return map[string]any{
		"status":   result.Status,
		"message":  result.Message,
		"events":   result.EventsImported + result.EventsUpdated + result.EventsExported + result.EventsDeleted,
		"tasks":    result.TasksImported + result.TasksUpdated + result.TasksExported + result.TasksDeleted,
		"contacts": result.ContactsImported + result.ContactsUpdated + result.ContactsExported + result.ContactsDeleted,
		"skipped":  result.Skipped,
		"warnings": result.Warnings,
	}
}

func (s *CalDAVService) RunAutoSync(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	s.syncDueConnections()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncDueConnections()
		}
	}
}

func (s *CalDAVService) syncDueConnections() {
	if !s.autoSyncMu.TryLock() {
		return
	}
	defer s.autoSyncMu.Unlock()
	connections, err := s.Store.ListEnabledCalDAVConnections()
	if err != nil {
		return
	}
	now := time.Now().UTC()
	for _, connection := range connections {
		interval := time.Duration(connection.SyncIntervalMinutes) * time.Minute
		if interval < 15*time.Minute {
			interval = 15 * time.Minute
		}
		if !connection.LastSyncAt.IsZero() && now.Sub(connection.LastSyncAt) < interval {
			continue
		}
		user, err := s.Store.FindUserByID(connection.UserID)
		if err != nil {
			continue
		}
		result, err := s.syncConnection(user, connection, DAVSyncInput{})
		if err != nil {
			_ = s.Store.UpdateCalDAVConnectionSync(connection.UserID, "error", err.Error())
			s.recordDAVSyncAudit(user.ID, "auto", "", "auto-sync", DAVSyncResult{OK: false, Status: "error", Message: err.Error()})
			continue
		}
		s.recordDAVSyncAudit(user.ID, "auto", "", "auto-sync", result)
	}
}

func (s *CalDAVService) syncConnection(user domain.User, connection domain.CalDAVConnection, input DAVSyncInput) (DAVSyncResult, error) {
	password, err := s.decryptPassword(connection.PasswordEncrypted)
	if err != nil {
		return DAVSyncResult{}, err
	}
	collections, err := s.Store.ListDAVCollections(user.ID)
	if err != nil {
		return DAVSyncResult{}, err
	}
	selected := make([]domain.DAVCollection, 0, len(collections))
	for _, collection := range collections {
		if collection.Selected {
			selected = append(selected, collection)
		}
	}
	if len(selected) == 0 {
		result := DAVSyncResult{OK: false, Status: "error", Message: "Bitte wähle zuerst mindestens eine DAV-Collection aus."}
		_ = s.Store.UpdateCalDAVConnectionSync(user.ID, result.Status, result.Message)
		return result, nil
	}
	result := DAVSyncResult{OK: true, Status: "ok"}
	forceLocal := input.ConflictStrategy == "local"
	remoteOnly := input.ConflictStrategy == "remote"
	shouldPull := connection.SyncDirection != "push" || remoteOnly
	shouldPush := connection.SyncDirection != "pull" && !remoteOnly
	if shouldPush {
		if err := s.exportDAVDeletions(user, connection, selected, password, forceLocal, &result); err != nil {
			result.Warnings = append(result.Warnings, err.Error())
		}
	}
	if shouldPull {
		for _, collection := range selected {
			switch collection.Kind {
			case "addressbook":
				if connection.SyncContacts {
					if err := s.syncAddressbookCollection(user, connection, collection, password, &result); err != nil {
						result.Warnings = append(result.Warnings, err.Error())
					}
				}
			case "calendar":
				if connection.SyncEvents && collection.SupportsEvents {
					if err := s.syncCalendarEvents(user, connection, collection, password, &result); err != nil {
						result.Warnings = append(result.Warnings, err.Error())
					}
				}
				if connection.SyncTasks && collection.SupportsTasks {
					if err := s.syncCalendarTasks(user, connection, collection, password, &result); err != nil {
						result.Warnings = append(result.Warnings, err.Error())
					}
				}
			}
		}
	}
	if shouldPush {
		if err := s.exportDAVData(user, connection, selected, password, forceLocal, &result); err != nil {
			result.Warnings = append(result.Warnings, err.Error())
		}
	}
	changed := result.EventsImported + result.EventsUpdated + result.TasksImported + result.TasksUpdated + result.ContactsImported + result.ContactsUpdated + result.EventsExported + result.TasksExported + result.ContactsExported + result.EventsDeleted + result.TasksDeleted + result.ContactsDeleted
	if len(result.Warnings) > 0 && changed == 0 {
		result.OK = false
		result.Status = "error"
		result.Message = "Synchronisierung konnte nicht abgeschlossen werden."
	} else {
		result.Message = fmt.Sprintf("Synchronisierung abgeschlossen: %d Termine, %d Tasks, %d Kontakte.", result.EventsImported+result.EventsUpdated+result.EventsExported+result.EventsDeleted, result.TasksImported+result.TasksUpdated+result.TasksExported+result.TasksDeleted, result.ContactsImported+result.ContactsUpdated+result.ContactsExported+result.ContactsDeleted)
	}
	_ = s.Store.UpdateCalDAVConnectionSync(user.ID, result.Status, result.Message)
	return result, nil
}

func (s *CalDAVService) exportDAVDeletions(user domain.User, connection domain.CalDAVConnection, collections []domain.DAVCollection, password string, force bool, result *DAVSyncResult) error {
	for _, collection := range collections {
		switch collection.Kind {
		case "addressbook":
			if connection.SyncContacts {
				s.exportDeletedDAVItems(user, connection, collection, password, "contact", force, result)
			}
		case "calendar":
			if connection.SyncEvents && collection.SupportsEvents {
				s.exportDeletedDAVItems(user, connection, collection, password, "event", force, result)
			}
			if connection.SyncTasks && collection.SupportsTasks {
				s.exportDeletedDAVItems(user, connection, collection, password, "task", force, result)
			}
		}
	}
	return nil
}

func (s *CalDAVService) exportDeletedDAVItems(user domain.User, connection domain.CalDAVConnection, collection domain.DAVCollection, password, kind string, force bool, result *DAVSyncResult) {
	items, err := s.Store.ListDAVSyncItemsForCollection(user.ID, collection.URL, kind)
	if err != nil {
		result.Warnings = append(result.Warnings, err.Error())
		return
	}
	for _, item := range items {
		if s.localDAVItemExists(user.ID, item.LocalID, kind) {
			continue
		}
		err := s.davDelete(item.ResourceURL, connection.Username, password, item.ETag)
		if force && isDAVConflict(err) {
			err = s.davDelete(item.ResourceURL, connection.Username, password, "")
		}
		if err != nil {
			result.Warnings = append(result.Warnings, davWarningMessage(item.ResourceURL, err))
			continue
		}
		_ = s.Store.DeleteDAVSyncItem(user.ID, item.ID)
		switch kind {
		case "event":
			result.EventsDeleted++
		case "task":
			result.TasksDeleted++
		case "contact":
			result.ContactsDeleted++
		}
	}
}

func (s *CalDAVService) localDAVItemExists(userID, localID int64, kind string) bool {
	var err error
	switch kind {
	case "event":
		_, err = s.Store.FindEventByID(localID, userID)
	case "task":
		_, err = s.Store.FindTaskByID(localID, userID)
	case "contact":
		_, err = s.Store.FindContactByID(localID, userID)
	default:
		return true
	}
	if err == nil {
		return true
	}
	return !errors.Is(err, sqlite.ErrNotFound)
}

func (s *CalDAVService) exportDAVData(user domain.User, connection domain.CalDAVConnection, collections []domain.DAVCollection, password string, force bool, result *DAVSyncResult) error {
	eventCollection, taskCollection, contactCollection := pickDAVExportTargets(collections)
	collectionByURL := mapDAVCollectionsByURL(collections)
	eventCollectionByCalendar := mapEventCollectionsByCalendar(collections)
	if connection.SyncEvents && eventCollection.URL != "" {
		calendars, err := s.Store.ListCalendars(user.ID)
		if err != nil {
			return err
		}
		calendarNames := make(map[int64]string, len(calendars))
		for _, calendar := range calendars {
			calendarNames[calendar.ID] = calendar.Name
		}
		events, err := s.Store.ListEvents(sqlite.EventFilter{UserID: user.ID, From: time.Now().UTC().AddDate(0, 0, -connection.SyncWindowPastDays), To: time.Now().UTC().AddDate(0, 0, connection.SyncWindowFutureDays), Limit: 500})
		if err != nil {
			return err
		}
		for _, event := range events {
			target := eventCollection
			if matched := eventCollectionByCalendar[normalizedDAVName(calendarNames[event.CalendarID])]; matched.URL != "" {
				target = matched
			}
			syncItem, hasSyncItem := s.findDAVSyncItemByLocalID(user.ID, event.ID, "event")
			if hasSyncItem {
				if existingCollection := collectionByURL[syncItem.CollectionURL]; existingCollection.URL != "" {
					target = existingCollection
				}
				if !shouldExportDAVItem(event.UpdatedAt, syncItem) {
					result.Skipped++
					continue
				}
			}
			uid := event.UID
			if hasSyncItem && syncItem.UID != "" {
				uid = syncItem.UID
			}
			if uid == "" {
				uid = fmt.Sprintf("event-%d@calendaradvanced.local", event.ID)
			}
			event.UID = uid
			resourceURL := syncItem.ResourceURL
			if resourceURL == "" {
				resourceURL = davResourceURL(target.URL, uid, ".ics")
			}
			etag, err := s.davPut(resourceURL, connection.Username, password, "text/calendar; charset=utf-8", calendarEventData(event), syncItem.ETag, !hasSyncItem)
			if force && isDAVConflict(err) {
				etag, err = s.davPut(resourceURL, connection.Username, password, "text/calendar; charset=utf-8", calendarEventData(event), "", false)
			}
			if err != nil {
				result.Warnings = append(result.Warnings, davWarningMessage(fmt.Sprintf("Termin %q (ID %d)", event.Title, event.ID), err))
				continue
			}
			_ = s.Store.UpsertDAVSyncItem(domain.DAVSyncItem{UserID: user.ID, CollectionURL: target.URL, ResourceURL: resourceURL, Kind: "event", LocalID: event.ID, UID: uid, ETag: etag})
			result.EventsExported++
		}
	}
	if connection.SyncTasks && taskCollection.URL != "" {
		tasks, err := s.Store.ListTasks(sqlite.TaskFilter{UserID: user.ID, Limit: 500})
		if err != nil {
			return err
		}
		for _, task := range tasks {
			uid := fmt.Sprintf("task-%d@calendaradvanced.local", task.ID)
			target := taskCollection
			syncItem, hasSyncItem := s.findDAVSyncItemByLocalID(user.ID, task.ID, "task")
			if hasSyncItem {
				if existingCollection := collectionByURL[syncItem.CollectionURL]; existingCollection.URL != "" {
					target = existingCollection
				}
				if !shouldExportDAVItem(task.UpdatedAt, syncItem) {
					result.Skipped++
					continue
				}
				if syncItem.UID != "" {
					uid = syncItem.UID
				}
			}
			resourceURL := syncItem.ResourceURL
			if resourceURL == "" {
				resourceURL = davResourceURL(target.URL, uid, ".ics")
			}
			etag, err := s.davPut(resourceURL, connection.Username, password, "text/calendar; charset=utf-8", calendarTaskData(uid, task), syncItem.ETag, !hasSyncItem)
			if force && isDAVConflict(err) {
				etag, err = s.davPut(resourceURL, connection.Username, password, "text/calendar; charset=utf-8", calendarTaskData(uid, task), "", false)
			}
			if err != nil {
				result.Warnings = append(result.Warnings, davWarningMessage(fmt.Sprintf("Task %q (ID %d)", task.Title, task.ID), err))
				continue
			}
			_ = s.Store.UpsertDAVSyncItem(domain.DAVSyncItem{UserID: user.ID, CollectionURL: target.URL, ResourceURL: resourceURL, Kind: "task", LocalID: task.ID, UID: uid, ETag: etag})
			result.TasksExported++
		}
	}
	if connection.SyncContacts && contactCollection.URL != "" {
		contacts, err := s.Store.ListContacts(sqlite.ContactFilter{UserID: user.ID, Limit: 500})
		if err != nil {
			return err
		}
		for _, contact := range contacts {
			uid := fmt.Sprintf("contact-%d@calendaradvanced.local", contact.ID)
			target := contactCollection
			syncItem, hasSyncItem := s.findDAVSyncItemByLocalID(user.ID, contact.ID, "contact")
			if hasSyncItem {
				if existingCollection := collectionByURL[syncItem.CollectionURL]; existingCollection.URL != "" {
					target = existingCollection
				}
				if !shouldExportDAVItem(contact.UpdatedAt, syncItem) {
					result.Skipped++
					continue
				}
				if syncItem.UID != "" {
					uid = syncItem.UID
				}
			}
			resourceURL := syncItem.ResourceURL
			if resourceURL == "" {
				resourceURL = davResourceURL(target.URL, uid, ".vcf")
			}
			etag, err := s.davPut(resourceURL, connection.Username, password, "text/vcard; charset=utf-8", vCardData(uid, contact), syncItem.ETag, !hasSyncItem)
			if force && isDAVConflict(err) {
				etag, err = s.davPut(resourceURL, connection.Username, password, "text/vcard; charset=utf-8", vCardData(uid, contact), "", false)
			}
			if err != nil {
				result.Warnings = append(result.Warnings, davWarningMessage(fmt.Sprintf("Kontakt %q (ID %d)", contactDisplayName(contact), contact.ID), err))
				continue
			}
			_ = s.Store.UpsertDAVSyncItem(domain.DAVSyncItem{UserID: user.ID, CollectionURL: target.URL, ResourceURL: resourceURL, Kind: "contact", LocalID: contact.ID, UID: uid, ETag: etag})
			result.ContactsExported++
		}
	}
	return nil
}

func pickDAVExportTargets(collections []domain.DAVCollection) (domain.DAVCollection, domain.DAVCollection, domain.DAVCollection) {
	var eventCollection domain.DAVCollection
	var taskCollection domain.DAVCollection
	var contactCollection domain.DAVCollection
	for _, collection := range collections {
		if collection.Kind == "addressbook" && contactCollection.URL == "" {
			contactCollection = collection
		}
		if collection.Kind != "calendar" {
			continue
		}
		if collection.SupportsEvents && eventCollection.URL == "" {
			eventCollection = collection
		}
		if collection.SupportsTasks && taskCollection.URL == "" {
			taskCollection = collection
		}
	}
	return eventCollection, taskCollection, contactCollection
}

func mapDAVCollectionsByURL(collections []domain.DAVCollection) map[string]domain.DAVCollection {
	out := make(map[string]domain.DAVCollection, len(collections))
	for _, collection := range collections {
		out[collection.URL] = collection
	}
	return out
}

func mapEventCollectionsByCalendar(collections []domain.DAVCollection) map[string]domain.DAVCollection {
	out := map[string]domain.DAVCollection{}
	for _, collection := range collections {
		if collection.Kind == "calendar" && collection.SupportsEvents {
			out[normalizedDAVName(collection.DisplayName)] = collection
		}
	}
	return out
}

func (s *CalDAVService) findDAVSyncItemByLocalID(userID, localID int64, kind string) (domain.DAVSyncItem, bool) {
	item, err := s.Store.FindDAVSyncItemByLocalID(userID, localID, kind)
	return item, err == nil
}

func (s *CalDAVService) findDAVSyncItem(userID int64, resourceURL, kind string) (domain.DAVSyncItem, bool) {
	normalized := normalizeDAVResourceURL(resourceURL)
	if item, err := s.Store.FindDAVSyncItem(userID, normalized, kind); err == nil {
		return item, true
	}
	if normalized != resourceURL {
		if item, err := s.Store.FindDAVSyncItem(userID, resourceURL, kind); err == nil {
			return item, true
		}
	}
	if item, err := s.Store.FindDAVSyncItem(userID, normalized+"/", kind); err == nil {
		return item, true
	}
	return domain.DAVSyncItem{}, false
}

func shouldExportDAVItem(localUpdated time.Time, syncItem domain.DAVSyncItem) bool {
	if syncItem.ID == 0 || syncItem.UpdatedAt.IsZero() {
		return true
	}
	return localUpdated.After(syncItem.UpdatedAt)
}

func normalizedDAVName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *CalDAVService) davPut(resourceURL, username, password, contentType, body, etag string, createOnly bool) (string, error) {
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, resourceURL, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "CalendarAdvanced DAV")
	if etag != "" {
		req.Header.Set("If-Match", etag)
	} else if createOnly {
		req.Header.Set("If-None-Match", "*")
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", NewError("dav_export_failed", "DAV-Server ist nicht erreichbar oder antwortet nicht.", nil)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode == http.StatusPreconditionFailed {
		return "", NewError("dav_export_conflict", "DAV-Ressource wurde seit dem letzten Sync verändert. Bitte zuerst importieren und erneut synchronisieren.", nil)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return "", NewError("dav_export_failed", fmt.Sprintf("DAV-Server antwortet beim Export mit Status %d.", resp.StatusCode), nil)
	}
	return strings.TrimSpace(resp.Header.Get("ETag")), nil
}

func (s *CalDAVService) davDelete(resourceURL, username, password, etag string) error {
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, normalizeDAVResourceURL(resourceURL), nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("User-Agent", "CalendarAdvanced DAV")
	if etag != "" {
		req.Header.Set("If-Match", etag)
	}
	resp, err := client.Do(req)
	if err != nil {
		return NewError("dav_delete_failed", "DAV-Server ist nicht erreichbar oder antwortet nicht.", nil)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent, http.StatusNotFound, http.StatusGone:
		return nil
	case http.StatusPreconditionFailed:
		return NewError("dav_delete_conflict", "DAV-Ressource wurde seit dem letzten Sync verändert. Bitte zuerst importieren und erneut synchronisieren.", nil)
	default:
		return NewError("dav_delete_failed", fmt.Sprintf("DAV-Server antwortet beim Löschen mit Status %d.", resp.StatusCode), nil)
	}
}

func isDAVConflict(err error) bool {
	var coded CodedError
	return errors.As(err, &coded) && (coded.Code == "dav_export_conflict" || coded.Code == "dav_delete_conflict")
}

func davWarningMessage(label string, err error) string {
	message := err.Error()
	var coded CodedError
	if errors.As(err, &coded) && coded.Message != "" {
		message = coded.Message
	}
	if strings.TrimSpace(label) == "" {
		return message
	}
	return strings.TrimSpace(label) + ": " + message
}

type davResource struct {
	URL  string
	ETag string
	Data string
}

type davTask struct {
	UID         string
	Title       string
	Description string
	DueAt       time.Time
	Priority    domain.TaskPriority
	Completed   bool
}

func (s *CalDAVService) syncCalendarEvents(user domain.User, connection domain.CalDAVConnection, collection domain.DAVCollection, password string, result *DAVSyncResult) error {
	rangeStart := time.Now().UTC().AddDate(0, 0, -connection.SyncWindowPastDays)
	rangeEnd := time.Now().UTC().AddDate(0, 0, connection.SyncWindowFutureDays)
	resources, err := s.reportCalendarCollection(collection.URL, connection.Username, password, "VEVENT", rangeStart, rangeEnd)
	if err != nil {
		return fmt.Errorf("%s: %w", collection.DisplayName, err)
	}
	seenResources := make(map[string]bool, len(resources))
	calendar, err := s.ensureCalendar(user, collection)
	if err != nil {
		return err
	}
	for _, resource := range resources {
		seenResources[normalizeDAVResourceURL(resource.URL)] = true
		if strings.TrimSpace(resource.Data) == "" {
			result.Skipped++
			continue
		}
		parsed, err := ics.ParseCalendar([]byte(resource.Data), calendar.Timezone)
		if err != nil {
			result.Skipped++
			continue
		}
		for _, item := range parsed.Events {
			if strings.TrimSpace(item.UID) == "" {
				item.UID = resource.URL
			}
			if syncItem, ok := s.findDAVSyncItem(user.ID, resource.URL, "event"); ok && syncItem.ETag == resource.ETag && resource.ETag != "" {
				if _, err := s.Store.FindEventByID(syncItem.LocalID, user.ID); err == nil {
					result.Skipped++
					continue
				}
			}
			event := domain.Event{
				CalendarID:  calendar.ID,
				UID:         item.UID,
				Title:       item.Title,
				Description: item.Description,
				Location:    item.Location,
				StartsAt:    item.StartsAt,
				EndsAt:      item.EndsAt,
				Timezone:    valueOrDefaultString(item.Timezone, calendar.Timezone),
				AllDay:      item.AllDay,
				Status:      domain.EventStatusConfirmed,
				CreatedBy:   user.ID,
				Reminders:   remindersFromDAVMinutes(item.ReminderMin),
			}
			var localID int64
			if syncItem, ok := s.findDAVSyncItem(user.ID, resource.URL, "event"); ok {
				event.ID = syncItem.LocalID
				if updated, updateErr := s.Store.UpdateEvent(event, user.ID); updateErr == nil {
					localID = updated.ID
					result.EventsUpdated++
				} else if errors.Is(updateErr, sqlite.ErrNotFound) {
					if restored, restoreErr := s.Store.RestoreEvent(event, user.ID); restoreErr == nil {
						localID = restored.ID
						result.EventsUpdated++
					}
				}
			}
			if localID == 0 {
				if existing, err := s.Store.FindEventByUID(item.UID, user.ID); err == nil {
					event.ID = existing.ID
					if updated, updateErr := s.Store.UpdateEvent(event, user.ID); updateErr == nil {
						localID = updated.ID
						result.EventsUpdated++
					}
				} else if existing, err := s.Store.FindEventByUIDIncludingDeleted(item.UID, user.ID); err == nil {
					event.ID = existing.ID
					if restored, restoreErr := s.Store.RestoreEvent(event, user.ID); restoreErr == nil {
						localID = restored.ID
						result.EventsUpdated++
					}
				}
			}
			if localID == 0 {
				created, createErr := s.Store.CreateEvent(event)
				if createErr != nil {
					result.Skipped++
					continue
				}
				localID = created.ID
				result.EventsImported++
			}
			_ = s.Store.UpsertDAVSyncItem(domain.DAVSyncItem{UserID: user.ID, CollectionURL: collection.URL, ResourceURL: resource.URL, Kind: "event", LocalID: localID, UID: item.UID, ETag: resource.ETag})
		}
	}
	s.deleteMissingRemoteEvents(user, collection, seenResources, rangeStart, rangeEnd, result)
	return nil
}

func (s *CalDAVService) syncCalendarTasks(user domain.User, connection domain.CalDAVConnection, collection domain.DAVCollection, password string, result *DAVSyncResult) error {
	resources, err := s.reportCalendarCollection(collection.URL, connection.Username, password, "VTODO", time.Time{}, time.Time{})
	if err != nil {
		return fmt.Errorf("%s: %w", collection.DisplayName, err)
	}
	seenResources := make(map[string]bool, len(resources))
	for _, resource := range resources {
		seenResources[normalizeDAVResourceURL(resource.URL)] = true
		tasks := parseVTODOs(resource.Data)
		if len(tasks) == 0 {
			result.Skipped++
			continue
		}
		for _, item := range tasks {
			if item.Title == "" {
				item.Title = "Ohne Titel"
			}
			if strings.TrimSpace(item.UID) == "" {
				item.UID = resource.URL
			}
			if syncItem, ok := s.findDAVSyncItem(user.ID, resource.URL, "task"); ok && syncItem.ETag == resource.ETag && resource.ETag != "" {
				result.Skipped++
				continue
			}
			task := domain.Task{
				UserID:         user.ID,
				Title:          item.Title,
				Description:    item.Description,
				DueAt:          item.DueAt,
				Priority:       item.Priority,
				Completed:      item.Completed,
				ShowInCalendar: !item.DueAt.IsZero(),
			}
			var localID int64
			if syncItem, ok := s.findDAVSyncItem(user.ID, resource.URL, "task"); ok {
				task.ID = syncItem.LocalID
				if updated, updateErr := s.Store.UpdateTask(task, user.ID); updateErr == nil {
					localID = updated.ID
					result.TasksUpdated++
				}
			}
			if localID == 0 {
				created, createErr := s.Store.CreateTask(task)
				if createErr != nil {
					result.Skipped++
					continue
				}
				localID = created.ID
				result.TasksImported++
			}
			_ = s.Store.UpsertDAVSyncItem(domain.DAVSyncItem{UserID: user.ID, CollectionURL: collection.URL, ResourceURL: resource.URL, Kind: "task", LocalID: localID, UID: item.UID, ETag: resource.ETag})
		}
	}
	s.deleteMissingRemoteItems(user, collection, "task", seenResources, result)
	return nil
}

func (s *CalDAVService) syncAddressbookCollection(user domain.User, connection domain.CalDAVConnection, collection domain.DAVCollection, password string, result *DAVSyncResult) error {
	resources, err := s.reportAddressbookCollection(collection.URL, connection.Username, password)
	if err != nil {
		return fmt.Errorf("%s: %w", collection.DisplayName, err)
	}
	seenResources := make(map[string]bool, len(resources))
	for _, resource := range resources {
		seenResources[normalizeDAVResourceURL(resource.URL)] = true
		if strings.TrimSpace(resource.Data) == "" {
			result.Skipped++
			continue
		}
		if syncItem, ok := s.findDAVSyncItem(user.ID, resource.URL, "contact"); ok && syncItem.ETag == resource.ETag && resource.ETag != "" {
			result.Skipped++
			continue
		}
		contact := contactFromVCard(resource.Data, user.ID)
		if strings.TrimSpace(contact.FirstName) == "" && strings.TrimSpace(contact.LastName) == "" && strings.TrimSpace(contact.Company) == "" {
			result.Skipped++
			continue
		}
		var localID int64
		if syncItem, ok := s.findDAVSyncItem(user.ID, resource.URL, "contact"); ok {
			contact.ID = syncItem.LocalID
			if updated, updateErr := s.Store.UpdateContact(contact, user.ID); updateErr == nil {
				localID = updated.ID
				result.ContactsUpdated++
			}
		}
		if localID == 0 {
			created, createErr := s.Store.CreateContact(contact)
			if createErr != nil {
				result.Skipped++
				continue
			}
			localID = created.ID
			result.ContactsImported++
		}
		_ = s.Store.UpsertDAVSyncItem(domain.DAVSyncItem{UserID: user.ID, CollectionURL: collection.URL, ResourceURL: resource.URL, Kind: "contact", LocalID: localID, UID: vCardUID(resource.Data), ETag: resource.ETag})
	}
	s.deleteMissingRemoteItems(user, collection, "contact", seenResources, result)
	return nil
}

func (s *CalDAVService) deleteMissingRemoteEvents(user domain.User, collection domain.DAVCollection, seenResources map[string]bool, rangeStart, rangeEnd time.Time, result *DAVSyncResult) {
	items, err := s.Store.ListDAVSyncItemsForCollection(user.ID, collection.URL, "event")
	if err != nil {
		result.Warnings = append(result.Warnings, err.Error())
		return
	}
	for _, item := range items {
		if seenResources[normalizeDAVResourceURL(item.ResourceURL)] {
			continue
		}
		event, err := s.Store.FindEventByID(item.LocalID, user.ID)
		if err != nil {
			_ = s.Store.DeleteDAVSyncItem(user.ID, item.ID)
			continue
		}
		if event.EndsAt.Before(rangeStart) || event.StartsAt.After(rangeEnd) {
			continue
		}
		if err := s.Store.DeleteEvent(item.LocalID, user.ID); err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}
		_ = s.Store.DeleteDAVSyncItem(user.ID, item.ID)
		result.EventsDeleted++
	}
}

func (s *CalDAVService) deleteMissingRemoteItems(user domain.User, collection domain.DAVCollection, kind string, seenResources map[string]bool, result *DAVSyncResult) {
	items, err := s.Store.ListDAVSyncItemsForCollection(user.ID, collection.URL, kind)
	if err != nil {
		result.Warnings = append(result.Warnings, err.Error())
		return
	}
	for _, item := range items {
		if seenResources[normalizeDAVResourceURL(item.ResourceURL)] {
			continue
		}
		var deleteErr error
		switch kind {
		case "task":
			deleteErr = s.Store.DeleteTask(item.LocalID, user.ID)
		case "contact":
			deleteErr = s.Store.DeleteContact(item.LocalID, user.ID)
		}
		if deleteErr != nil && !errors.Is(deleteErr, sqlite.ErrNotFound) {
			result.Warnings = append(result.Warnings, deleteErr.Error())
			continue
		}
		_ = s.Store.DeleteDAVSyncItem(user.ID, item.ID)
		switch kind {
		case "task":
			result.TasksDeleted++
		case "contact":
			result.ContactsDeleted++
		}
	}
}

func (s *CalDAVService) ensureCalendar(user domain.User, collection domain.DAVCollection) (domain.Calendar, error) {
	calendars, err := s.Store.ListCalendars(user.ID)
	if err != nil {
		return domain.Calendar{}, err
	}
	name := strings.TrimSpace(collection.DisplayName)
	if name == "" {
		name = fallbackDAVName(collection.URL)
	}
	for _, calendar := range calendars {
		if strings.EqualFold(calendar.Name, name) {
			return calendar, nil
		}
	}
	return s.Store.CreateCalendar(domain.Calendar{
		OwnerUserID:         user.ID,
		Name:                name,
		Color:               "#6d8cff",
		Timezone:            "Europe/Berlin",
		Visible:             true,
		ReminderTime:        "09:00",
		SameDayReminderTime: "09:00",
	})
}

func (s *CalDAVService) reportCalendarCollection(collectionURL, username, password, component string, from, to time.Time) ([]davResource, error) {
	filter := fmt.Sprintf(`<cal:comp-filter name="VCALENDAR"><cal:comp-filter name="%s">`, component)
	if component == "VEVENT" && !from.IsZero() && !to.IsZero() {
		filter += fmt.Sprintf(`<cal:time-range start="%s" end="%s"/>`, from.UTC().Format("20060102T150405Z"), to.UTC().Format("20060102T150405Z"))
	}
	filter += `</cal:comp-filter></cal:comp-filter>`
	body := `<?xml version="1.0" encoding="utf-8"?><cal:calendar-query xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav"><d:prop><d:getetag/><cal:calendar-data/></d:prop><cal:filter>` + filter + `</cal:filter></cal:calendar-query>`
	return s.davReport(collectionURL, username, password, body, "calendar")
}

func (s *CalDAVService) reportAddressbookCollection(collectionURL, username, password string) ([]davResource, error) {
	body := `<?xml version="1.0" encoding="utf-8"?><card:addressbook-query xmlns:d="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav"><d:prop><d:getetag/><card:address-data/></d:prop></card:addressbook-query>`
	return s.davReport(collectionURL, username, password, body, "addressbook")
}

func (s *CalDAVService) davReport(collectionURL, username, password, body, dataKind string) ([]davResource, error) {
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "REPORT", collectionURL, bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("User-Agent", "CalendarAdvanced DAV")
	resp, err := client.Do(req)
	if err != nil {
		return nil, NewError("dav_sync_failed", "DAV-Server ist nicht erreichbar oder antwortet nicht.", nil)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, NewError("dav_auth_failed", "Anmeldung fehlgeschlagen. Bitte Username/E-Mail und Passwort prüfen.", nil)
	}
	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, NewError("dav_sync_failed", fmt.Sprintf("DAV-Server antwortet mit Status %d.", resp.StatusCode), nil)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	var multistatus davPropfindResponse
	if err := xml.Unmarshal(data, &multistatus); err != nil {
		return nil, NewError("dav_sync_failed", "DAV-Antwort konnte nicht gelesen werden.", nil)
	}
	resources := make([]davResource, 0, len(multistatus.Responses))
	for _, response := range multistatus.Responses {
		prop := firstDAVProp(response)
		payload := prop.CalendarData
		if dataKind == "addressbook" {
			payload = prop.AddressData
		}
		if strings.TrimSpace(payload) == "" {
			continue
		}
		resources = append(resources, davResource{URL: resolveDAVResourceHref(collectionURL, response.Href), ETag: strings.TrimSpace(prop.ETag), Data: payload})
	}
	return resources, nil
}

func parseVTODOs(data string) []davTask {
	lines := unfoldDAVLines(data)
	tasks := []davTask{}
	var current *davTask
	for _, raw := range lines {
		name, params, value := parseDAVLine(raw)
		switch name {
		case "BEGIN":
			if strings.EqualFold(value, "VTODO") {
				current = &davTask{Priority: domain.TaskPriorityNormal}
			}
		case "END":
			if strings.EqualFold(value, "VTODO") && current != nil {
				tasks = append(tasks, *current)
				current = nil
			}
		default:
			if current == nil {
				continue
			}
			switch name {
			case "UID":
				current.UID = strings.TrimSpace(value)
			case "SUMMARY":
				current.Title = unescapeDAVText(value)
			case "DESCRIPTION":
				current.Description = unescapeDAVText(value)
			case "DUE", "DTSTART":
				if current.DueAt.IsZero() {
					current.DueAt = parseDAVDateTime(value, params)
				}
			case "STATUS":
				current.Completed = strings.EqualFold(value, "COMPLETED")
			case "COMPLETED":
				current.Completed = strings.TrimSpace(value) != ""
			case "PRIORITY":
				current.Priority = taskPriorityFromDAV(value)
			}
		}
	}
	return tasks
}

func contactFromVCard(data string, userID int64) domain.Contact {
	props := parsePropertyMap(data)
	first, last := "", ""
	if n := props["N"]; n != "" {
		parts := strings.Split(n, ";")
		if len(parts) > 0 {
			last = unescapeDAVText(parts[0])
		}
		if len(parts) > 1 {
			first = unescapeDAVText(parts[1])
		}
	}
	if first == "" && last == "" {
		first, last = splitDisplayName(unescapeDAVText(props["FN"]))
	}
	contact := domain.Contact{
		UserID:    userID,
		FirstName: first,
		LastName:  last,
		Company:   unescapeDAVText(props["ORG"]),
		Email:     props["EMAIL"],
		Phone:     props["TEL"],
		Mobile:    props["TEL_CELL"],
		Address:   unescapeDAVText(props["ADR"]),
		Birthday:  normalizeBirthday(props["BDAY"]),
		Notes:     unescapeDAVText(props["NOTE"]),
	}
	if contact.Mobile == "" {
		contact.Mobile = props["TEL_MOBILE"]
	}
	if contact.Phone == contact.Mobile {
		contact.Phone = ""
	}
	return contact
}

func parsePropertyMap(data string) map[string]string {
	props := map[string]string{}
	for _, raw := range unfoldDAVLines(data) {
		name, params, value := parseDAVLine(raw)
		switch name {
		case "EMAIL":
			if props["EMAIL"] == "" {
				props["EMAIL"] = strings.TrimSpace(value)
			}
		case "TEL":
			paramText := strings.ToUpper(strings.Join(params, " "))
			if strings.Contains(paramText, "CELL") || strings.Contains(paramText, "MOBILE") {
				if props["TEL_CELL"] == "" {
					props["TEL_CELL"] = strings.TrimSpace(value)
				}
			} else if props["TEL"] == "" {
				props["TEL"] = strings.TrimSpace(value)
			}
		case "FN", "N", "ORG", "ADR", "BDAY", "NOTE", "UID":
			if props[name] == "" {
				props[name] = strings.TrimSpace(value)
			}
		}
	}
	return props
}

func vCardUID(data string) string {
	return parsePropertyMap(data)["UID"]
}

func unfoldDAVLines(data string) []string {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	lines := strings.Split(data, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(out) > 0 {
			out[len(out)-1] += strings.TrimLeft(line, " \t")
			continue
		}
		out = append(out, line)
	}
	return out
}

func parseDAVLine(line string) (string, []string, string) {
	left, value, _ := strings.Cut(line, ":")
	parts := strings.Split(left, ";")
	name := strings.ToUpper(parts[0])
	params := []string{}
	for _, part := range parts[1:] {
		params = append(params, strings.ToUpper(strings.Trim(part, `"`)))
	}
	return name, params, value
}

func parseDAVDateTime(value string, params []string) time.Time {
	value = strings.TrimSpace(value)
	for _, param := range params {
		if strings.Contains(param, "VALUE=DATE") {
			if parsed, err := time.ParseInLocation("20060102", value, time.Local); err == nil {
				return parsed
			}
		}
	}
	if len(value) == 8 && !strings.Contains(value, "T") {
		parsed, _ := time.ParseInLocation("20060102", value, time.Local)
		return parsed
	}
	if strings.HasSuffix(value, "Z") {
		parsed, _ := time.Parse("20060102T150405Z", value)
		return parsed
	}
	parsed, _ := time.ParseInLocation("20060102T150405", value, time.Local)
	return parsed
}

func taskPriorityFromDAV(value string) domain.TaskPriority {
	switch strings.TrimSpace(value) {
	case "1", "2", "3", "4":
		return domain.TaskPriorityHigh
	case "7", "8", "9":
		return domain.TaskPriorityLow
	default:
		return domain.TaskPriorityNormal
	}
}

func remindersFromDAVMinutes(items []int) []domain.Reminder {
	reminders := make([]domain.Reminder, 0, len(items))
	seen := map[int]bool{}
	for _, minutes := range items {
		if !seen[minutes] {
			reminders = append(reminders, domain.Reminder{MinutesBefore: minutes})
			seen[minutes] = true
		}
	}
	return reminders
}

func splitDisplayName(name string) (string, string) {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return strings.Join(parts[:len(parts)-1], " "), parts[len(parts)-1]
}

func normalizeBirthday(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.Format("2006-01-02")
	}
	if parsed, err := time.Parse("20060102", value); err == nil {
		return parsed.Format("2006-01-02")
	}
	if birthdayPattern.MatchString(value) {
		return value
	}
	return ""
}

var birthdayPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func unescapeDAVText(value string) string {
	value = strings.ReplaceAll(value, `\n`, "\n")
	value = strings.ReplaceAll(value, `\N`, "\n")
	value = strings.ReplaceAll(value, `\,`, ",")
	value = strings.ReplaceAll(value, `\;`, ";")
	value = strings.ReplaceAll(value, `\\`, `\`)
	return strings.TrimSpace(value)
}

func calendarEventData(event domain.Event) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//CalendarAdvanced//DAV Sync//DE\r\nBEGIN:VEVENT\r\n")
	b.WriteString("UID:" + escapeDAVText(event.UID) + "\r\n")
	b.WriteString("DTSTAMP:" + formatDAVDateTime(time.Now().UTC()) + "\r\n")
	if event.AllDay {
		b.WriteString("DTSTART;VALUE=DATE:" + formatDAVAllDayDate(event.StartsAt, event.Timezone) + "\r\n")
		b.WriteString("DTEND;VALUE=DATE:" + formatDAVAllDayDate(event.EndsAt, event.Timezone) + "\r\n")
	} else {
		b.WriteString("DTSTART:" + formatDAVDateTime(event.StartsAt.UTC()) + "\r\n")
		b.WriteString("DTEND:" + formatDAVDateTime(event.EndsAt.UTC()) + "\r\n")
	}
	b.WriteString("SUMMARY:" + escapeDAVText(event.Title) + "\r\n")
	if event.Description != "" {
		b.WriteString("DESCRIPTION:" + escapeDAVText(event.Description) + "\r\n")
	}
	if event.Location != "" {
		b.WriteString("LOCATION:" + escapeDAVText(event.Location) + "\r\n")
	}
	if event.Private {
		b.WriteString("CLASS:PRIVATE\r\n")
	}
	if event.Status == domain.EventStatusCancelled {
		b.WriteString("STATUS:CANCELLED\r\n")
	} else {
		b.WriteString("STATUS:CONFIRMED\r\n")
	}
	if event.Recurrence != nil && event.Recurrence.Frequency != "" {
		b.WriteString("RRULE:" + domain.RRULE(*event.Recurrence) + "\r\n")
	}
	b.WriteString("END:VEVENT\r\nEND:VCALENDAR\r\n")
	return b.String()
}

func calendarTaskData(uid string, task domain.Task) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//CalendarAdvanced//DAV Sync//DE\r\nBEGIN:VTODO\r\n")
	b.WriteString("UID:" + escapeDAVText(uid) + "\r\n")
	b.WriteString("DTSTAMP:" + formatDAVDateTime(time.Now().UTC()) + "\r\n")
	b.WriteString("SUMMARY:" + escapeDAVText(task.Title) + "\r\n")
	if task.Description != "" {
		b.WriteString("DESCRIPTION:" + escapeDAVText(task.Description) + "\r\n")
	}
	if !task.DueAt.IsZero() {
		b.WriteString("DUE:" + formatDAVDateTime(task.DueAt.UTC()) + "\r\n")
	}
	if task.Completed {
		b.WriteString("STATUS:COMPLETED\r\n")
		if !task.CompletedAt.IsZero() {
			b.WriteString("COMPLETED:" + formatDAVDateTime(task.CompletedAt.UTC()) + "\r\n")
		}
	} else {
		b.WriteString("STATUS:NEEDS-ACTION\r\n")
	}
	b.WriteString("PRIORITY:" + davTaskPriority(task.Priority) + "\r\n")
	b.WriteString("END:VTODO\r\nEND:VCALENDAR\r\n")
	return b.String()
}

func vCardData(uid string, contact domain.Contact) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCARD\r\nVERSION:3.0\r\n")
	b.WriteString("UID:" + escapeDAVText(uid) + "\r\n")
	b.WriteString("FN:" + escapeDAVText(contactDisplayName(contact)) + "\r\n")
	b.WriteString("N:" + escapeDAVText(contact.LastName) + ";" + escapeDAVText(contact.FirstName) + ";;;\r\n")
	if contact.Email != "" {
		b.WriteString("EMAIL;TYPE=HOME:" + escapeDAVText(contact.Email) + "\r\n")
	}
	if contact.Phone != "" {
		b.WriteString("TEL;TYPE=HOME,VOICE:" + escapeDAVText(contact.Phone) + "\r\n")
	}
	if contact.Mobile != "" {
		b.WriteString("TEL;TYPE=CELL:" + escapeDAVText(contact.Mobile) + "\r\n")
	}
	if contact.Company != "" {
		b.WriteString("ORG:" + escapeDAVText(contact.Company) + "\r\n")
	}
	if contact.CompanyEmail != "" {
		b.WriteString("EMAIL;TYPE=WORK:" + escapeDAVText(contact.CompanyEmail) + "\r\n")
	}
	if contact.CompanyPhone != "" {
		b.WriteString("TEL;TYPE=WORK,VOICE:" + escapeDAVText(contact.CompanyPhone) + "\r\n")
	}
	if contact.CompanyMobile != "" {
		b.WriteString("TEL;TYPE=WORK,CELL:" + escapeDAVText(contact.CompanyMobile) + "\r\n")
	}
	if contact.Birthday != "" {
		b.WriteString("BDAY:" + strings.ReplaceAll(contact.Birthday, "-", "") + "\r\n")
	}
	if contact.Address != "" {
		b.WriteString("ADR;TYPE=HOME:;;" + escapeDAVText(contact.Address) + ";;;;\r\n")
	}
	if contact.Notes != "" {
		b.WriteString("NOTE:" + escapeDAVText(contact.Notes) + "\r\n")
	}
	b.WriteString("END:VCARD\r\n")
	return b.String()
}

func contactDisplayName(contact domain.Contact) string {
	name := strings.TrimSpace(strings.Join([]string{contact.FirstName, contact.LastName}, " "))
	if name != "" {
		return name
	}
	if strings.TrimSpace(contact.Company) != "" {
		return strings.TrimSpace(contact.Company)
	}
	return "Kontakt"
}

func davResourceURL(collectionURL, uid, suffix string) string {
	return strings.TrimRight(collectionURL, "/") + "/" + url.PathEscape(uid) + suffix
}

func davTaskPriority(priority domain.TaskPriority) string {
	switch priority {
	case domain.TaskPriorityHigh:
		return "1"
	case domain.TaskPriorityLow:
		return "9"
	default:
		return "5"
	}
}

func formatDAVDateTime(value time.Time) string {
	return value.UTC().Format("20060102T150405Z")
}

func formatDAVAllDayDate(value time.Time, timezone string) string {
	location := time.Local
	if timezone != "" {
		if loaded, err := time.LoadLocation(timezone); err == nil {
			location = loaded
		}
	}
	return value.In(location).Format("20060102")
}

func escapeDAVText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, ";", `\;`)
	value = strings.ReplaceAll(value, ",", `\,`)
	return value
}

func valueOrDefaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (s *CalDAVService) connectionFromInput(user domain.User, input CalDAVConnectionInput, allowPlainPassword bool) (domain.CalDAVConnection, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = "CalDAV"
	}
	baseURL, err := normalizeCalDAVURL(input.BaseURL)
	if err != nil {
		return domain.CalDAVConnection{}, NewError("invalid_caldav_url", "CalDAV-URL ist ungültig.", nil)
	}
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return domain.CalDAVConnection{}, NewError("invalid_caldav_username", "Username/E-Mail ist erforderlich.", nil)
	}
	direction := input.SyncDirection
	if direction == "" {
		direction = "pull"
	}
	if direction != "pull" && direction != "push" && direction != "two_way" {
		return domain.CalDAVConnection{}, NewError("invalid_caldav_sync_direction", "Sync-Richtung ist ungültig.", nil)
	}
	interval := input.SyncIntervalMinutes
	if interval == 0 {
		interval = 60
	}
	if interval < 15 || interval > 1440 {
		return domain.CalDAVConnection{}, NewError("invalid_caldav_interval", "Sync-Intervall muss zwischen 15 und 1440 Minuten liegen.", nil)
	}
	pastDays := input.SyncWindowPastDays
	if pastDays == 0 {
		pastDays = 30
	}
	futureDays := input.SyncWindowFutureDays
	if futureDays == 0 {
		futureDays = 365
	}
	if pastDays < 0 || pastDays > 3650 || futureDays < 1 || futureDays > 3650 {
		return domain.CalDAVConnection{}, NewError("invalid_caldav_window", "Sync-Zeitraum ist ungültig.", nil)
	}
	passwordEncrypted := ""
	if strings.TrimSpace(input.Password) != "" {
		if s.Cipher == nil {
			return domain.CalDAVConnection{}, NewError("caldav_encryption_unavailable", "Passwort-Verschlüsselung ist nicht verfügbar.", nil)
		}
		passwordEncrypted, err = s.Cipher.Encrypt(input.Password)
		if err != nil {
			return domain.CalDAVConnection{}, err
		}
	} else if !allowPlainPassword {
		if existing, err := s.Store.FindCalDAVConnection(user.ID); err == nil {
			passwordEncrypted = existing.PasswordEncrypted
		} else {
			return domain.CalDAVConnection{}, NewError("caldav_password_required", "Passwort ist erforderlich.", nil)
		}
	}
	return domain.CalDAVConnection{
		UserID:               user.ID,
		DisplayName:          displayName,
		BaseURL:              baseURL,
		Username:             username,
		PasswordEncrypted:    passwordEncrypted,
		SyncEnabled:          input.SyncEnabled,
		SyncDirection:        direction,
		SyncEvents:           input.SyncEvents,
		SyncTasks:            input.SyncTasks,
		SyncContacts:         input.SyncContacts,
		SyncIntervalMinutes:  interval,
		SyncWindowPastDays:   pastDays,
		SyncWindowFutureDays: futureDays,
	}, nil
}

func (s *CalDAVService) decryptPassword(encrypted string) (string, error) {
	if strings.TrimSpace(encrypted) == "" {
		return "", NewError("caldav_password_required", "Passwort ist erforderlich.", nil)
	}
	if s.Cipher == nil {
		return "", NewError("caldav_encryption_unavailable", "Passwort-Verschlüsselung ist nicht verfügbar.", nil)
	}
	return s.Cipher.Decrypt(encrypted)
}

func (s *CalDAVService) probeConnection(baseURL, username, password string) CalDAVConnectionTestResult {
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	body := bytes.NewBufferString(`<?xml version="1.0" encoding="utf-8"?><d:propfind xmlns:d="DAV:"><d:prop><d:displayname/><d:resourcetype/><d:current-user-principal/></d:prop></d:propfind>`)
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", baseURL, body)
	if err != nil {
		return CalDAVConnectionTestResult{OK: false, Status: "error", Message: "Verbindungstest konnte nicht vorbereitet werden."}
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("User-Agent", "CalendarAdvanced CalDAV")
	resp, err := client.Do(req)
	if err != nil {
		return CalDAVConnectionTestResult{OK: false, Status: "error", Message: "Server ist nicht erreichbar oder antwortet nicht."}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	switch {
	case resp.StatusCode == http.StatusMultiStatus || resp.StatusCode == http.StatusOK:
		return CalDAVConnectionTestResult{OK: true, Status: "ok", Message: "Verbindung erfolgreich.", StatusCode: resp.StatusCode, CalendarURL: baseURL}
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return CalDAVConnectionTestResult{OK: false, Status: "auth_failed", Message: "Anmeldung fehlgeschlagen. Bitte Username/E-Mail und Passwort prüfen.", StatusCode: resp.StatusCode}
	default:
		return CalDAVConnectionTestResult{OK: false, Status: "error", Message: fmt.Sprintf("Server antwortet mit Status %d.", resp.StatusCode), StatusCode: resp.StatusCode}
	}
}

func (s *CalDAVService) discoverDAVCollections(baseURL, username, password string) ([]domain.DAVCollection, error) {
	items, err := s.propfindCollections(baseURL, username, password)
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return items, nil
	}
	usernameURL := strings.TrimRight(baseURL, "/") + "/" + url.PathEscape(username) + "/"
	if usernameURL != baseURL {
		if fallbackItems, fallbackErr := s.propfindCollections(usernameURL, username, password); fallbackErr == nil && len(fallbackItems) > 0 {
			return fallbackItems, nil
		}
	}
	return items, nil
}

func (s *CalDAVService) propfindCollections(baseURL, username, password string) ([]domain.DAVCollection, error) {
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	body := bytes.NewBufferString(`<?xml version="1.0" encoding="utf-8"?><d:propfind xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav" xmlns:card="urn:ietf:params:xml:ns:carddav" xmlns:cs="http://calendarserver.org/ns/"><d:prop><d:displayname/><d:resourcetype/><cal:supported-calendar-component-set/><cs:getctag/><d:sync-token/></d:prop></d:propfind>`)
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", baseURL, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("User-Agent", "CalendarAdvanced DAV")
	resp, err := client.Do(req)
	if err != nil {
		return nil, NewError("dav_discovery_failed", "DAV-Server ist nicht erreichbar oder antwortet nicht.", nil)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, NewError("dav_auth_failed", "Anmeldung fehlgeschlagen. Bitte Username/E-Mail und Passwort prüfen.", nil)
	}
	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, NewError("dav_discovery_failed", fmt.Sprintf("DAV-Server antwortet mit Status %d.", resp.StatusCode), nil)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	var multistatus davPropfindResponse
	if err := xml.Unmarshal(data, &multistatus); err != nil {
		return nil, NewError("dav_discovery_failed", "DAV-Antwort konnte nicht gelesen werden.", nil)
	}
	return davCollectionsFromResponses(baseURL, multistatus.Responses), nil
}

func davCollectionsFromResponses(baseURL string, responses []davResponse) []domain.DAVCollection {
	items := make([]domain.DAVCollection, 0, len(responses))
	seen := map[string]bool{}
	for _, response := range responses {
		prop := firstDAVProp(response)
		resource := strings.ToLower(prop.Resource.Inner)
		components := strings.ToUpper(prop.Components.Inner)
		isCalendar := strings.Contains(resource, "calendar")
		isAddressbook := strings.Contains(resource, "addressbook")
		if !isCalendar && !isAddressbook {
			continue
		}
		resolvedURL := resolveDAVCollectionHref(baseURL, response.Href)
		if resolvedURL == "" || seen[resolvedURL] {
			continue
		}
		seen[resolvedURL] = true
		displayName := strings.TrimSpace(prop.DisplayName)
		if displayName == "" {
			displayName = fallbackDAVName(resolvedURL)
		}
		supportsEvents := isCalendar
		supportsTasks := isCalendar
		if strings.TrimSpace(components) != "" {
			supportsEvents = strings.Contains(components, "VEVENT")
			supportsTasks = strings.Contains(components, "VTODO")
		}
		kind := "calendar"
		if isAddressbook {
			kind = "addressbook"
			supportsEvents = false
			supportsTasks = false
		}
		items = append(items, domain.DAVCollection{
			Kind:           kind,
			DisplayName:    displayName,
			URL:            resolvedURL,
			Selected:       true,
			SupportsEvents: supportsEvents,
			SupportsTasks:  supportsTasks,
			CTag:           strings.TrimSpace(prop.CTag),
			SyncToken:      strings.TrimSpace(prop.SyncToken),
		})
	}
	return items
}

func firstDAVProp(response davResponse) davProp {
	for _, propstat := range response.Propstat {
		if propstat.Prop.Resource.Inner != "" || propstat.Prop.DisplayName != "" || propstat.Prop.CalendarData != "" || propstat.Prop.AddressData != "" || propstat.Prop.ETag != "" {
			return propstat.Prop
		}
	}
	return davProp{}
}

func resolveDAVCollectionHref(baseURL, href string) string {
	resolved := resolveDAVHref(baseURL, href)
	if resolved == "" {
		return ""
	}
	return strings.TrimRight(resolved, "/") + "/"
}

func resolveDAVResourceHref(baseURL, href string) string {
	return normalizeDAVResourceURL(resolveDAVHref(baseURL, href))
}

func resolveDAVHref(baseURL, href string) string {
	if strings.TrimSpace(href) == "" {
		return ""
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	parsed, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return ""
	}
	return base.ResolveReference(parsed).String()
}

func normalizeDAVResourceURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func fallbackDAVName(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "DAV Collection"
	}
	path := strings.Trim(strings.TrimRight(parsed.Path, "/"), "/")
	if path == "" {
		return "DAV Collection"
	}
	parts := strings.Split(path, "/")
	name, err := url.PathUnescape(parts[len(parts)-1])
	if err != nil || strings.TrimSpace(name) == "" {
		return parts[len(parts)-1]
	}
	return name
}

func normalizeCalDAVURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty caldav url")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid caldav url")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", fmt.Errorf("invalid caldav url scheme")
	}
	return strings.TrimRight(parsed.String(), "/") + "/", nil
}

func defaultCalDAVConnection(userID int64) domain.CalDAVConnection {
	return domain.CalDAVConnection{
		UserID:               userID,
		DisplayName:          "CalDAV",
		SyncDirection:        "pull",
		SyncEvents:           true,
		SyncTasks:            false,
		SyncContacts:         true,
		SyncIntervalMinutes:  60,
		SyncWindowPastDays:   30,
		SyncWindowFutureDays: 365,
	}
}
