package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shopspring/decimal"

	"github.com/al-hudhud/backend/internal/accounting"
	"github.com/al-hudhud/backend/internal/admin"
	adminAlerts "github.com/al-hudhud/backend/internal/admin/alerts"
	adminDist "github.com/al-hudhud/backend/internal/admin/distribution"
	adminHealth "github.com/al-hudhud/backend/internal/admin/health"
	adminMetrics "github.com/al-hudhud/backend/internal/admin/metrics"
	adminRating "github.com/al-hudhud/backend/internal/admin/rating"
	adminRouting "github.com/al-hudhud/backend/internal/admin/routing"
	"github.com/al-hudhud/backend/internal/agent"

	"github.com/al-hudhud/backend/internal/alerting"
	"github.com/al-hudhud/backend/internal/api"
	"github.com/al-hudhud/backend/internal/audit"
	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/balance"
	"github.com/al-hudhud/backend/internal/billing"
	"github.com/al-hudhud/backend/internal/cache"
	"github.com/al-hudhud/backend/internal/channel"
	"github.com/al-hudhud/backend/internal/config"
	"github.com/al-hudhud/backend/internal/contact"
	"github.com/al-hudhud/backend/internal/db"
	"github.com/al-hudhud/backend/internal/finance"
	"github.com/al-hudhud/backend/internal/device"
	"github.com/al-hudhud/backend/internal/dispatcher"
	"github.com/al-hudhud/backend/internal/dlr"
	"github.com/al-hudhud/backend/internal/excelimport"
	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/al-hudhud/backend/internal/middleware"
	"github.com/al-hudhud/backend/internal/orchestrator"
	"github.com/al-hudhud/backend/internal/payment"
	"github.com/al-hudhud/backend/internal/provider"
	"github.com/al-hudhud/backend/internal/queue"
	"github.com/al-hudhud/backend/internal/receiver"
	"github.com/al-hudhud/backend/internal/redis"
	"github.com/al-hudhud/backend/internal/referral"
	"github.com/al-hudhud/backend/internal/reseller"
	"github.com/al-hudhud/backend/internal/rollover"
	"github.com/al-hudhud/backend/internal/staff"
	"github.com/al-hudhud/backend/internal/subscription"
	"github.com/al-hudhud/backend/internal/support"
	"github.com/al-hudhud/backend/internal/tenant"
	"github.com/al-hudhud/backend/internal/worker"
	"github.com/al-hudhud/backend/internal/workers/alhudhud"
	"github.com/al-hudhud/backend/internal/workers/private"
	"github.com/al-hudhud/backend/internal/ws"
	"github.com/al-hudhud/backend/pkg/jwt"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const Version = "3.2.7"

func init() {
	// Register the River migrator in the db package to avoid circular dependency
	db.MigrateRiver = func(tenantID uuid.UUID) error {
		schemaName := db.GetSchemaName(tenantID)
		return queue.MigrateSchema(context.Background(), schemaName)
	}
}

func main() {
	// Initialize Global Logger
	logger.InitLogger()
	log := logger.GetLogger()
	defer logger.Sync()

	log.Info("Starting ALHudhudAI Backend - Global Standard Edition")
	log.Info("Current APP_ENV", zap.String("app_env", os.Getenv("APP_ENV")))

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Warn(".env file not found, using system environment variables")
	}

	// Load and validate configuration early (fail fast)
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Configuration validation failed", zap.Error(err))
	}

	// Log configuration (without secrets)
	log.Info("Configuration loaded successfully",
		zap.String("db_host", cfg.Database.Host),
		zap.String("db_port", cfg.Database.Port),
		zap.String("db_name", cfg.Database.Name),
		zap.String("redis_host", cfg.Redis.Host),
		zap.String("redis_port", cfg.Redis.Port),
		zap.String("server_port", cfg.Server.Port),
		zap.String("app_env", cfg.Server.AppEnv),
		zap.Int("batch_size", cfg.BatchProcess.BatchSize),
		zap.Int("worker_count", cfg.WorkerPool.WorkerCount),
	)

	// Initialize JWT Secret
	jwt.SetJWTSecret(cfg.JWT.Secret)
	log.Info("JWT configuration loaded", zap.Int("expiry_hours", cfg.JWT.ExpiryHours))

	// Initialize Database
	log.Info("Initializing database connection...")
	db.InitDB()
	db.InitGormDB()

	// Initialize Finance Bridge for the isolated finance microservice
	finance.Init(cfg.Finance.ServiceURL, db.DB)

	// Seed Database (Idempotent)
	db.SeedDatabase()

	// Initialize Redis for Metrics
	redisClient, err := redis.NewRedisClient(cfg, log)
	if err != nil {
		log.Warn("Failed to initialize Redis client, metrics collection will be limited", zap.Error(err))
	}

	// Initialize Fast Processing Components
	var messageQueue *redis.MessageQueue
	var channelCache *redis.ChannelCache
	var pricingCache *redis.PricingCache

	if redisClient != nil {
		messageQueue = redis.NewMessageQueue(redisClient)
		channelCache = redis.NewChannelCache(redisClient)
		pricingCache = redis.NewPricingCache(redisClient)

		// Warmup pricing cache with common values
		if err := pricingCache.Warmup(context.Background(), 0.05); err != nil {
			log.Warn("Failed to warmup pricing cache", zap.Error(err))
		} else {
			log.Info("Pricing cache warmed up successfully")
		}
	}

	// Initialize Balance Manager and Accounting Service (Requirement 1.1)
	balanceManager := balance.NewManager(db.PgxPool, log)
	accountingSvc := accounting.NewService(balanceManager)

	// Initialize WebSocket Global Hub (Phase 3: Hybrid Queue support)
	ws.InitGlobalHub(cfg.WebSocket, accountingSvc, redisClient)

	// Start the queued-message flusher: re-dispatches stale QUEUED messages
	// to connected mobile devices so delivery continues as long as a phone is online
	device.StartQueuedMessageFlusher(context.Background(), 10*time.Second, 30*time.Second)

	// Initialize Metrics Collector
	var metricsCollector *metrics.Collector
	if redisClient != nil {
		metricsCollector = metrics.NewCollector(db.DB, redisClient.Client)
	} else {
		metricsCollector = metrics.NewCollector(db.DB, nil)
	}
	go metricsCollector.Start(context.Background(), 10*time.Second)

	// Initialize Global Provider Registry
	provider.InitGlobalRegistry()

	// Start Background Services - New Tenant Isolation Architecture
	dispatcherManager := dispatcher.NewManager(redisClient)
	dispatcherManager.Start()

	msgOrchestrator := orchestrator.NewOrchestrator()
	msgOrchestrator.Start()

	// Initialize Query Cache
	var queryCache *cache.QueryCache
	if redisClient != nil {
		queryCache = cache.NewQueryCache(redisClient.Client)
	}

	// Registration Cleanup Job
	tenantSvc := tenant.NewService(queryCache)
	go func() {
		log.Info("Registration cleanup job started")
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			tenantSvc.CleanupExpiredRegistrations()
		}
	}()

	// Phase 3: Admin Statistics Refresher (runs every 60 seconds)
	go admin.StartStatisticsRefresher(context.Background(), 60*time.Second)

	// Session Cleanup Job
	auth.StartSessionCleanupJob(context.Background(), log, 1*time.Hour)

	// Initialize Channel Handler first so it can be used for health checks
	channelHandler := channel.NewHandler(channelCache, log)

	// Start Channel Health Check (Bug 13 - Task 15.3)
	// This proactively detects offline channels and invalidates cache every 1 minute
	if channelHandler != nil {
		channelHandler.GetService().StartHealthCheck(context.Background())
		log.Info("Channel health check started")
	}

	// Initialize Staff Service
	staffSvc := staff.NewService(queryCache)

	// Initialize Referral Manager (Requirement 2.3)
	var referralManager *referral.Manager
	if cfg.Referral.Enabled {
		// Convert ReferralConfig from config to referral.ReferralConfig
		referralConfig := &referral.ReferralConfig{
			DefaultRewardType:      referral.RewardTypeBalance,
			DefaultRewardAmount:    10.0,
			DefaultRewardCondition: referral.RewardConditionFirstPayment,
			CodePrefix:             "REF",
			CodeLength:             cfg.Referral.CodeLength,
			MaxReferralsPerUser:    cfg.Referral.MaxUsesPerCode,
			Enabled:                cfg.Referral.Enabled,
		}

		sqlxDB := sqlx.NewDb(db.DB, "postgres")
		referralManager = referral.NewManager(sqlxDB, referralConfig)

		// Set Redis client for caching (if available)
		if redisClient != nil {
			referralManager.SetRedisClient(redisClient.Client)
			log.Info("Referral system initialized with Redis caching",
				zap.Bool("enabled", cfg.Referral.Enabled),
				zap.Int("code_length", cfg.Referral.CodeLength),
				zap.Int("max_uses", cfg.Referral.MaxUsesPerCode),
				zap.String("cache_ttl", "5m"),
			)
		} else {
			log.Info("Referral system initialized without caching (Redis unavailable)",
				zap.Bool("enabled", cfg.Referral.Enabled),
				zap.Int("code_length", cfg.Referral.CodeLength),
				zap.Int("max_uses", cfg.Referral.MaxUsesPerCode),
			)
		}
	} else {
		log.Info("Referral system is disabled")
	}

	// Initialize Handlers
	authHandler := auth.NewHandler()
	auditLogger := audit.NewLogger(db.DB)
	auditHandler := audit.NewHandler(auditLogger)
	staffHandler := staff.NewHandler(staffSvc)
	receiverHandler := receiver.NewHandler(redisClient)
	// Excel import from the mobile app — isolated feature (internal/excelimport).
	excelHandler := excelimport.NewHandler(receiver.NewService(redisClient), excelimport.NewStore())
	tenantHandler := tenant.NewHandler(tenantSvc, balanceManager)
	deviceHandler := device.NewHandler()
	metricsHandler := metrics.NewHandler()
	// channelHandler already initialized above for health check
	invoiceHandler := api.NewInvoiceHandler(db.DB, log, "storage/invoices")

	contactService := contact.NewService()
	contactHandler := contact.NewHandler(contactService)

	// Initialize Referral Handler (Requirement 2.3)
	var referralHandler *referral.Handler
	if referralManager != nil {
		referralHandler = referral.NewHandler(referralManager, log)
		log.Info("Referral handler initialized")
	}

	// Advanced Admin Modules Initialization
	sqlxDB := sqlx.NewDb(db.DB, "postgres")

	ratingEngine := adminRating.NewDefaultCalculator()
	metricsRepo, err := adminMetrics.NewRepository(sqlxDB)
	if err != nil {
		log.Fatal("Failed to init admin metrics repo", zap.Error(err))
	}

	routingConfigManager := adminRouting.NewRoutingConfigManager(sqlxDB)
	routingMetricsTracker := adminRouting.NewMetricsTracker(sqlxDB)
	routingMetricsTracker.Start(context.Background())
	routingEngine := adminRouting.NewRoutingEngine(sqlxDB, routingConfigManager, routingMetricsTracker)

	deviceFailureHandler := adminDist.NewDeviceFailureHandler()
	poolManager := adminDist.NewDevicePoolManager(db.GormDB, deviceFailureHandler)
	strategySelector := adminDist.NewStrategySelector(db.GormDB, poolManager, deviceFailureHandler)

	alertManager := adminAlerts.NewManager(sqlxDB, log)

	// Accounting (cashboxes & vouchers) manager
	accountingManager := accounting.NewManager(sqlxDB, balanceManager, log)

	// Payment Gateway clients + manager (Requirements 39.x)
	stripeClient := payment.NewStripeClient(
		os.Getenv("STRIPE_API_KEY"),
		os.Getenv("STRIPE_WEBHOOK_SECRET"),
	)
	paypalClient := payment.NewPayPalClient(
		os.Getenv("PAYPAL_CLIENT_ID"),
		os.Getenv("PAYPAL_CLIENT_SECRET"),
		os.Getenv("PAYPAL_SANDBOX") == "true",
	)

	// balanceTopup credits the tenant wallet after a successful gateway payment.
	balanceTopup := func(ctx context.Context, tenantID uuid.UUID, amount float64, currency, reason string) error {
		_, err := balanceManager.CreditBalance(ctx, balance.BalanceOperation{
			TenantID:  tenantID,
			Amount:    decimal.NewFromFloat(amount),
			Type:      balance.OperationTypeCharge,
			Reason:    reason,
			CreatedBy: "payment-gateway",
		})
		return err
	}

	// notifier emits a log-level notification for payment events (WS/email stub).
	notifier := func(ctx context.Context, tenantID uuid.UUID, notificationType, message string) error {
		log.Info("Payment notification",
			zap.String("tenant_id", tenantID.String()),
			zap.String("type", notificationType),
			zap.String("message", message),
		)
		return nil
	}

	paymentManager := payment.NewManager(sqlxDB, log, stripeClient, paypalClient, balanceTopup, notifier)
	paymentHandler := api.NewPaymentHandler(paymentManager, stripeClient, paypalClient)

	// Health Checker sub-components
	healthClassifier := adminHealth.NewStatusClassifier()
	healthTrigger := adminHealth.NewAlertTrigger(sqlxDB)
	healthHistory := adminHealth.NewHistoryLogger(sqlxDB)
	healthChecker := adminHealth.NewHealthChecker(sqlxDB, healthClassifier, healthTrigger, healthHistory)
	healthChecker.Start(context.Background())

	// Initialize UNIFIED Alerting System (Bridge transient and persistent)
	systemAlertManager, err := alerting.Initialize(log, alertManager)
	if err != nil {
		log.Warn("Failed to initialize alerting system, continuing without alerts", zap.Error(err))
		systemAlertManager = nil
	} else {
		log.Info("Unified system alerting initialized successfully")
	}

	// Start balance monitoring (Bug 24)
	if systemAlertManager != nil {
		go monitorLowBalance(context.Background(), balanceManager, systemAlertManager, log)
		log.Info("Balance monitoring started")
	}

	// Start Fast Message Processor (Redis-based ultra-fast processing)
	if redisClient != nil && messageQueue != nil && channelCache != nil {
		fastProcessor := worker.NewFastMessageProcessor(
			messageQueue,
			channelCache,
			balanceManager,
			ws.GlobalHub,
			systemAlertManager,
			log,
			cfg.WorkerPool.BatchSize,
			cfg.WorkerPool.WorkerCount,
		)
		fastProcessor.Start(context.Background())
		log.Info("Fast Message Processor started successfully",
			zap.Int("batch_size", cfg.WorkerPool.BatchSize),
			zap.Int("worker_count", cfg.WorkerPool.WorkerCount),
			zap.Int("ticker_ms", cfg.WorkerPool.TickerMS),
		)

		// Start monitoring goroutines for alerting (Bug 24)
		if systemAlertManager != nil {
			go fastProcessor.MonitorQueueHealth(context.Background())
			go fastProcessor.MonitorErrorRates(context.Background())
			log.Info("Alert monitoring started for queue health and error rates")
		}

		// Start Alhudhud Official Gateway Worker
		alhudhudWorker := alhudhud.NewWorker(redisClient, ws.GlobalHub)
		alhudhudWorker.Start()

		// Start Private Gateway Worker
		privateWorker := private.NewWorker(redisClient, ws.GlobalHub)
		privateWorker.Start()
	}

	// Initialize Granular Admin Handlers
	adminAlertsH := api.NewAdminAlertsHandler(alertManager)
	adminAccountingH := api.NewAdminAccountingHandler(accountingManager)
	adminDashboardH := api.NewAdminDashboardHandler(ratingEngine, metricsRepo, routingEngine, poolManager, healthChecker, alertManager)
	adminHealthH := api.NewAdminHealthHandler(healthChecker)
	adminMetricsH := api.NewAdminMetricsHandler(metricsRepo)
	adminDistH := api.NewAdminDistributionHandler(poolManager, strategySelector)
	adminRatingH := api.NewAdminRatingHandler(ratingEngine)
	adminRoutingH := api.NewAdminRoutingHandler(routingEngine, routingConfigManager)

	adminTariffH := api.NewAdminTariffHandler(db.GormDB)
	adminGatewayH := api.NewAdminGatewayHandler(alertManager)
	adminReportH := api.NewAdminReportHandler()

	// Subscription & Package Management (Requirements 3.6, 4.x, 18.x, 21.x)
	subscriptionManager := subscription.NewManager(db.PgxPool, log)
	rolloverManager := rollover.NewManager(db.PgxPool, log)
	packageHandler := api.NewPackageHandler(db.PgxPool, log)
	subscriptionHandler := api.NewSubscriptionHandler(subscriptionManager, rolloverManager, db.PgxPool, log)
	adminPackageH := api.NewAdminPackageHandler(db.PgxPool, log, packageHandler)
	adminSubscriptionH := api.NewAdminSubscriptionHandler(subscriptionManager, db.PgxPool, log)

	resellerSvc := reseller.NewService(db.GormDB, db.PgxPool, balanceManager, subscriptionManager, log)
	resellerHandler := reseller.NewHandler(resellerSvc)

	// Auto-renewal service (Requirement 14.x): periodic check of expired subscriptions.
	// River is not initialized in this process, so the client is nil (unused by the service).
	autoRenewalSvc := billing.NewAutoRenewalService(db.PgxPool, nil, log)
	go startAutoRenewalChecker(context.Background(), autoRenewalSvc, log)

	ticketManager := support.NewTicketManager(db.DB)
	supportHandler := support.NewHandler(ticketManager)
	dlrHandler := dlr.NewHandler()
	privateGatewayHandler := api.NewPrivateGatewayHandler(ws.GlobalHub)
	csrfHandler := api.NewCSRFHandler()

	gin.SetMode(gin.ReleaseMode)
	if os.Getenv("APP_ENV") == "development" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Structured Logging Middleware for Gin
	r.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		log.Info("Request Handled",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", time.Since(start)),
		)
	})

	// CORS Configuration
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// Allow all origins dynamically to support local file testing (like test.html)
			// and web dashboard, while working properly with AllowCredentials: true
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Accounting-Key", "X-CSRF-Token", "X-API-Key", "X-Idempotency-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Root handler
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":  "Welcome to ALHudhudAI API - Isolated & Secure",
			"version":  "3.2.7",
			"status":   "UP",
			"health":   "/health",
			"frontend": "http://localhost:5173",
		})
	})

	// Add WebSocket route
	// Note: Protected by JWT token query parameter
	r.GET("/ws", func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			// Try cookie if query param is missing
			if cookie, err := c.Cookie("access_token"); err == nil {
				token = cookie
			}
		}

		if token == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Token required"})
			return
		}

		claims, err := jwt.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		tenantID := claims.TenantID
		if tenantID == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tenant ID missing from token"})
			return
		}

		ws.ServeWs(ws.GlobalHub, c.Writer, c.Request, tenantID, uuid.Nil, "", false)

		// Trigger Wakeup for Wait Mode
		uid, _ := uuid.Parse(tenantID)
		deviceSvc := device.NewService()
		go deviceSvc.WakeUpTenant(context.Background(), uid)
	})

	// Add WebSocket pairing endpoint for mobile devices
	r.GET("/ws/pair", func(c *gin.Context) {
		apiKey := c.Query("api_key")
		deviceID := c.Query("device_id")
		if deviceID == "" {
			// Mobile app's reconnect path uses camelCase deviceId without api_key
			deviceID = c.Query("deviceId")
		}

		if deviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "device_id and api_key are required"})
			return
		}

		// Validate device credentials and get tenant ID
		deviceSvc := device.NewService()
		var tenantID uuid.UUID
		var deviceUUID uuid.UUID
		var err error

		if apiKey != "" {
			tenantID, deviceUUID, err = deviceSvc.GetTenantIDByAPIKey(context.Background(), apiKey, deviceID)
		} else {
			// Previously paired device reconnecting without api_key
			tenantID, deviceUUID, err = deviceSvc.GetTenantByDeviceID(context.Background(), deviceID)
		}

		if err != nil {
			log.Error("Failed to validate device credentials",
				zap.String("device_id", deviceID),
				zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		log.Info("Device connecting via WebSocket",
			zap.String("tenant_id", tenantID.String()),
			zap.String("channel_id", deviceUUID.String()),
			zap.String("device_id", deviceID))

		ws.ServeWs(ws.GlobalHub, c.Writer, c.Request, tenantID.String(), deviceUUID, deviceID, true)

		// Trigger Wakeup for Wait Mode
		go deviceSvc.WakeUpTenant(context.Background(), tenantID)

		// Redeliver any QUEUED messages to the freshly connected device(s)
		go func(tenant uuid.UUID) {
			time.Sleep(2 * time.Second)
			if err := deviceSvc.RedeliverQueuedMessages(context.Background(), tenant, 0); err != nil {
				log.Warn("Failed to redeliver queued messages after device connect",
					zap.String("tenant_id", tenant.String()),
					zap.Error(err))
			}
		}(tenantID)
	})

	// Reconnection and Monitoring for WS is now handled inside InitGlobalHub

	// API Routes with Rate Limiting
	// Apply rate limiting to all /api/v1 routes: 100 requests per minute per IP
	// Implements automatic fallback to in-memory rate limiter when Redis is unavailable (Requirement 2.5)
	var rateLimiter gin.HandlerFunc
	if redisClient != nil {
		rateLimiter = middleware.NewRateLimiter(redisClient.Client, 100, time.Minute, log)
		log.Info("Rate limiting enabled for API routes (Redis-based)",
			zap.Int("limit", 100),
			zap.String("window", "1 minute"),
			zap.String("mode", "redis"))
	} else {
		// If Redis is not available, use in-memory rate limiter fallback (Requirement 2.5)
		rateLimiter = middleware.NewMemoryRateLimiter(100, log)
		log.Warn("Rate limiting enabled with in-memory fallback (Redis unavailable)",
			zap.Int("limit", 100),
			zap.String("window", "1 minute"),
			zap.String("mode", "in-memory"))
	}

	api := r.Group("/api/v1")
	api.Use(rateLimiter) // Apply rate limiting to all API routes
	{
		// Health Check
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"service": "ALHudhudAI Backend",
				"version": "3.2.7",
			})
		})

		// CSRF Token Endpoint (no authentication required, but rate-limited)
		// This endpoint provides CSRF tokens to authenticated clients
		api.GET("/csrf-token", csrfHandler.GetCSRFToken)

		// Payment Gateway Webhooks (no authentication - signature/event verified in handler)
		api.POST("/webhooks/stripe", paymentHandler.StripeWebhook)
		api.POST("/webhooks/paypal", paymentHandler.PayPalWebhook)

		// Metrics & Monitoring (Tenant Isolated)
		authenticated := api.Group("/")
		authenticated.Use(auth.AuthMiddleware())
		{
			authenticated.GET("/metrics", metricsHandler.GetMetricsHandler)
			authenticated.GET("/system/health", metricsHandler.GetHealthHandler)

			// Audit Logs (Only for staff with permission)
			authenticated.GET("/audit", auth.RequireUserType("SUPER_ADMIN", "STAFF"), auditHandler.GetAuditLogs)
		}

		// Auth Routes
		authGroup := api.Group("/auth")
		{
			if redisClient != nil {
				authGroup.POST("/login", auth.IPRateLimiter(redisClient.Client, 10, 5*time.Minute), authHandler.Login)
			} else {
				authGroup.POST("/login", auth.IPRateLimiter(nil, 10, 5*time.Minute), authHandler.Login)
			}
			authGroup.POST("/refresh", authHandler.RefreshToken)

			authProtected := authGroup.Group("/")
			authProtected.Use(auth.AuthMiddleware(), middleware.CSRFProtection())
			{
				authProtected.POST("/logout", authHandler.Logout)
				authProtected.GET("/me", authHandler.Me)
				authProtected.POST("/change-password", authHandler.ChangePassword)
			}
			// Google Auth Placeholder
			authGroup.POST("/google", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Google Authentication coming soon"})
			})
		}

		// Tenant Registration (Multi-step)
		tenantAuth := api.Group("/tenants")
		{
			if redisClient != nil {
				tenantAuth.POST("/register/step1", auth.IPRateLimiter(redisClient.Client, 5, 10*time.Minute), tenantHandler.RegisterStep1)
				tenantAuth.POST("/verify-otp", auth.IPRateLimiter(redisClient.Client, 10, 10*time.Minute), tenantHandler.VerifyOTP)
				tenantAuth.POST("/register/step2", tenantHandler.RegisterStep2)
				tenantAuth.POST("/register", auth.IPRateLimiter(redisClient.Client, 5, 10*time.Minute), tenantHandler.Register)
			} else {
				tenantAuth.POST("/register/step1", auth.IPRateLimiter(nil, 5, 10*time.Minute), tenantHandler.RegisterStep1)
				tenantAuth.POST("/verify-otp", auth.IPRateLimiter(nil, 10, 10*time.Minute), tenantHandler.VerifyOTP)
				tenantAuth.POST("/register/step2", tenantHandler.RegisterStep2)
				tenantAuth.POST("/register", auth.IPRateLimiter(nil, 5, 10*time.Minute), tenantHandler.Register)
			}
		}

		// Staff Management (Tenant and Super Admin)
		staffGroup := api.Group("/staff")
		staffGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			staffGroup.GET("/", staffHandler.ListStaff)
			staffGroup.POST("/create", staffHandler.CreateStaff)
			staffGroup.PUT("/update", staffHandler.UpdateStaff)
			staffGroup.DELETE("/delete", staffHandler.DeleteStaff)

			// Role Management
			staffGroup.GET("/roles", staffHandler.ListRoles)
			staffGroup.POST("/roles/create", staffHandler.CreateRole)
			staffGroup.PUT("/roles/update", staffHandler.UpdateRole)
			staffGroup.DELETE("/roles/delete", staffHandler.DeleteRole)

			// Permissions
			staffGroup.GET("/permissions", staffHandler.ListPermissions)

			staffGroup.POST("/bulk-update", staffHandler.BulkUpdateStaff)
		}

		// Invoice Management (Tenant and Super Admin)
		invoiceGroup := api.Group("/invoices")
		invoiceGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB))
		{
			invoiceGroup.GET("/", invoiceHandler.ListInvoices)
			invoiceGroup.GET("/:id", invoiceHandler.GetInvoice)
			invoiceGroup.GET("/:id/pdf", invoiceHandler.DownloadInvoicePDF)
		}

		// Device Management (Tenant only)
		deviceGroup := api.Group("/devices")
		deviceGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			deviceGroup.POST("/pair", deviceHandler.Pair)
			deviceGroup.GET("/list", deviceHandler.List)
			deviceGroup.GET("/fetch", deviceHandler.FetchMessagesHandler)
			deviceGroup.POST("/report", deviceHandler.ReportStatusHandler)
			deviceGroup.POST("/contacts", contactHandler.SyncContactsHandler)
			deviceGroup.DELETE("/:id", deviceHandler.Delete)
			deviceGroup.GET("/:id/ping", deviceHandler.Ping)
		}

		// Contact Management (Tenant only)
		contactGroup := api.Group("/contacts")
		contactGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			contactGroup.GET("/", contactHandler.ListContacts)
			contactGroup.POST("/", contactHandler.CreateContact)
			contactGroup.PUT("/", contactHandler.UpdateContact)
			contactGroup.DELETE("/:id", contactHandler.DeleteContact)
		}

		// Settings Operations (Tenant isolated)
		settingsGroup := api.Group("/settings")
		settingsGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			settingsGroup.POST("/api-key/regenerate", tenantHandler.RegenerateAPIKey)
		}

		// Tenant Dashboard & Finance (Tenant Only)
		tenantGroup := api.Group("/tenant")
		tenantGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB))
		{
			tenantGroup.GET("/dashboard", tenantHandler.GetDashboard)
		}

		financeGroup := api.Group("/finance")
		financeGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB))
		{
			financeGroup.GET("/balance", tenantHandler.GetBalance)
		}

		// Payments (Tenant and Super Admin) - Requirements 39.x
		paymentsGroup := api.Group("/payments")
		paymentsGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			paymentsGroup.POST("/initiate", paymentHandler.InitiatePayment)
			paymentsGroup.GET("/:transaction_id", paymentHandler.GetPaymentStatus)
			paymentsGroup.GET("", paymentHandler.ListPaymentTransactions)
			paymentsGroup.POST("/paypal/capture/:order_id", paymentHandler.PayPalCapturePayment)
		}

		// Packages (Tenant and Super Admin) - Requirements 3.6, 18.5, 18.6
		packagesGroup := api.Group("/packages")
		packagesGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			packagesGroup.GET("/", packageHandler.ListPackages)
			packagesGroup.GET("/:id", packageHandler.GetPackage)
			packagesGroup.POST("/", packageHandler.CreatePackage)
			packagesGroup.PUT("/:id", packageHandler.UpdatePackage)
			packagesGroup.DELETE("/:id", packageHandler.DeletePackage)
		}

		// Subscriptions (Tenant and Super Admin) - Requirements 4.x, 21.x
		subscriptionsGroup := api.Group("/subscriptions")
		subscriptionsGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			subscriptionsGroup.POST("/", subscriptionHandler.Subscribe)
			subscriptionsGroup.GET("/usage", subscriptionHandler.GetUsage)
			subscriptionsGroup.GET("/active", subscriptionHandler.GetSubscription)
			subscriptionsGroup.PUT("/:id/cancel", subscriptionHandler.CancelSubscription)
			subscriptionsGroup.PUT("/:id/enable-auto-renew", subscriptionHandler.EnableAutoRenew)
		}

		supportTenantGroup := api.Group("/support")
		supportTenantGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB))
		{
			supportTenantGroup.GET("/chat", supportHandler.GetTenantChat)
			supportTenantGroup.POST("/tickets/:id/reply", supportHandler.ReplyToTicket)
		}

		// Messaging Engine (Tenant Only - Supports JWT and API Keys)
		// Note: CSRF protection skips API key authentication automatically
		messagingGroup := api.Group("/messaging")
		messagingGroup.Use(auth.APIKeyMiddleware(tenantSvc, redisClient), auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			messagingGroup.POST("/send", receiverHandler.SendMessageHandler)
			messagingGroup.POST("/bulk-send", receiverHandler.BulkSendMessageHandler)
			messagingGroup.GET("/bulk/:id/progress", receiverHandler.BulkProgressHandler)
			messagingGroup.POST("/ingest", receiverHandler.IngestMessageHandler)
			messagingGroup.POST("/excel-ingest", receiverHandler.BulkIngestExcelHandler)
			messagingGroup.POST("/excel-upload", excelHandler.Upload)
			messagingGroup.POST("/excel-upload/confirm", excelHandler.Confirm)
			messagingGroup.GET("/list", receiverHandler.ListMessagesHandler)
			messagingGroup.POST("/bulk-delete", receiverHandler.BulkDeleteMessagesHandler)
			messagingGroup.POST("/bulk-resend", receiverHandler.BulkResendMessagesHandler)
			messagingGroup.GET("/incoming", receiverHandler.ListIncomingMessagesHandler)
		}

		// Private Gateway Webhook (External API - No CSRF)
		privateGatewayGroup := api.Group("/private-gateway")
		privateGatewayGroup.Use(auth.APIKeyMiddleware(tenantSvc, redisClient), auth.TenantIsolationMiddleware())
		{
			privateGatewayGroup.POST("/dlr", privateGatewayHandler.HandleDLR)
		}

		// Monitoring Routes (Super Admin only)
		monitoringHandler := dispatcher.NewMonitoringHandler(dispatcherManager)
		monitoringGroup := api.Group("/monitoring")
		monitoringGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("SUPER_ADMIN"), middleware.PreventCrossTenantAccess(db.GormDB))
		{
			monitoringGroup.GET("/stats", monitoringHandler.GetStats)
			monitoringGroup.GET("/health", monitoringHandler.HealthCheck)
			monitoringGroup.GET("/tenant/:tenant_id", monitoringHandler.GetTenantStats)
		}

		// Admin Routes (Super Admin only)
		adminGroup := api.Group("/admin")
		adminGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("SUPER_ADMIN"), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			// Tenant Management
			adminGroup.GET("/tenants", tenantHandler.ListTenants)
			adminGroup.POST("/tenants", tenantHandler.AdminCreateTenant)
			adminGroup.GET("/tenants/:id/subscription", tenantHandler.GetTenantSubscription)
			adminGroup.POST("/tenants/:id/subscribe", adminSubscriptionH.Subscribe)
			adminGroup.POST("/tenants/:id/balance", tenantHandler.AdminAddBalance)
			adminGroup.PUT("/tenants/:id/subscription/auto-renew", adminSubscriptionH.SetAutoRenew)
			adminGroup.GET("/tenants/:id/transactions", tenantHandler.GetTenantTransactions)
			adminGroup.GET("/tenants/:id/impersonate", tenantHandler.ImpersonateTenant)
			adminGroup.POST("/tenants/bulk-update", tenantHandler.BulkUpdateTenants)
			adminGroup.POST("/tenants/bulk-delete", tenantHandler.BulkDeleteTenants)
			adminGroup.GET("/resellers", resellerHandler.ListResellers)
			adminGroup.GET("/resellers/:id", resellerHandler.GetResellerDetail)
			adminGroup.POST("/resellers/:id/credit", resellerHandler.CreditReseller)

			// Dashboard & Stats
			adminGroup.GET("/stats", adminDashboardH.GetDashboardOverview)
			adminGroup.GET("/dashboard", adminDashboardH.GetDashboardOverview)
			adminGroup.GET("/metrics/:tenant_id", adminMetricsH.GetCompanyMetrics)
			adminGroup.GET("/companies/:tenant_id/rating", adminRatingH.GetCompanyRating)

			// Infrastructure & Health
			adminGroup.GET("/health", adminHealthH.GetServers)
			adminGroup.GET("/health/servers", adminHealthH.GetServers)
			adminGroup.GET("/health/servers/:server_id", adminHealthH.GetServerDetails)

			// Tariffs
			adminGroup.GET("/tariffs", adminTariffH.ListTariffs)
			adminGroup.POST("/tariffs", adminTariffH.CreateTariff)
			adminGroup.GET("/tariffs/:id", adminTariffH.GetTariff)
			adminGroup.PUT("/tariffs/:id", adminTariffH.UpdateTariff)
			adminGroup.DELETE("/tariffs/:id", adminTariffH.DeleteTariff)

			// Packages
			adminGroup.GET("/packages", adminPackageH.ListPackages)
			adminGroup.POST("/packages", adminPackageH.CreatePackage)
			adminGroup.GET("/packages/:id", adminPackageH.GetPackage)
			adminGroup.PUT("/packages/:id", adminPackageH.UpdatePackage)
			adminGroup.DELETE("/packages/:id", adminPackageH.DeletePackage)

			// Gateways (Dispatch Channels)
			adminGroup.GET("/gateways", adminGatewayH.ListGateways)
			adminGroup.GET("/gateways/alerts", adminGatewayH.GetGatewayAlerts)
			adminGroup.POST("/gateways", adminGatewayH.CreateGateway)
			adminGroup.GET("/gateways/:id/usage", adminGatewayH.GetGatewayUsage)

			// Alerts
			adminGroup.GET("/alerts", adminAlertsH.GetAlerts)
			adminGroup.GET("/alerts/:alert_id", adminAlertsH.GetAlert)
			adminGroup.PUT("/alerts/:alert_id/acknowledge", adminAlertsH.AcknowledgeAlert)
			adminGroup.PUT("/alerts/:alert_id/resolve", adminAlertsH.ResolveAlert)
			adminGroup.GET("/alerts/config", adminAlertsH.ConfigureAlerts)
			adminGroup.POST("/alerts/config", adminAlertsH.ConfigureAlerts)

			// Accounting (Cashboxes & Vouchers)
			adminGroup.GET("/accounting/cashboxes", adminAccountingH.ListCashboxes)
			adminGroup.POST("/accounting/cashboxes", adminAccountingH.CreateCashbox)
			adminGroup.GET("/accounting/cashboxes/:id", adminAccountingH.GetCashbox)
			adminGroup.PUT("/accounting/cashboxes/:id", adminAccountingH.UpdateCashbox)
			adminGroup.PUT("/accounting/cashboxes/:id/status", adminAccountingH.SetCashboxStatus)
			adminGroup.POST("/accounting/cashboxes/:id/statement", adminAccountingH.GetCashboxStatement)
			adminGroup.GET("/accounting/receipts", adminAccountingH.ListReceipts)
			adminGroup.POST("/accounting/receipts", adminAccountingH.CreateReceipt)
			adminGroup.GET("/accounting/receipts/:id", adminAccountingH.GetReceipt)
			adminGroup.POST("/accounting/receipts/:id/post", adminAccountingH.PostReceipt)
			adminGroup.POST("/accounting/receipts/:id/reverse", adminAccountingH.ReverseReceipt)
			adminGroup.GET("/accounting/payments", adminAccountingH.ListPayments)
			adminGroup.POST("/accounting/payments", adminAccountingH.CreatePayment)
			adminGroup.GET("/accounting/payments/:id", adminAccountingH.GetPayment)
			adminGroup.POST("/accounting/payments/:id/post", adminAccountingH.PostPayment)
			adminGroup.POST("/accounting/payments/:id/reverse", adminAccountingH.ReversePayment)
			adminGroup.GET("/accounting/numbering/:type", adminAccountingH.PreviewNextNumber)

			// Reports
			adminGroup.GET("/reports", adminReportH.GetReports)
			adminGroup.GET("/reports/export", adminReportH.ExportReport)

			// Routing
			adminGroup.GET("/routing/stats", adminRoutingH.GetRoutingStats)
			adminGroup.PUT("/routing/config", adminRoutingH.UpdateRoutingConfig)
			adminGroup.GET("/routing/:tenant_id", adminRoutingH.GetRoutingConfigByTenant)
			adminGroup.PUT("/routing/:tenant_id", adminRoutingH.UpdateRoutingConfigByTenant)
			adminGroup.GET("/routing/:tenant_id/metrics", adminRoutingH.GetRoutingMetricsByTenant)

			// Distribution (Pools)
			adminGroup.GET("/distribution/pools", adminDistH.GetPools)
			adminGroup.GET("/distribution/pools/:pool_id", adminDistH.GetPoolDetails)
			adminGroup.GET("/distribution/:tenant_id", adminDistH.GetConfig)
			adminGroup.PUT("/distribution/:tenant_id", adminDistH.UpdateConfig)
			adminGroup.GET("/distribution/:tenant_id/devices", adminDistH.GetDevices)
			adminGroup.GET("/distribution/:tenant_id/metrics", adminDistH.GetMetrics)

			// Support
			adminGroup.GET("/support/tickets", supportHandler.GetAllTickets)
		}

		// Phase 4: DLR (Delivery Report) endpoint - internal, protected by API key
		// No tenant auth: the gateway posts here with its API key directly.
		dlrGroup := api.Group("/internal")
		dlrGroup.Use(auth.APIKeyMiddleware(tenantSvc, redisClient))
		{
			dlrGroup.POST("/dlr", dlrHandler.ReceiveDLR)
		}

		// Channel (Gateway) Management Routes (Tenant only)
		channelGroup := api.Group("/channels")
		channelGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			channelGroup.GET("/list", channelHandler.List)
			channelGroup.POST("/create", channelHandler.Create)
			channelGroup.PUT("/:id", channelHandler.Update)
			channelGroup.PUT("/:id/key", channelHandler.RegenerateKey)
			channelGroup.DELETE("/:id", channelHandler.Delete)
		}

		// Referral System Routes (Tenant only) - Requirement 2.3
		if referralHandler != nil {
			referralGroup := api.Group("/referrals")
			referralGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("TENANT", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
			{
				referralGroup.POST("/generate", referralHandler.GenerateReferralCode)
				referralGroup.POST("/validate", referralHandler.ValidateReferralCode)
				referralGroup.POST("/process", referralHandler.ProcessReferral)
				referralGroup.GET("/stats/:tenantID", referralHandler.GetReferralStats)
				referralGroup.GET("/report/:tenantID", referralHandler.GetReferralReport)
			}
			log.Info("Referral API endpoints registered successfully")
		}

		// Reseller Routes (Software Companies)
		resellerGroup := api.Group("/reseller")
		resellerGroup.Use(auth.AuthMiddleware(), auth.RequireUserType("RESELLER", "SUPER_ADMIN"), auth.TenantIsolationMiddleware(), middleware.PreventCrossTenantAccess(db.GormDB), middleware.CSRFProtection())
		{
			resellerGroup.GET("/tenants", resellerHandler.GetTenants)
			resellerGroup.POST("/tenants", resellerHandler.CreateTenant)
			resellerGroup.GET("/balance", resellerHandler.GetWalletBalance)
			resellerGroup.GET("/tenants/:id/balance", resellerHandler.GetTenantBalance)
			resellerGroup.POST("/tenants/:id/transfer", resellerHandler.TransferToTenant)
			resellerGroup.POST("/tenants/:id/subscribe", resellerHandler.SubscribeTenant)
			resellerGroup.POST("/tenants/:id/api-key", resellerHandler.UpdateAPIKey)
			resellerGroup.GET("/staff", resellerHandler.GetStaff)

			// Packages & Subscriptions (reseller's own tenant scope) - Requirement 3.3
			resellerGroup.GET("/packages", packageHandler.ListPackages)
			resellerGroup.GET("/packages/:id", packageHandler.GetPackage)
			resellerGroup.POST("/subscriptions", subscriptionHandler.Subscribe)
			resellerGroup.GET("/subscriptions/active", subscriptionHandler.GetSubscription)
			resellerGroup.GET("/subscriptions/usage", subscriptionHandler.GetUsage)
			resellerGroup.PUT("/subscriptions/:id/cancel", subscriptionHandler.CancelSubscription)
			resellerGroup.PUT("/subscriptions/:id/enable-auto-renew", subscriptionHandler.EnableAutoRenew)
		}

		// Agent Routes (Reseller Companies - مستقل عن reseller الحالي)
		agent.RegisterRoutes(r, db.GormDB)
	}

	port := os.Getenv("PORT")
	if port == "" {
		// 8085 matches the Android SMS gateway app defaults and the plan docs
		port = "8085"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Info("Server listening", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen: ", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	dispatcherManager.Stop()
	msgOrchestrator.Stop()

	// Stop alert manager gracefully (Bug 24)
	if systemAlertManager != nil {
		systemAlertManager.Stop()
		log.Info("Alert manager stopped")
	}

	// Stop channel health check gracefully (Bug 13 - Task 15.3)
	if channelHandler != nil {
		channelHandler.GetService().StopHealthCheck()
		log.Info("Channel health check stopped")
	}

	// Close database connections
	db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", zap.Error(err))
	}

	log.Info("Server exiting")
}

// startAutoRenewalChecker runs the auto-renewal check on startup and then on a
// 6-hour ticker. CheckExpiredSubscriptions is safe to re-run: renewed
// subscriptions move to next period and are no longer matched.
func startAutoRenewalChecker(ctx context.Context, svc *billing.AutoRenewalService, logger *zap.Logger) {
	logger.Info("Auto-renewal checker started")
	runAutoRenewalCheck(ctx, svc, logger)

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Auto-renewal checker stopped")
			return
		case <-ticker.C:
			runAutoRenewalCheck(ctx, svc, logger)
		}
	}
}

// runAutoRenewalCheck performs a single pass over expired subscriptions.
func runAutoRenewalCheck(ctx context.Context, svc *billing.AutoRenewalService, logger *zap.Logger) {
	if err := svc.CheckExpiredSubscriptions(ctx); err != nil {
		logger.Error("Auto-renewal check failed", zap.Error(err))
	}
}

// monitorLowBalance monitors tenant balances and sends alerts when low
func monitorLowBalance(ctx context.Context, balanceManager balance.Manager, alertManager *alerting.Manager, logger *zap.Logger) {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	logger.Info("Low balance monitoring started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Low balance monitoring stopped")
			return
		case <-ticker.C:
			checkLowBalances(ctx, balanceManager, alertManager, logger)
		}
	}
}

// checkLowBalances checks all tenant balances and sends alerts for low balances
func checkLowBalances(ctx context.Context, balanceManager balance.Manager, alertManager *alerting.Manager, logger *zap.Logger) {
	// Query all tenant balances
	query := `
		SELECT id AS tenant_id, balance 
		FROM tenants 
		WHERE balance < 100.00 AND balance > 0 AND deleted_at IS NULL
	`

	rows, err := db.PgxPool.Query(ctx, query)
	if err != nil {
		logger.Error("Failed to query tenant balances", zap.Error(err))
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tenantID uuid.UUID
		var balance decimal.Decimal

		if err := rows.Scan(&tenantID, &balance); err != nil {
			logger.Error("Failed to scan balance row", zap.Error(err))
			continue
		}

		// Determine alert level based on balance
		var level alerting.AlertLevel
		var title string

		if balance.LessThan(decimal.NewFromFloat(10.0)) {
			level = alerting.AlertLevelCritical
			title = "Critical: Very Low Balance"
		} else if balance.LessThan(decimal.NewFromFloat(50.0)) {
			level = alerting.AlertLevelWarning
			title = "Warning: Low Balance"
		} else {
			level = alerting.AlertLevelInfo
			title = "Info: Balance Below Threshold"
		}

		alert := alerting.NewAlert(
			alerting.AlertTypeLowBalance,
			level,
			title,
			fmt.Sprintf("Tenant %s has low balance: %s SAR", tenantID.String(), balance.String()),
			"balance_monitor",
		).WithTenantID(tenantID).
			WithMetadata("balance", balance.String()).
			WithMetadata("threshold", "100.00")

		alertManager.SendAlertAsync(alert)

		logger.Info("Low balance alert sent",
			zap.String("tenant_id", tenantID.String()),
			zap.String("balance", balance.String()),
			zap.String("level", string(level)),
		)
	}
}
