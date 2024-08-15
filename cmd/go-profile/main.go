package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/oswaldo-oliveira/go-profile/internal/api"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Failed to execute code.", "error", err)
		os.Exit(1)
	}
	slog.Info("All system offline.")
}

func run() error {
	db := api.NewApplication(make(map[api.ID]api.User))
	handler := api.NewHandler(db)

	srv := http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  time.Minute,
		Addr:         ":8080",
		Handler:      handler,
	}
	if err := srv.ListenAndServe(); err != nil {
		return err
	}

	return nil
}
