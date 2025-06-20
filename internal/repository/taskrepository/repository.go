package taskrepository

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/nzb3/workmate_test/internal/models/taskmodel"
)

type InMemoryTaskRepository struct {
	store sync.Map // [uuid.UUID]*taskmodel.Task
}

func NewInMemoryTaskRepository() *InMemoryTaskRepository {
	return &InMemoryTaskRepository{}
}

func (r *InMemoryTaskRepository) Create(task *taskmodel.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	if _, exists := r.store.Load(task.ID); exists {
		return fmt.Errorf("task with ID %s already exists", task.ID.String())
	}

	task.CreatedAt = time.Now()

	taskCopy := r.copyTask(task)
	r.store.Store(task.ID, taskCopy)

	return nil
}

func (r *InMemoryTaskRepository) GetByID(id uuid.UUID) (*taskmodel.Task, error) {
	value, exists := r.store.Load(id)
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", id.String())
	}

	task, ok := value.(*taskmodel.Task)
	if !ok {
		return nil, fmt.Errorf("invalid task data for ID %s", id.String())
	}

	return r.copyTask(task), nil
}

func (r *InMemoryTaskRepository) Update(task *taskmodel.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	if _, exists := r.store.Load(task.ID); !exists {
		return fmt.Errorf("task with ID %s not found", task.ID.String())
	}

	taskCopy := r.copyTask(task)
	r.store.Store(task.ID, taskCopy)

	return nil
}

func (r *InMemoryTaskRepository) Delete(id uuid.UUID) error {
	if _, exists := r.store.Load(id); !exists {
		return fmt.Errorf("task with ID %s not found", id.String())
	}

	r.store.Delete(id)
	return nil
}

func (r *InMemoryTaskRepository) GetAll() ([]*taskmodel.Task, error) {
	var tasks []*taskmodel.Task

	r.store.Range(func(key, value interface{}) bool {
		if task, ok := value.(*taskmodel.Task); ok {
			tasks = append(tasks, r.copyTask(task))
		}
		return true
	})

	return tasks, nil
}

func (r *InMemoryTaskRepository) copyTask(original *taskmodel.Task) *taskmodel.Task {
	if original == nil {
		return nil
	}

	return &taskmodel.Task{
		ID:             original.ID,
		Name:           original.Name,
		Status:         original.Status,
		CreatedAt:      original.CreatedAt,
		ProcessingTime: original.ProcessingTime,
	}
}

func (r *InMemoryTaskRepository) GetTaskCount() int {
	count := 0
	r.store.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func (r *InMemoryTaskRepository) GetTasksByStatus(status taskmodel.TaskStatus) ([]*taskmodel.Task, error) {
	var tasks []*taskmodel.Task

	r.store.Range(func(key, value interface{}) bool {
		if task, ok := value.(*taskmodel.Task); ok && task.Status == status {
			tasks = append(tasks, r.copyTask(task))
		}
		return true
	})

	return tasks, nil
}

func (r *InMemoryTaskRepository) Clear() {
	r.store.Range(func(key, value interface{}) bool {
		r.store.Delete(key)
		return true
	})
}
