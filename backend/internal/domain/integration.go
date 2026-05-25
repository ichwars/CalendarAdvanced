package domain

import "time"

type CalDAVAccount struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"userId"`
	Name       string    `json:"name"`
	TokenHint  string    `json:"tokenHint"`
	LastUsedAt time.Time `json:"lastUsedAt,omitempty"`
	RevokedAt  time.Time `json:"revokedAt,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

type CalDAVConnection struct {
	ID                   int64     `json:"id"`
	UserID               int64     `json:"userId"`
	DisplayName          string    `json:"displayName"`
	BaseURL              string    `json:"baseUrl"`
	Username             string    `json:"username"`
	PasswordEncrypted    string    `json:"-"`
	PasswordConfigured   bool      `json:"passwordConfigured"`
	SyncEnabled          bool      `json:"syncEnabled"`
	SyncDirection        string    `json:"syncDirection"`
	SyncEvents           bool      `json:"syncEvents"`
	SyncTasks            bool      `json:"syncTasks"`
	SyncContacts         bool      `json:"syncContacts"`
	SyncIntervalMinutes  int       `json:"syncIntervalMinutes"`
	SyncWindowPastDays   int       `json:"syncWindowPastDays"`
	SyncWindowFutureDays int       `json:"syncWindowFutureDays"`
	LastTestAt           time.Time `json:"lastTestAt,omitempty"`
	LastTestStatus       string    `json:"lastTestStatus,omitempty"`
	LastTestMessage      string    `json:"lastTestMessage,omitempty"`
	LastSyncAt           time.Time `json:"lastSyncAt,omitempty"`
	LastSyncStatus       string    `json:"lastSyncStatus,omitempty"`
	LastSyncMessage      string    `json:"lastSyncMessage,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type DAVCollection struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"userId"`
	Kind           string    `json:"kind"`
	DisplayName    string    `json:"displayName"`
	URL            string    `json:"url"`
	Selected       bool      `json:"selected"`
	SupportsEvents bool      `json:"supportsEvents"`
	SupportsTasks  bool      `json:"supportsTasks"`
	CTag           string    `json:"ctag,omitempty"`
	SyncToken      string    `json:"syncToken,omitempty"`
	LastSeenAt     time.Time `json:"lastSeenAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type DAVSyncItem struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"userId"`
	CollectionURL string    `json:"collectionUrl"`
	ResourceURL   string    `json:"resourceUrl"`
	Kind          string    `json:"kind"`
	LocalID       int64     `json:"localId"`
	UID           string    `json:"uid,omitempty"`
	ETag          string    `json:"etag,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type DAVSyncRun struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Mode      string    `json:"mode"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Events    int       `json:"events"`
	Tasks     int       `json:"tasks"`
	Contacts  int       `json:"contacts"`
	Skipped   int       `json:"skipped"`
	Warnings  string    `json:"warnings"`
	CreatedAt time.Time `json:"createdAt"`
}

type ExcelExport struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"userId"`
	Kind       string    `json:"kind"`
	Format     string    `json:"format"`
	RangeStart time.Time `json:"rangeStart,omitempty"`
	RangeEnd   time.Time `json:"rangeEnd,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}
