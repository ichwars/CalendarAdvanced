package application

import (
	"fmt"
	"time"

	"calendaradvanced/internal/domain"
	"calendaradvanced/internal/infrastructure/sqlite"
)

type TaskService struct {
	Store *sqlite.Store
	Audit *AuditService
}

type TaskInput struct {
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	DueAt          time.Time           `json:"dueAt"`
	ReminderAt     time.Time           `json:"reminderAt"`
	Priority       domain.TaskPriority `json:"priority"`
	Completed      bool                `json:"completed"`
	ShowInCalendar bool                `json:"showInCalendar"`
}

type TaskListInput struct {
	Query     string
	Completed *bool
	Limit     int
	Offset    int
}

func (s *TaskService) List(user domain.User, input TaskListInput) ([]domain.Task, error) {
	tasks, err := s.Store.ListTasks(sqlite.TaskFilter{UserID: user.ID, Query: input.Query, Completed: input.Completed, Limit: input.Limit, Offset: input.Offset})
	if err != nil {
		return nil, err
	}
	synced, err := s.Store.ListDAVSyncedLocalIDs(user.ID, "task")
	if err != nil {
		return tasks, nil
	}
	for i := range tasks {
		tasks[i].DAVSynced = synced[tasks[i].ID]
	}
	return tasks, nil
}

func (s *TaskService) Create(input TaskInput, user domain.User, ip, userAgent string) (domain.Task, error) {
	task := taskFromInput(input, user.ID)
	if err := domain.ValidateTask(task); err != nil {
		return domain.Task{}, NewError("invalid_task", err.Error(), nil)
	}
	created, err := s.Store.CreateTask(task)
	if err != nil {
		return domain.Task{}, err
	}
	s.Audit.Record(user.ID, domain.AuditTaskChanged, "task", fmt.Sprint(created.ID), ip, userAgent, map[string]any{"operation": "create"})
	return created, nil
}

func (s *TaskService) Update(id int64, input TaskInput, user domain.User, ip, userAgent string) (domain.Task, error) {
	task := taskFromInput(input, user.ID)
	task.ID = id
	if err := domain.ValidateTask(task); err != nil {
		return domain.Task{}, NewError("invalid_task", err.Error(), nil)
	}
	updated, err := s.Store.UpdateTask(task, user.ID)
	if err != nil {
		return domain.Task{}, err
	}
	s.Audit.Record(user.ID, domain.AuditTaskChanged, "task", fmt.Sprint(updated.ID), ip, userAgent, map[string]any{"operation": "update"})
	return updated, nil
}

func (s *TaskService) Delete(id int64, user domain.User, ip, userAgent string) error {
	if err := s.Store.DeleteTask(id, user.ID); err != nil {
		return err
	}
	s.Audit.Record(user.ID, domain.AuditTaskChanged, "task", fmt.Sprint(id), ip, userAgent, map[string]any{"operation": "delete"})
	return nil
}

func (s *TaskService) MarkReminderDelivered(id int64, user domain.User) error {
	if id <= 0 {
		return ErrValidation
	}
	return s.Store.MarkTaskReminderDelivered(id, user.ID, time.Now().UTC())
}

func taskFromInput(input TaskInput, userID int64) domain.Task {
	return domain.Task{UserID: userID, Title: input.Title, Description: input.Description, DueAt: input.DueAt, ReminderAt: input.ReminderAt, Priority: input.Priority, Completed: input.Completed, ShowInCalendar: input.ShowInCalendar}
}
