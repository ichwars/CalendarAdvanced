package domain

import "time"

type AuditAction string

const (
	AuditSetupAdminCreated      AuditAction = "setup.admin_created"
	AuditLoginSucceeded         AuditAction = "auth.login_succeeded"
	AuditLoginFailed            AuditAction = "auth.login_failed"
	AuditLogout                 AuditAction = "auth.logout"
	AuditPasswordChanged        AuditAction = "auth.password_changed"
	AuditPasswordResetRequested AuditAction = "auth.password_reset_requested"
	AuditPasswordResetCompleted AuditAction = "auth.password_reset_completed"
	AuditTwoFactorEnabled       AuditAction = "auth.2fa_enabled"
	AuditTwoFactorDisabled      AuditAction = "auth.2fa_disabled"
	AuditTwoFactorRecoveryUsed  AuditAction = "auth.2fa_recovery_used"
	AuditUserCreated            AuditAction = "users.created"
	AuditUserUpdated            AuditAction = "users.updated"
	AuditCalendarChanged        AuditAction = "calendar.changed"
	AuditEventChanged           AuditAction = "event.changed"
	AuditTaskChanged            AuditAction = "task.changed"
	AuditContactChanged         AuditAction = "contact.changed"
	AuditIntegrationChanged     AuditAction = "integration.changed"
	AuditExportCreated          AuditAction = "export.created"
	AuditBackupCreated          AuditAction = "backup.created"
	AuditBackupRestored         AuditAction = "backup.restored"
)

type AuditEntry struct {
	ID         int64       `json:"id"`
	ActorID    int64       `json:"actorId,omitempty"`
	Action     AuditAction `json:"action"`
	EntityType string      `json:"entityType,omitempty"`
	EntityID   string      `json:"entityId,omitempty"`
	IP         string      `json:"ip,omitempty"`
	UserAgent  string      `json:"userAgent,omitempty"`
	Metadata   string      `json:"metadata,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
}
