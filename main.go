package main

import "github.com/yourusername/go-api-starter/setup"

// @title           Go API Starter
// @version         1.0
// @description     A plug-and-play Go REST API using Fiber, PostgreSQL (SQLC), Redis async jobs, and testcontainers.

// @host      localhost:3000
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the token with the `Bearer ` prefix, e.g. "Bearer abcde12345"

func main() {
	app, dbConn := setup.SetupApp()
	setup.StartApp(app, dbConn)
}
