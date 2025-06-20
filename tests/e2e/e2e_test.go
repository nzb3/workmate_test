package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"Workmate/internal/app"
	"Workmate/internal/models/taskmodel"
)

type E2ETestSuite struct {
	suite.Suite
	server  *httptest.Server
	client  *http.Client
	baseURL string
	ctx     context.Context
	cancel  context.CancelFunc
}

func (s *E2ETestSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	container := app.NewDIContainer()
	engine := container.GinEngine(s.ctx)
	s.server = httptest.NewServer(engine)
	s.baseURL = s.server.URL + "/api/v1"
	s.client = &http.Client{
		Timeout: 30 * time.Second,
	}
}

func (s *E2ETestSuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}
	if s.cancel != nil {
		s.cancel()
	}
}

type CreateTaskRequest struct {
	Name string `json:"name"`
}

type TaskResponse struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Status         taskmodel.TaskStatus `json:"status"`
	CreatedAt      string               `json:"created_at"`
	ProcessingTime int64                `json:"processing_time"`
}

type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (s *E2ETestSuite) createTaskRequest(name string) (TaskResponse, *http.Response, error) {
	request := CreateTaskRequest{Name: name}
	body, err := json.Marshal(request)
	if err != nil {
		return TaskResponse{}, nil, err
	}

	resp, err := s.client.Post(
		s.baseURL+"/task/create",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return TaskResponse{}, resp, err
	}

	var taskResp TaskResponse
	if resp.StatusCode == http.StatusAccepted {
		err = json.NewDecoder(resp.Body).Decode(&taskResp)
	}

	return taskResp, resp, err
}

func (s *E2ETestSuite) getTaskRequest(taskID string) (TaskResponse, *http.Response, error) {
	resp, err := s.client.Get(s.baseURL + "/task/" + taskID)
	if err != nil {
		return TaskResponse{}, resp, err
	}

	var taskResp TaskResponse
	if resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&taskResp)
	}

	return taskResp, resp, err
}

func (s *E2ETestSuite) listTasksRequest() (TaskListResponse, *http.Response, error) {
	resp, err := s.client.Get(s.baseURL + "/tasks")
	if err != nil {
		return TaskListResponse{}, resp, err
	}

	var listResp TaskListResponse
	if resp.StatusCode == http.StatusOK {
		err = json.NewDecoder(resp.Body).Decode(&listResp)
	}

	return listResp, resp, err
}

func (s *E2ETestSuite) deleteTaskRequest(taskID string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", s.baseURL+"/task/"+taskID, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req)
}

func (s *E2ETestSuite) getErrorResponse(resp *http.Response) (ErrorResponse, error) {
	var errorResp ErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&errorResp)
	return errorResp, err
}

func (s *E2ETestSuite) TestCreateTask() {
	taskResp, resp, err := s.createTaskRequest("Test Task")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusAccepted, resp.StatusCode)

	location := resp.Header.Get("Location")
	assert.NotEmpty(s.T(), location)
	assert.Contains(s.T(), location, "/api/v1/task/")

	assert.NotEmpty(s.T(), taskResp.ID)
	assert.Equal(s.T(), "Test Task", taskResp.Name)
	assert.Equal(s.T(), taskmodel.StatusProcessing, taskResp.Status)
	assert.NotEmpty(s.T(), taskResp.CreatedAt)
	assert.GreaterOrEqual(s.T(), taskResp.ProcessingTime, int64(0))

	_, err = uuid.Parse(taskResp.ID)
	assert.NoError(s.T(), err)
}

func (s *E2ETestSuite) TestGetTask() {
	taskID := s.createTestTask("Get Task Test")

	taskResp, resp, err := s.getTaskRequest(taskID)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.Equal(s.T(), taskID, taskResp.ID)
	assert.Equal(s.T(), "Get Task Test", taskResp.Name)
	assert.Contains(s.T(), []taskmodel.TaskStatus{
		taskmodel.StatusProcessing,
		taskmodel.StatusDone,
		taskmodel.StatusFailed,
	}, taskResp.Status)
}

func (s *E2ETestSuite) TestListTasks() {
	taskNames := []string{"List Task 1", "List Task 2", "List Task 3"}
	createdIDs := make([]string, 0)

	for _, name := range taskNames {
		id := s.createTestTask(name)
		createdIDs = append(createdIDs, id)
	}

	listResp, resp, err := s.listTasksRequest()
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	assert.GreaterOrEqual(s.T(), len(listResp.Tasks), len(taskNames))

	foundTasks := make(map[string]bool)
	for _, task := range listResp.Tasks {
		foundTasks[task.ID] = true
	}

	for _, id := range createdIDs {
		assert.True(s.T(), foundTasks[id], "Task %s not found in list", id)
	}
}

func (s *E2ETestSuite) TestDeleteTask() {
	taskID := s.createTestTask("Delete Task Test")

	resp, err := s.deleteTaskRequest(taskID)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNoContent, resp.StatusCode)

	_, getResp, err := s.getTaskRequest(taskID)
	require.NoError(s.T(), err)
	defer getResp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, getResp.StatusCode)
}

func (s *E2ETestSuite) TestCreateTaskInvalidInput() {
	testCases := []struct {
		name           string
		request        interface{}
		expectedStatus int
	}{
		{
			name:           "Empty name",
			request:        CreateTaskRequest{Name: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Too long name",
			request:        CreateTaskRequest{Name: strings.Repeat("a", 101)},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			request:        `{"name": }`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing name field",
			request:        map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tc.request.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tc.request)
				require.NoError(t, err)
			}

			resp, err := s.client.Post(
				s.baseURL+"/task/create",
				"application/json",
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			if resp.StatusCode == http.StatusBadRequest {
				errorResp, err := s.getErrorResponse(resp)
				require.NoError(t, err)
				assert.NotEmpty(t, errorResp.Error)
			}
		})
	}
}

func (s *E2ETestSuite) TestGetTaskNotFound() {
	randomID := uuid.New().String()

	_, resp, err := s.getTaskRequest(randomID)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)

	errorResp, err := s.getErrorResponse(resp)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), errorResp.Error)
}

func (s *E2ETestSuite) TestGetTaskInvalidID() {
	invalidIDs := []string{
		"invalid-uuid",
		"123",
		"not-a-uuid-at-all",
	}

	for _, invalidID := range invalidIDs {
		s.T().Run("Invalid ID: "+invalidID, func(t *testing.T) {
			_, resp, err := s.getTaskRequest(invalidID)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			errorResp, err := s.getErrorResponse(resp)
			require.NoError(t, err)
			assert.NotEmpty(t, errorResp.Error)
		})
	}
}

func (s *E2ETestSuite) TestDeleteTaskNotFound() {
	randomID := uuid.New().String()

	resp, err := s.deleteTaskRequest(randomID)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *E2ETestSuite) TestTaskLifecycle() {
	taskID := s.createTestTask("Lifecycle Test Task")

	task := s.getTask(taskID)
	assert.Equal(s.T(), taskmodel.StatusProcessing, task.Status)
	assert.GreaterOrEqual(s.T(), task.ProcessingTime, int64(0))

	time.Sleep(2 * time.Second)
	updatedTask := s.getTask(taskID)
	assert.GreaterOrEqual(s.T(), updatedTask.ProcessingTime, int64(1))

	tasks := s.listTasks()
	found := false
	for _, t := range tasks.Tasks {
		if t.ID == taskID {
			found = true
			break
		}
	}
	assert.True(s.T(), found)

	s.deleteTask(taskID)

	_, resp, err := s.getTaskRequest(taskID)
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *E2ETestSuite) TestConcurrentTaskCreation() {
	const numTasks = 10
	results := make(chan string, numTasks)
	errors := make(chan error, numTasks)

	for i := 0; i < numTasks; i++ {
		go func(index int) {
			taskName := fmt.Sprintf("Concurrent Task %d", index)
			taskResp, resp, err := s.createTaskRequest(taskName)
			if resp != nil {
				defer resp.Body.Close()
			}

			if err != nil {
				errors <- err
				return
			}

			if resp.StatusCode != http.StatusAccepted {
				errors <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
				return
			}

			results <- taskResp.ID
		}(i)
	}

	createdIDs := make([]string, 0)
	for i := 0; i < numTasks; i++ {
		select {
		case id := <-results:
			createdIDs = append(createdIDs, id)
		case err := <-errors:
			s.T().Errorf("Error creating task: %v", err)
		case <-time.After(10 * time.Second):
			s.T().Fatal("Timeout waiting for task creation")
		}
	}

	assert.Equal(s.T(), numTasks, len(createdIDs))

	idSet := make(map[string]bool)
	for _, id := range createdIDs {
		assert.False(s.T(), idSet[id], "Duplicate task ID: %s", id)
		idSet[id] = true
	}
}

func (s *E2ETestSuite) TestTaskProcessingTime() {
	taskID := s.createTestTask("Processing Time Test")

	times := make([]int64, 0)

	for i := 0; i < 3; i++ {
		task := s.getTask(taskID)
		times = append(times, task.ProcessingTime)

		if i < 2 {
			time.Sleep(1 * time.Second)
		}
	}

	for i := 1; i < len(times); i++ {
		assert.GreaterOrEqual(s.T(), times[i], times[i-1],
			"Processing time should be non-decreasing")
	}
}

func (s *E2ETestSuite) createTestTask(name string) string {
	taskResp, resp, err := s.createTaskRequest(name)
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	require.Equal(s.T(), http.StatusAccepted, resp.StatusCode)
	return taskResp.ID
}

func (s *E2ETestSuite) getTask(id string) TaskResponse {
	taskResp, resp, err := s.getTaskRequest(id)
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	return taskResp
}

func (s *E2ETestSuite) listTasks() TaskListResponse {
	listResp, resp, err := s.listTasksRequest()
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	return listResp
}

func (s *E2ETestSuite) deleteTask(id string) {
	resp, err := s.deleteTaskRequest(id)
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	require.Equal(s.T(), http.StatusNoContent, resp.StatusCode)
}

func TestMain(m *testing.M) {
	m.Run()
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
