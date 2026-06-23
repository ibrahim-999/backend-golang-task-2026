package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/command"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/eventhandler"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/query"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/auth"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/messaging"
	notificationinfra "github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/notification"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/payment"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/persistence/gormrepo"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/handler"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/router"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/server"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/concurrency"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := observability.NewLogger(cfg.App.Name, cfg.App.LogLevel, cfg.App.LogPretty)
	metrics := observability.NewMetrics("orders")

	db, err := gormrepo.Open(cfg.DB, log)
	if err != nil {
		return err
	}
	if err := gormrepo.AutoMigrate(db); err != nil {
		log.Error().Err(err).Msg("database schema migration failed")
		return err
	}
	log.Info().Msg("database schema migrated")
	defer func() {
		if sqlDB, derr := db.DB(); derr == nil {
			_ = sqlDB.Close()
		}
	}()

	reads := gormrepo.NewRepositories(db)
	uow := gormrepo.NewUnitOfWork(db)

	gateway := payment.NewMockGateway(cfg.Payment.FailureRate, cfg.Payment.MinLatency, cfg.Payment.MaxLatency)

	bus := messaging.NewInProcessBus(cfg.Worker.EventWorkers, cfg.Worker.EventQueueSize, log)
	bus.Start()

	dispatcher := notificationinfra.NewDispatcher(
		reads.Notifications(),
		notificationinfra.NewLogSender(log),
		cfg.Worker.NotificationWorkers,
		cfg.Worker.NotificationQueueSize,
		log,
	)
	dispatcher.Start()

	pool := concurrency.NewPool(cfg.Worker.OrderWorkers, cfg.Worker.OrderQueueSize)
	pool.Start()

	hasher := auth.NewBcryptHasher()
	tokens := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.AccessTTL)

	bus.Subscribe(eventhandler.NotificationOnOrderPaid(reads.Orders(), dispatcher))
	bus.Subscribe(eventhandler.NotificationOnOrderFailed(reads.Orders(), dispatcher))
	bus.Subscribe(eventhandler.AuditHandler(reads.Audit()))

	registerUser := command.NewRegisterUser(uow, hasher)
	loginUser := command.NewLoginUser(uow, reads, hasher, tokens)
	updateUser := command.NewUpdateUser(uow)
	getUser := query.NewGetUser(reads)

	createProduct := command.NewCreateProduct(uow)
	updateProduct := command.NewUpdateProduct(uow)
	listProducts := query.NewListProducts(reads)
	getProduct := query.NewGetProduct(reads)
	getInventory := query.NewGetInventory(reads)

	placeOrder := command.NewPlaceOrder(uow, reads, gateway, bus, pool)
	cancelOrder := command.NewCancelOrder(uow, reads, bus)
	getOrder := query.NewGetOrder(reads)
	orderStatus := query.NewOrderStatus(reads)
	listUserOrders := query.NewListUserOrders(reads)

	listAllOrders := query.NewListAllOrders(reads)
	updateOrderStatus := command.NewUpdateOrderStatus(uow)
	dailyReport := query.NewDailyReport(reads)
	lowStock := query.NewLowStock(reads)

	authHandler := handler.NewAuth(registerUser, loginUser)
	userHandler := handler.NewUser(registerUser, updateUser, getUser)
	productHandler := handler.NewProduct(createProduct, updateProduct, listProducts, getProduct, getInventory)
	orderHandler := handler.NewOrder(placeOrder, cancelOrder, getOrder, orderStatus, listUserOrders)
	adminHandler := handler.NewAdmin(listAllOrders, updateOrderStatus, dailyReport, lowStock)

	engine := router.New(router.Dependencies{
		Version:       cfg.App.Version,
		Environment:   cfg.App.Environment,
		Logger:        log,
		Metrics:       metrics,
		DB:            db,
		TokenVerifier: tokens,
		RateLimit:     cfg.RateLimit,
		Auth:          authHandler,
		User:          userHandler,
		Product:       productHandler,
		Order:         orderHandler,
		Admin:         adminHandler,
	})

	srv := server.New(cfg.HTTP, engine, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info().
		Str("version", cfg.App.Version).
		Str("environment", cfg.App.Environment).
		Str("addr", cfg.HTTP.Addr()).
		Msg("starting order-processing service")

	runErr := srv.Run(ctx)

	pool.Stop()
	dispatcher.Stop()
	bus.Stop()

	return runErr
}

func healthcheck() int {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
