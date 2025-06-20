package taskcontroller

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nzb3/workmate_test/internal/models/taskmodel"
)

type TaskService interface {
	CreateTask(ctx context.Context, name string) (*taskmodel.Task, error)
	GetTask(ctx context.Context, taskID uuid.UUID) (*taskmodel.Task, error)
	DeleteTask(ctx context.Context, taskID uuid.UUID) error
	ListTasks(ctx context.Context) ([]*taskmodel.Task, error)
}

// CreateTaskRequest represents a request to create a new task.
// @Description Request payload for creating a task.
type CreateTaskRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

// TaskResponse represents a response with task information.
// @Description Task information including status and processing time.
type TaskResponse struct {
	ID             uuid.UUID            `json:"id"`
	Name           string               `json:"name"`
	Status         taskmodel.TaskStatus `json:"status"`
	CreatedAt      time.Time            `json:"created_at"`
	ProcessingTime time.Duration        `json:"processing_time" swaggertype:"integer"`
}

// TaskListResponse represents a response with a list of tasks.
// @Description List of tasks.
type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
}

// ErrorResponse represents an error response.
// @Description Error response with error code and message.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type Controller struct {
	taskService TaskService
}

func NewController(service TaskService) *Controller {
	return &Controller{
		taskService: service,
	}
}

func (c *Controller) RegisterRoutes(router *gin.RouterGroup) {
	tasks := router.Group("/tasks")
	{
		tasks.GET("", c.ListTasks)
	}
	task := router.Group("/task")
	{
		task.POST("/create", c.CreateTask)
		task.GET("/:id", c.GetTask)
		task.DELETE("/:id", c.DeleteTask)
	}
}

// CreateTask godoc
// @Summary      Create a new task
// @Description  Creates a new task with the specified name
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        request body CreateTaskRequest true "Task info"
// @Success      202 {object} TaskResponse "Task accepted for processing"
// @Failure      400 {object} ErrorResponse "Invalid input"
// @Failure      500 {object} ErrorResponse "Internal error"
// @Header       202 {string} Location "Location of the created task"
// @Router       /task/create [post]
func (c *Controller) CreateTask(ctx *gin.Context) {
	var req CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	task, err := c.taskService.CreateTask(ctx.Request.Context(), req.Name)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create task",
		})
		return
	}

	response := c.mapTaskToResponse(task)
	ctx.Header("Location", "/api/v1/task/"+task.ID.String())
	ctx.JSON(http.StatusAccepted, response)
}

// GetTask godoc
// @Summary      Get task info
// @Description  Returns information about a task by its ID
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id path string true "Task ID (UUID)"
// @Success      200 {object} TaskResponse "Task found"
// @Failure      400 {object} ErrorResponse "Invalid ID format"
// @Failure      404 {object} ErrorResponse "Task not found"
// @Router       /task/{id} [get]
func (c *Controller) GetTask(ctx *gin.Context) {
	taskIDStr := ctx.Param("id")
	if taskIDStr == "" {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Missing task id",
		})
		return
	}
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid task ID format",
		})
		return
	}

	task, err := c.taskService.GetTask(ctx.Request.Context(), taskID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "task_not_found",
			Message: "Task not found",
		})
		return
	}

	response := c.mapTaskToResponse(task)
	ctx.JSON(http.StatusOK, response)
}

// DeleteTask godoc
// @Summary      Delete a task
// @Description  Deletes a task by its ID
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id path string true "Task ID (UUID)"
// @Success      204 "Task deleted"
// @Failure      400 {object} ErrorResponse "Invalid ID format"
// @Failure      404 {object} ErrorResponse "Task not found"
// @Router       /task/{id} [delete]
func (c *Controller) DeleteTask(ctx *gin.Context) {
	taskIDStr := ctx.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid task ID format",
		})
		return
	}

	err = c.taskService.DeleteTask(ctx.Request.Context(), taskID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "task_not_found",
			Message: "Task not found",
		})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ListTasks godoc
// @Summary      List all tasks
// @Description  Returns a list of all tasks
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Success      200 {object} TaskListResponse "List of tasks"
// @Failure      500 {object} ErrorResponse "Internal error"
// @Router       /tasks [get]
func (c *Controller) ListTasks(ctx *gin.Context) {
	tasks, err := c.taskService.ListTasks(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve tasks",
		})
		return
	}

	response := TaskListResponse{
		Tasks: make([]TaskResponse, len(tasks)),
	}

	for i, task := range tasks {
		response.Tasks[i] = c.mapTaskToResponse(task)
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *Controller) mapTaskToResponse(task *taskmodel.Task) TaskResponse {
	return TaskResponse{
		ID:             task.ID,
		Name:           task.Name,
		Status:         task.Status,
		CreatedAt:      task.CreatedAt,
		ProcessingTime: task.ProcessingTime,
	}
}
