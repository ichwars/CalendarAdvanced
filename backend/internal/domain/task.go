package domain

import (
	"errors"
	"strings"
	"time"
)

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityNormal TaskPriority = "normal"
	TaskPriorityHigh   TaskPriority = "high"
)

type Task struct {
	ID                  int64        `json:"id"`
	UserID              int64        `json:"userId"`
	Title               string       `json:"title"`
	Description         string       `json:"description,omitempty"`
	DueAt               time.Time    `json:"dueAt,omitempty"`
	ReminderAt          time.Time    `json:"reminderAt,omitempty"`
	Priority            TaskPriority `json:"priority"`
	Completed           bool         `json:"completed"`
	ShowInCalendar      bool         `json:"showInCalendar"`
	CompletedAt         time.Time    `json:"completedAt,omitempty"`
	ReminderDeliveredAt time.Time    `json:"reminderDeliveredAt,omitempty"`
	CreatedAt           time.Time    `json:"createdAt"`
	UpdatedAt           time.Time    `json:"updatedAt"`
	DAVSynced           bool         `json:"davSynced,omitempty"`
}

func ValidateTask(task Task) error {
	if task.UserID <= 0 {
		return errors.New("task user is required")
	}
	if strings.TrimSpace(task.Title) == "" || len(task.Title) > 200 {
		return errors.New("task title is required and must be at most 200 characters")
	}
	if len(task.Description) > 5000 {
		return errors.New("task description is too long")
	}
	if task.Priority == "" {
		return nil
	}
	switch task.Priority {
	case TaskPriorityLow, TaskPriorityNormal, TaskPriorityHigh:
		return nil
	default:
		return errors.New("task priority is invalid")
	}
}
