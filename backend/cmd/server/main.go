package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/client"
	"github.com/xavierli/network-ticket/internal/config"
	"github.com/xavierli/network-ticket/internal/handler"
	"github.com/xavierli/network-ticket/internal/middleware"
	"github.com/xavierli/network-ticket/internal/nonce"
	"github.com/xavierli/network-ticket/internal/repository"
	"github.com/xavierli/network-ticket/internal/service"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// ---------------------------------------------------------------------------
	// 1. Load configuration
	// ---------------------------------------------------------------------------
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// ---------------------------------------------------------------------------
	// 2. Initialize logger
	// ---------------------------------------------------------------------------
	logger, err := config.InitLogger(&cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	logger.Info("network-ticket server starting",
		zap.Int("port", cfg.Server.Port),
		zap.String("mode", cfg.Server.Mode),
	)

	// ---------------------------------------------------------------------------
	// 3. Connect to database
	// ---------------------------------------------------------------------------
	db, err := repository.NewDB(&cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect database", zap.Error(err))
	}
	defer db.Close()

	// ---------------------------------------------------------------------------
	// 4. Create repositories
	// ---------------------------------------------------------------------------
	ticketRepo := repository.NewTicketRepo(db)
	workflowStateRepo := repository.NewWorkflowStateRepo(db)
	alertSourceRepo := repository.NewAlertSourceRepo(db)
	alertRecordRepo := repository.NewAlertRecordRepo(db)
	clientRepo := repository.NewClientRepo(db)
	ticketLogRepo := repository.NewTicketLogRepo(db)
	auditLogRepo := repository.NewAuditLogRepo(db)
	userRepo := repository.NewUserRepo(db)

	// ---------------------------------------------------------------------------
	// 5. Create nonce store
	// ---------------------------------------------------------------------------
	var nonceStore nonce.Store
	switch cfg.Security.Nonce.Backend {
	case "file":
		fileStore, err := nonce.NewFileStore(cfg.Security.Nonce.File.Path)
		if err != nil {
			logger.Fatal("failed to create file nonce store", zap.Error(err))
		}
		nonceStore = fileStore
	default:
		// Default to database-backed nonce store.
		nonceStore = nonce.NewDBStore(db)
	}

	// ---------------------------------------------------------------------------
	// 6. Create services
	// ---------------------------------------------------------------------------
	ticketService := service.NewTicketService(
		ticketRepo,
		workflowStateRepo,
		ticketLogRepo,
		auditLogRepo,
		alertRecordRepo,
		logger,
	)

	alertService := service.NewAlertService(
		alertSourceRepo,
		ticketService,
		logger,
	)

	authService := service.NewAuthService(
		userRepo,
		&cfg.JWT,
	)

	clientService := service.NewClientService(
		clientRepo,
		logger,
	)

	// ---------------------------------------------------------------------------
	// 7. Create and start worker pool
	// ---------------------------------------------------------------------------
	retryCfg := client.RetryConfig{
		MaxAttempts:  cfg.Worker.RetryMax,
		BaseInterval: cfg.Worker.RetryBaseInterval,
		MaxInterval:  cfg.Worker.RetryMaxInterval,
	}
	if retryCfg.MaxAttempts <= 0 {
		retryCfg.MaxAttempts = 5
	}
	if retryCfg.BaseInterval <= 0 {
		retryCfg.BaseInterval = time.Second
	}
	if retryCfg.MaxInterval <= 0 {
		retryCfg.MaxInterval = 30 * time.Second
	}

	poolSize := cfg.Worker.PoolSize
	if poolSize <= 0 {
		poolSize = 4
	}

	workerPool := client.NewWorkerPool(
		poolSize,
		retryCfg,
		ticketRepo,
		clientRepo,
		workflowStateRepo,
		logger,
	)
	workerPool.Start()
	defer workerPool.Stop()

	// ---------------------------------------------------------------------------
	// 8. Create handlers
	// ---------------------------------------------------------------------------
	authHandler := handler.NewAuthHandler(authService, logger)
	alertHandler := handler.NewAlertHandler(alertService, alertSourceRepo, logger)
	ticketHandler := handler.NewTicketHandler(ticketService, logger)
	clientHandler := handler.NewClientHandler(clientService, logger)
	callbackHandler := handler.NewCallbackHandler(ticketService, logger)
	adminHandler := handler.NewAdminHandler(auditLogRepo, logger)

	// ---------------------------------------------------------------------------
	// 9. Set up Gin router
	// ---------------------------------------------------------------------------
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(logger))

	// Static files for frontend SPA.
	r.Static("/assets", "./frontend/dist/assets")
	r.NoRoute(func(c *gin.Context) {
		c.File("./frontend/dist/index.html")
	})

	// API routes.
	api := r.Group("/api/v1")
	{
		// Alert endpoints.
		api.POST("/alerts/webhook/:source_id", alertHandler.Webhook)

		alertsAuth := api.Group("")
		alertsAuth.Use(middleware.JWTAuth(authService))
		{
			alertsAuth.POST("/alerts", alertHandler.Create)
			alertsAuth.GET("/alert-sources", alertHandler.ListSources)
			alertsAuth.POST("/alert-sources", alertHandler.CreateSource)
			alertsAuth.PUT("/alert-sources/:id", alertHandler.UpdateSource)
			alertsAuth.DELETE("/alert-sources/:id", alertHandler.DeleteSource)
		}

		// Callback endpoint (HMAC-signed external client callbacks).
		api.POST("/callback/authorization",
			middleware.HMACSignature(clientRepo),
			middleware.NonceCheck(nonceStore),
			callbackHandler.Handle,
		)

		// Ticket endpoints.
		tickets := api.Group("")
		tickets.Use(middleware.JWTAuth(authService))
		{
			tickets.GET("/tickets", ticketHandler.List)
			tickets.GET("/tickets/:id", ticketHandler.Get)
			tickets.PUT("/tickets/:id", ticketHandler.Update)
			tickets.POST("/tickets/:id/retry", ticketHandler.Retry)
			tickets.POST("/tickets/:id/cancel", ticketHandler.Cancel)
		}

		// Client endpoints (admin-only for mutations).
		clients := api.Group("")
		clients.Use(middleware.JWTAuth(authService))
		{
			clients.GET("/clients", clientHandler.List)
		}
		clientsAdmin := api.Group("")
		clientsAdmin.Use(middleware.JWTAuth(authService), middleware.RequireAdmin())
		{
			clientsAdmin.POST("/clients", clientHandler.Create)
			clientsAdmin.PUT("/clients/:id", clientHandler.Update)
			clientsAdmin.DELETE("/clients/:id", clientHandler.Delete)
		}

		// Audit log endpoint.
		auditLogs := api.Group("")
		auditLogs.Use(middleware.JWTAuth(authService))
		{
			auditLogs.GET("/audit-logs", adminHandler.ListAuditLogs)
		}

		// Auth endpoint.
		api.POST("/auth/login", authHandler.Login)
	}

	// ---------------------------------------------------------------------------
	// 10. Start HTTP server with graceful shutdown
	// ---------------------------------------------------------------------------
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		logger.Info("HTTP server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutting down server", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}
