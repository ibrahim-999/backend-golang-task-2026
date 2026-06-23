package bootstrap

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/command"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/eventhandler"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/query"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/auth"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/messaging"
	notificationinfra "github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/notification"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/observability"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/payment"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/persistence/gormrepo"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/handler"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/router"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/interfaces/http/ws"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/platform/config"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/concurrency"
)

type App struct {
	Router *gin.Engine
	stop   func()
}

func New(cfg *config.Config, log zerolog.Logger, metrics *observability.Metrics) (*App, error) {
	db, err := gormrepo.Open(cfg.DB, log)
	if err != nil {
		return nil, err
	}
	if err := gormrepo.AutoMigrate(db); err != nil {
		log.Error().Err(err).Msg("database schema migration failed")
		return nil, err
	}
	log.Info().Msg("database schema migrated")

	reads := gormrepo.NewRepositories(db)
	uow := gormrepo.NewUnitOfWork(db)

	gateway := payment.NewMockGateway(cfg.Payment.FailureRate, cfg.Payment.MinLatency, cfg.Payment.MaxLatency)

	inproc := messaging.NewInProcessBus(cfg.Worker.EventWorkers, cfg.Worker.EventQueueSize, log)
	inproc.Start()

	var bus ports.EventBus = inproc
	var rabbitPublisher *messaging.RabbitPublisher
	var rabbitConsumer *messaging.RabbitConsumer
	if cfg.RabbitMQ.Enabled {
		publisher, pubErr := messaging.NewRabbitPublisher(cfg.RabbitMQ.URL(), cfg.RabbitMQ.Exchange, log)
		if pubErr != nil {
			log.Warn().Err(pubErr).Msg("rabbitmq publisher unavailable; continuing with in-process bus")
		} else {
			rabbitPublisher = publisher
			bus = messaging.NewCompositeBus(inproc, publisher)

			consumer, conErr := messaging.NewRabbitConsumer(cfg.RabbitMQ.URL(), cfg.RabbitMQ.Exchange, log)
			if conErr != nil {
				log.Warn().Err(conErr).Msg("rabbitmq consumer unavailable; continuing without it")
			} else if startErr := consumer.Start(context.Background()); startErr != nil {
				log.Warn().Err(startErr).Msg("rabbitmq consumer failed to start; continuing without it")
				consumer.Stop()
			} else {
				rabbitConsumer = consumer
				log.Info().Msg("rabbitmq publisher and consumer started")
			}
		}
	}

	hub := ws.New(log)
	go hub.Run()

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

	inproc.Subscribe(eventhandler.NotificationOnOrderPaid(reads.Orders(), dispatcher))
	inproc.Subscribe(eventhandler.NotificationOnOrderFailed(reads.Orders(), dispatcher))
	inproc.Subscribe(eventhandler.AuditHandler(reads.Audit()))
	inproc.Subscribe(eventhandler.NewWSBroadcast(hub))

	registerUser := command.NewRegisterUser(uow, hasher)
	loginUser := command.NewLoginUser(uow, reads, hasher, tokens)
	updateUser := command.NewUpdateUser(uow)
	getUser := query.NewGetUser(reads)

	createProduct := command.NewCreateProduct(uow)
	updateProduct := command.NewUpdateProduct(uow)
	listProducts := query.NewListProducts(reads)
	getProduct := query.NewGetProduct(reads)
	getInventory := query.NewGetInventory(reads)

	placeOrder := command.NewPlaceOrder(uow, reads, gateway, bus, pool, cfg.Payment.MaxAttempts, cfg.Payment.RetryBackoff)
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
	docsHandler := handler.NewDocs("./docs/openapi.yaml")

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
		Docs:          docsHandler,
		WS:            ws.ServeWS(hub),
	})

	return &App{
		Router: engine,
		stop: func() {
			pool.Stop()
			inproc.Stop()
			dispatcher.Stop()
			hub.Stop()
			if rabbitConsumer != nil {
				rabbitConsumer.Stop()
			}
			if rabbitPublisher != nil {
				if e := rabbitPublisher.Close(); e != nil {
					log.Warn().Err(e).Msg("rabbitmq publisher close failed")
				}
			}
			if sqlDB, e := db.DB(); e == nil {
				_ = sqlDB.Close()
			}
		},
	}, nil
}

func (a *App) Stop() {
	a.stop()
}
