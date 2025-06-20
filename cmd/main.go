// @title Workmate API
// @version 1.0
// @description API for task management
// @host localhost:8080
// @BasePath /api/v1
package main

import (
	_ "github.com/nzb3/workmate_test/docs"
	"github.com/nzb3/workmate_test/internal/app"
)

func main() {
	app.Start()
}
