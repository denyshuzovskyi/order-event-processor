package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"log/slog"
	"net/http"
	"order-event-processor/internal/broadcaster"
	"order-event-processor/internal/config"
	"order-event-processor/internal/handler"
	"order-event-processor/internal/storage/postgresql"
	"os"
)

func main() {
	cfg := config.ReadConfig("./config/local.yaml")
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	validate := validator.New()

	dbpool, err := pgxpool.New(context.Background(), cfg.Datasource.Url)
	if err != nil {
		log.Error("unable to create connection pool", "error", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	driver, err := postgres.WithInstance(stdlib.OpenDBFromPool(dbpool), &postgres.Config{})
	if err != nil {
		log.Error("unable to acquire database driver", "error", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres", driver)
	if err != nil {
		log.Error("unable to set up migrations", "error", err)
		os.Exit(1)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Error("unable to apply migrations", "error", err)
		os.Exit(1)
	}
	log.Info("migration completed successfully")

	storage := postgresql.New(dbpool)

	router := http.NewServeMux()
	producer := broadcaster.NewFromDbEventProducer(storage)
	orderEventBroadcaster := broadcaster.NewOrderEventBroadcaster(producer)
	producer.OrderEventBroadcaster = orderEventBroadcaster
	paymentSystemEventHandler := handler.NewPaymentSystemEventHandler(log, validate, storage, orderEventBroadcaster)
	orderEventStreamHandler := handler.NewOrderEventStreamHandler(log, orderEventBroadcaster)
	ordersHandler := handler.NewOrdersHandler(log, storage)

	router.HandleFunc("POST /webhooks/payments/orders", paymentSystemEventHandler.Handle)
	router.HandleFunc("GET /orders/{order_id}/events", orderEventStreamHandler.StreamOrderEvents)
	router.HandleFunc("GET /orders", ordersHandler.GetAllOrders)

	server := http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.HTTPServer.Host, cfg.HTTPServer.Port),
		Handler: router,
	}

	log.Info("starting server", "host", cfg.HTTPServer.Host, "port", cfg.HTTPServer.Port)

	err = server.ListenAndServe()
	if err != nil {
		log.Error("failed to start server", "error", err)
		return
	}
}
