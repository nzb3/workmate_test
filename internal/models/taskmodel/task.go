package taskmodel

import (
	"github.com/google/uuid"
	"time"
)

type TaskStatus string

const (
	StatusDone       TaskStatus = "DONE"
	StatusProcessing TaskStatus = "PROCESSING"
	StatusFailed     TaskStatus = "FAILED"
)

type Task struct {
	ID             uuid.UUID
	Name           string
	Status         TaskStatus
	CreatedAt      time.Time
	ProcessingTime time.Duration
}

func NewTask(opts ...Option) *Task {
	task := new(Task)
	task.ID = uuid.New()

	for _, opt := range opts {
		opt(task)
	}

	return task
}

func (t *Task) IsDone() bool {
	return t.Status == StatusDone
}

func (t *Task) IsFailed() bool {
	return t.Status == StatusFailed
}

func (t *Task) IsProcessing() bool {
	return t.Status == StatusProcessing
}

func (t *Task) SetStatus(status TaskStatus) {
	t.Status = status
}
