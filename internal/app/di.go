package app

import (
	"context"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/nzb3/workmate_test/internal/controllers"
	"github.com/nzb3/workmate_test/internal/controllers/taskcontroller"
	"github.com/nzb3/workmate_test/internal/repository/taskrepository"
	"github.com/nzb3/workmate_test/internal/service/taskservice"
)

type DIContainer struct {
	taskController *taskcontroller.Controller
	taskService    *taskservice.Service
	taskRepository *taskrepository.InMemoryTaskRepository
	server         *http.Server
	ginEngine      *gin.Engine
}

func NewDIContainer() *DIContainer {
	return &DIContainer{}
}

func (c *DIContainer) TaskController(ctx context.Context) *taskcontroller.Controller {
	if c.taskController != nil {
		return c.taskController
	}

	controller := taskcontroller.NewController(c.TaskService(ctx))
	c.taskController = controller

	return controller
}

func (c *DIContainer) TaskService(ctx context.Context) *taskservice.Service {
	if c.taskService != nil {
		return c.taskService
	}

	service := taskservice.NewService(c.TaskRepository(ctx))
	c.taskService = service
	return service
}

func (c *DIContainer) TaskRepository(ctx context.Context) *taskrepository.InMemoryTaskRepository {
	if c.taskRepository != nil {
		return c.taskRepository
	}

	repository := taskrepository.NewInMemoryTaskRepository()
	c.taskRepository = repository
	return repository
}

func (c *DIContainer) Server(ctx context.Context) *http.Server {
	if c.server != nil {
		return c.server
	}

	s := &http.Server{
		Addr:    ":8080",
		Handler: c.GinEngine(ctx),
	}

	c.server = s
	return s
}

func (c *DIContainer) GinEngine(ctx context.Context) *gin.Engine {
	if c.ginEngine != nil {
		return c.ginEngine
	}

	engine := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true

	engine.Use(cors.New(corsConfig))

	api := engine.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			c.TaskController(ctx).RegisterRoutes(v1)
			v1.GET("/health", controllers.HealthCheck)
			v1.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}
	}

	c.ginEngine = engine
	return engine
}
