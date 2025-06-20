// @title Workmate API
// @version 1.0
// @description API for task management
// @host localhost:8080
// @BasePath /api/v1
package main

import (
	_ "Workmate/docs"
	"Workmate/internal/app"
)

func main() {
	app.Start()
}
