package main

import (
	"errors"
	"github.com/Piszmog/make-a-decision/internal/db"
	"github.com/Piszmog/make-a-decision/internal/log"
	"github.com/Piszmog/make-a-decision/internal/server"
	"github.com/Piszmog/make-a-decision/internal/server/router"
	"os"

	"github.com/golang-migrate/migrate/v4"
)

func main() {
	logger := log.New(
		log.GetLevel(),
		log.GetOutput(),
	)

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "./db.sqlite3"
	}

	database, err := db.New(logger, dbURL)
	if err != nil {
		logger.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if cerr := database.Close(); cerr != nil {
			logger.Error("failed to close the database", "error", cerr)
		}
	}()

	if err = db.Migrate(database); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.Error("failed to migrate database", "error", err)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	svr := server.New(
		logger,
		":"+port,
		server.WithRouter(router.New(logger, database)),
	)

	svr.StartAndWait()
}
