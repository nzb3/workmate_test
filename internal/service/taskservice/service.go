package taskservice

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/nzb3/workmate_test/internal/models/taskmodel"
)

const defaultTimeToProcessTask = 6 * time.Minute

type Repository interface {
	Create(task *taskmodel.Task) error
	GetByID(id uuid.UUID) (*taskmodel.Task, error)
	Update(task *taskmodel.Task) error
	Delete(id uuid.UUID) error
	GetAll() ([]*taskmodel.Task, error)
}

type TaskContext struct {
	ID      uuid.UUID
	Cancel  context.CancelFunc
	Started time.Time
	Done    chan struct{}
	Status  taskmodel.TaskStatus
	mu      sync.RWMutex
}

func (tc *TaskContext) IsFinished() bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	select {
	case <-tc.Done:
		return true
	default:
		return false
	}
}

func (tc *TaskContext) markFinished(status taskmodel.TaskStatus) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.Status = status
	select {
	case <-tc.Done:
	default:
		close(tc.Done)
	}
}

type Service struct {
	repo     Repository
	contexts sync.Map //[uuid.UUID]*TaskContext
	wg       sync.WaitGroup
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateTask(ctx context.Context, name string) (*taskmodel.Task, error) {
	task := taskmodel.NewTask(taskmodel.WithName(name))
	task.SetStatus(taskmodel.StatusProcessing)
	task.CreatedAt = time.Now()

	if err := s.repo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	taskCtx, cancel := context.WithTimeout(context.Background(), defaultTimeToProcessTask)
	taskContext := &TaskContext{
		ID:      task.ID,
		Cancel:  cancel,
		Started: time.Now(),
		Done:    make(chan struct{}),
		Status:  taskmodel.StatusProcessing,
	}

	s.contexts.Store(task.ID, taskContext)
	s.wg.Add(1)

	go s.executeTask(taskCtx, *task, taskContext)

	return task, nil
}

func (s *Service) GetTask(ctx context.Context, taskID uuid.UUID) (*taskmodel.Task, error) {
	task, err := s.repo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	s.updateTaskProcessingTime(task)
	return task, nil
}

func (s *Service) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	_, err := s.repo.GetByID(taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if taskContext, ok := s.loadTaskContext(taskID); ok {
		taskContext.Cancel()
		s.contexts.Delete(taskID)
	}

	if err := s.repo.Delete(taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

func (s *Service) ListTasks(ctx context.Context) ([]*taskmodel.Task, error) {
	tasks, err := s.repo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	for _, task := range tasks {
		s.updateTaskProcessingTime(task)
	}

	return tasks, nil
}

func (s *Service) loadTaskContext(taskID uuid.UUID) (*TaskContext, bool) {
	if value, exists := s.contexts.Load(taskID); exists {
		if tc, ok := value.(*TaskContext); ok {
			return tc, true
		}
	}
	return nil, false
}

func (s *Service) updateTaskProcessingTime(task *taskmodel.Task) {
	if !task.IsProcessing() {
		return
	}

	if taskContext, exists := s.loadTaskContext(task.ID); exists && !taskContext.IsFinished() {
		task.ProcessingTime = time.Since(taskContext.Started)
	}
}

func (s *Service) executeTask(ctx context.Context, task taskmodel.Task, taskContext *TaskContext) {
	defer func() {
		s.wg.Done()
		if !taskContext.IsFinished() {
			taskContext.markFinished(taskmodel.StatusFailed)
		}
		s.contexts.Delete(task.ID)
		log.Printf("Task %s execution finished with status: %s", task.ID, taskContext.Status)
	}()

	log.Printf("Starting task execution: %s (ID: %s)", task.Name, task.ID)

	workDuration := time.Duration(3+rand.Intn(3)) * time.Minute
	log.Printf("Task %s will take %v to complete", task.ID, workDuration)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Task %s was cancelled", task.ID)
			s.finalizeTask(&task, taskmodel.StatusFailed, time.Since(start))
			taskContext.markFinished(taskmodel.StatusFailed)
			return

		case <-ticker.C:
			elapsed := time.Since(start)
			task.ProcessingTime = elapsed

			if elapsed >= workDuration {
				log.Printf("Task %s completed successfully", task.ID)
				s.finalizeTask(&task, taskmodel.StatusDone, elapsed)
				taskContext.markFinished(taskmodel.StatusDone)
				return
			}

			if err := s.repo.Update(&task); err != nil {
				log.Printf("Failed to update task %s during execution: %v", task.ID, err)
				s.finalizeTask(&task, taskmodel.StatusFailed, elapsed)
				taskContext.markFinished(taskmodel.StatusFailed)
				return
			}
		}
	}
}

func (s *Service) finalizeTask(task *taskmodel.Task, status taskmodel.TaskStatus, processingTime time.Duration) {
	task.Status = status
	task.ProcessingTime = processingTime

	if err := s.repo.Update(task); err != nil {
		log.Printf("Failed to finalize task %s: %v", task.ID, err)
	}
}

func (s *Service) Shutdown(ctx context.Context) error {
	log.Println("Shutting down task service...")

	s.contexts.Range(func(key, value interface{}) bool {
		if taskContext, ok := value.(*TaskContext); ok && !taskContext.IsFinished() {
			log.Printf("Cancelling task %s", taskContext.ID)
			taskContext.Cancel()
		}
		return true
	})

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	shutdownTimeout := 30 * time.Second
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	select {
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout reached")
		return errors.New("shutdown timeout")
	case <-done:
		log.Println("All tasks finished, service shutdown complete")
		return nil
	}
}

func (s *Service) WaitForTask(ctx context.Context, taskID uuid.UUID) error {
	taskContext, exists := s.loadTaskContext(taskID)
	if !exists {
		return fmt.Errorf("task %s not found or already finished", taskID)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-taskContext.Done:
		return nil
	}
}

func (s *Service) GetTaskStatus(taskID uuid.UUID) (taskmodel.TaskStatus, bool) {
	if taskContext, exists := s.loadTaskContext(taskID); exists {
		taskContext.mu.RLock()
		status := taskContext.Status
		taskContext.mu.RUnlock()
		return status, true
	}
	return "", false
}
