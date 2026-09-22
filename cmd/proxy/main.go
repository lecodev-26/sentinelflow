package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/lecodev-26/sentinelflow/internal/accounting"
	"github.com/lecodev-26/sentinelflow/internal/audit"
	"github.com/lecodev-26/sentinelflow/internal/controlplane"
	"github.com/lecodev-26/sentinelflow/internal/enterprise/hierarchy"
	"github.com/lecodev-26/sentinelflow/internal/enterprise/regions"
	"github.com/lecodev-26/sentinelflow/internal/enterprise/scim"
	"github.com/lecodev-26/sentinelflow/internal/enterprise/sso"
	"github.com/lecodev-26/sentinelflow/internal/enterprise/vault"
	"github.com/lecodev-26/sentinelflow/internal/gateway"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/metrics"
	"github.com/lecodev-26/sentinelflow/internal/observability"
	"github.com/lecodev-26/sentinelflow/internal/observability/analytics"
	"github.com/lecodev-26/sentinelflow/internal/observability/traces"
	"github.com/lecodev-26/sentinelflow/internal/proxy"
	"github.com/lecodev-26/sentinelflow/internal/ratelimit"
	"github.com/lecodev-26/sentinelflow/internal/rbac"
	"github.com/lecodev-26/sentinelflow/internal/storage/sqlite"
	"github.com/redis/go-redis/v9"
)

const Version = "2.9.0"

func main() {
	port := flag.String("port", "8080", "Puerto del proxy")
	adminPort := flag.String("admin-port", "8081", "Puerto del control plane")
	metricsPort := flag.String("metrics-port", "9090", "Puerto para métricas")
	configFile := flag.String("config", "configs/rules.yaml", "Archivo de configuración")
	dbPath := flag.String("db", "sentinelflow.db", "Ruta de SQLite")
	flag.Parse()

	log.Printf("🛡️ SentinelFlow v%s", Version)
	log.Printf("   Gateway:       :%s", *port)
	log.Printf("   Control Plane: :%s", *adminPort)
	log.Printf("   Métricas:      :%s", *metricsPort)
	log.Printf("   Database:      %s", *dbPath)

	if err := observability.InitTracing("sentinelflow", ""); err != nil {
		log.Printf("⚠️ Tracing no disponible: %v", err)
	}

	// === STORAGE ===
	store, err := sqlite.NewStore(*dbPath)
	if err != nil {
		log.Fatalf("❌ Error abriendo base de datos: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(context.Background()); err != nil {
		log.Fatalf("❌ Error aplicando migraciones: %v", err)
	}
	log.Printf("✅ Base de datos lista: %s", *dbPath)

	// === ENTERPRISE V2.9 ===
	ssoManager := sso.NewManager()
	logger.Info("🔐 SSO Manager iniciado")

	scimHandler := scim.NewHandler()
	logger.Info("👥 SCIM handler iniciado")

	hierarchyTree := hierarchy.NewTree()
	// Crear root org
	hierarchyTree.CreateNode("", "root", "Root", "org")
	logger.Info("🌳 Organization hierarchy iniciada")

	vaultKey := os.Getenv("SENTINELFLOW_VAULT_KEY")
	credentialVault, err := vault.NewVault(vaultKey)
	if err != nil {
		log.Fatalf("❌ Error creando vault: %v", err)
	}
	if vaultKey == "" {
		logger.Warnf("⚠️ Vault iniciado con key aleatoria (NO persistente). Configura SENTINELFLOW_VAULT_KEY")
	} else {
		logger.Info("🔒 Credential vault iniciado (AES-256-GCM)")
	}

	regionResolver := regions.NewResolver()
	_ = regionResolver
	logger.Info("🌍 Regional routing iniciado")

	// === OBSERVABILITY V2.7 ===
	traceStore := traces.NewStore(10000)
	providerAnalytics := analytics.NewProviderAnalytics()
	routingAnalytics := analytics.NewRoutingAnalytics(10000)
	costAnalytics := analytics.NewCostAnalytics()
	logger.Info("🔍 Observability avanzada iniciada")

	p, err := proxy.NewProxy(*configFile)
	if err != nil {
		log.Fatalf("❌ Error creando proxy: %v", err)
	}

	// RBAC
	orgMgr := rbac.NewOrganizationManager()
	userMgr := rbac.NewUserManager(orgMgr)

	defaultOrg := orgMgr.CreateOrganization("default", "Default Organization")
	defaultUser, err := userMgr.CreateUser("admin@local", "Admin", rbac.RoleAdmin, defaultOrg.ID)
	if err != nil {
		log.Fatalf("❌ Error creando usuario: %v", err)
	}

	rawKey, _, err := userMgr.CreateAPIKey(defaultUser.ID, "default-key", "", nil, 0)
	if err != nil {
		log.Fatalf("❌ Error creando API key: %v", err)
	}
	log.Printf("🔑 API key demo: %s", rawKey)

	// Redis (opcional)
	var redisClient *redis.Client
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			redisClient = redis.NewClient(opt)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if err := redisClient.Ping(ctx).Err(); err != nil {
				log.Printf("⚠️ Redis no disponible: %v", err)
				redisClient = nil
			} else {
				log.Printf("✅ Redis conectado: %s", redisURL)
			}
			cancel()
		}
	}

	var distLimiter *ratelimit.DistributedLimiterV2
	if redisClient != nil {
		distLimiter = ratelimit.NewDistributedLimiterV2(redisClient, "sf:ratelimit")
	}

	// Accounting
	pricingEngine := accounting.NewPricingEngine(p.GetModelRegistry())
	budgetEngine := accounting.NewBudgetEngine(func(alert accounting.BudgetAlert) {
		logger.Warnf("🚨 BUDGET ALERT: tenant=%s threshold=%d%% spent=$%.2f/%.2f forecast=$%.2f",
			alert.TenantID, alert.Threshold, alert.Spent, alert.Limit, alert.Forecast)
	})
	budgetEngine.SetBudget("default", 100.0)
	accountingSvc := accounting.NewService(pricingEngine, budgetEngine)
	logger.Info("💰 Accounting service iniciado")

	// Audit
	audit.SubscribeAll(&audit.LoggerWriter{})
	logger.Info("📝 Audit system iniciado")

	// Middlewares
	limiter := ratelimit.NewLimiter(100, time.Minute)
	limits := gateway.NewLimitMiddleware(gateway.DefaultLimits())

	auth, err := gateway.NewAuthMiddlewareFromEnv(userMgr)
	if err != nil {
		log.Fatalf("❌ Error configurando auth: %v", err)
	}

	security := gateway.NewSecurityMiddleware(false)
	cacheMw := gateway.NewCacheMiddleware(true, 5*time.Minute)
	quotaMw := gateway.NewQuotaMiddleware(true)
	quotaMw.SetQuota("default", gateway.DefaultQuota())
	obsMw := gateway.NewObservabilityMiddleware(false)
	dedupMw := gateway.NewDedupMiddleware(true)
	bulkheadMw := gateway.NewBulkheadMiddleware(200, 5*time.Second)
	accountingMw := gateway.NewAccountingMiddleware(accountingSvc, true)

	// Pipeline completo V2.9
	pipeline := gateway.NewPipeline().
		Use(gateway.ContextMiddleware()).
		Use(obsMw.Handler).
		Use(metrics.MetricsMiddleware).
		Use(limits.Handler).
		Use(bulkheadMw.Handler).
		Use(dedupMw.Handler).
		Use(auth.Handler).
		Use(security.Handler).
		Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !limiter.Allow(r.RemoteAddr) {
					gateway.WriteError(w, gateway.NewRateLimitedError("too many requests"))
					return
				}
				if distLimiter != nil {
					allowed, _, err := distLimiter.Allow(r.Context(), r.RemoteAddr, 100, time.Minute)
					if err != nil || !allowed {
						gateway.WriteError(w, gateway.NewRateLimitedError("too many requests"))
						return
					}
				}
				next.ServeHTTP(w, r)
			})
		}).
		Use(quotaMw.Handler).
		Use(cacheMw.Handler).
		Use(accountingMw.Handler)

	mainHandler := pipeline.Then(p.Handler())

	// === GATEWAY ROUTER ===
	gatewayRouter := mux.NewRouter()
	gatewayRouter.HandleFunc("/health", p.HealthCheck)
	gatewayRouter.PathPrefix("/dashboard").Handler(
		http.StripPrefix("/dashboard", http.FileServer(http.Dir("./web/dashboard"))),
	)
	gatewayRouter.PathPrefix("/demo").Handler(
		http.StripPrefix("/demo", http.FileServer(http.Dir("./web/demo"))),
	)
	gatewayRouter.PathPrefix("/").Handler(mainHandler)

	// === CONTROL PLANE ROUTER ===
	adminRouter := mux.NewRouter()
	adminRouter.Use(corsMiddleware)

	orgHandler := controlplane.NewOrganizationHandler(orgMgr)
	userHandler := controlplane.NewUserHandler(userMgr)
	providerHandler := controlplane.NewProviderHandler(p)
	metricsHandler := controlplane.NewMetricsHandler(p, traceStore, providerAnalytics, routingAnalytics, costAnalytics)

	cpRouter := controlplane.NewRouter(orgHandler, userHandler, providerHandler, metricsHandler)
	cpRouter.Register(adminRouter)

	// Servidores
	gatewaySrv := &http.Server{
		Addr:         ":" + *port,
		Handler:      gatewayRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	adminSrv := &http.Server{
		Addr:         ":" + *adminPort,
		Handler:      adminRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	metricsSrv := &http.Server{
		Addr:    ":" + *metricsPort,
		Handler: metrics.Handler(),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("✅ Gateway en http://localhost:%s", *port)
		log.Printf("   Pipeline V2.9: context → obs → metrics → limits → bulkhead → dedup → auth → security → ratelimit → quota → cache → accounting → engine")
		if err := gatewaySrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Gateway error: %v", err)
		}
	}()

	go func() {
		log.Printf("✅ Control Plane en http://localhost:%s", *adminPort)
		log.Printf("   Enterprise V2.9:")
		log.Printf("   SSO providers registrados: %d", len(ssoManager.ListProviders()))
		log.Printf("   SCIM users: %d", len(scimHandler.List()))
		log.Printf("   Hierarchy nodes: %d", hierarchyTree.Size())
		log.Printf("   Vault entries: %d", len(credentialVault.List()))
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Admin error: %v", err)
		}
	}()

	go func() {
		log.Printf("✅ Métricas en http://localhost:%s/metrics", *metricsPort)
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("⚠️ Error en métricas: %v", err)
		}
	}()

	<-stop
	log.Println("🔄 Apagando...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gatewaySrv.Shutdown(ctx)
	adminSrv.Shutdown(ctx)
	metricsSrv.Shutdown(ctx)
	p.Stop()
	if redisClient != nil {
		redisClient.Close()
	}

	log.Println("✅ SentinelFlow detenido correctamente")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
