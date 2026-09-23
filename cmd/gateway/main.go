package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	accountingv3 "github.com/lecodev-26/sentinelflow/internal/accounting/v3"
	"github.com/lecodev-26/sentinelflow/internal/events"
	"github.com/lecodev-26/sentinelflow/internal/gateway/v3/executor"
	"github.com/lecodev-26/sentinelflow/internal/gateway/v3/middleware"
	"github.com/lecodev-26/sentinelflow/internal/gateway/v3/normalizer"
	"github.com/lecodev-26/sentinelflow/internal/identity"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	observabilityv3 "github.com/lecodev-26/sentinelflow/internal/observability/v3"
	policyv3 "github.com/lecodev-26/sentinelflow/internal/policy/v3"
	providers "github.com/lecodev-26/sentinelflow/internal/providers/v3"
	"github.com/lecodev-26/sentinelflow/internal/providers/v3/adapters"
	"github.com/lecodev-26/sentinelflow/internal/rbac"
	routing "github.com/lecodev-26/sentinelflow/internal/routing/v3"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
	"github.com/lecodev-26/sentinelflow/internal/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log.Printf("🛡️ SentinelFlow Gateway v%s", version.Full())

	logger.Init(&struct {
		Level  string
		Format string
		Output string
	}{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})

	dbURL := os.Getenv("SENTINELFLOW_DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("❌ SENTINELFLOW_DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pgClient, err := postgres.New(ctx, postgres.DefaultConfig(dbURL))
	if err != nil {
		log.Fatalf("❌ Error conectando a PostgreSQL: %v", err)
	}
	defer pgClient.Close()
	log.Printf("✅ PostgreSQL conectado")

	if err := pgClient.Migrate(ctx); err != nil {
		log.Printf("⚠️ Error aplicando migraciones: %v", err)
	}

	// === Provider Registry ===
	registry := providers.NewRegistry()

	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		_ = registry.Register(adapters.NewOpenAIAdapter(apiKey, ""))
		log.Printf("✅ Provider registrado: openai")
	}
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		_ = registry.Register(adapters.NewAnthropicAdapter(apiKey, ""))
		log.Printf("✅ Provider registrado: anthropic")
	}
	_ = registry.Register(adapters.NewOllamaAdapter(""))
	log.Printf("✅ Provider registrado: ollama")

	manager := providers.NewManager(registry)
	manager.Start(context.Background())
	defer manager.Stop()

	// === Model Registry + Routing Engine ===
	modelRegistry := routing.NewModelRegistry()
	routingEngine := routing.NewEngine(modelRegistry, manager, routing.DefaultWeights())

	// === Executor ===
	exec := executor.NewExecutor(registry, manager)
	log.Printf("🔄 Executor iniciado (failover real)")

	log.Printf("📊 Providers: %d, Models: %d", registry.Count(), len(modelRegistry.List()))

	// === Policy Engine ===
	policyEvaluator := policyv3.NewEvaluator()
	defaultPolicy := policyv3.NewDefaultPolicy("default")
	compiled, err := policyv3.NewCompiler().Compile(defaultPolicy)
	if err != nil {
		log.Fatalf("❌ Error compilando policy: %v", err)
	}
	policyEvaluator.Register(compiled)
	log.Printf("📋 Policy Engine: %d políticas", len(policyEvaluator.List()))

	// === Accounting ===
	outbox := events.NewOutbox(pgClient.Pool(), nil, events.DefaultOutboxConfig())
	accountingSvc := accountingv3.NewService(pgClient, pgClient.Usage(), pgClient.Budgets(), modelRegistry, outbox)
	accountingSvc.OnBudgetAlert(func(alert accountingv3.BudgetAlert) {
		logger.Warnf("🚨 BUDGET ALERT: tenant=%s threshold=%d%% spent=$%.2f/%.2f",
			alert.TenantID, alert.Threshold, alert.Spent, alert.Limit)
	})
	log.Printf("💰 Accounting service iniciado")

	// === Tracing ===
	traceStore := observabilityv3.NewStore(10000)
	log.Printf("🔍 Trace Store iniciado: 10000 capacity")

	// === Identity ===
	identitySvc := identity.NewService(pgClient)
	env := os.Getenv("SENTINELFLOW_ENV")
	if env == "" {
		log.Printf("⚠️  SENTINELFLOW_ENV not set — auth ENABLED by default (fail-safe). Set SENTINELFLOW_ENV=development to disable.")
	}
	authEnabled := env != "development"
	if authEnabled {
		log.Printf("🔒 Auth: ENABLED (env=%s)", env)
	} else {
		log.Printf("🔓 Auth: DISABLED (env=%s, dev only)", env)
	}

	// === Middlewares ===
	authMw := middleware.NewAuth(identitySvc, authEnabled)
	idempotencyMw := middleware.NewIdempotencyDistributedMiddleware(pgClient.Idempotency(), true, 24*time.Hour)
	tracingMw := middleware.NewTracingMiddleware(traceStore, true)
	policyMw := middleware.NewPolicyMiddleware(policyEvaluator, true)
	accountingMw := middleware.NewAccountingMiddleware(accountingSvc, true)
	normalizerSvc := normalizer.New()

	// === Router ===
	r := mux.NewRouter()

	// ============================================================
	// PUBLIC ENDPOINTS (no auth)
	// ============================================================
	r.HandleFunc("/livez", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	}).Methods("GET")

	r.HandleFunc("/readyz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()

		if err := pgClient.HealthCheck(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not_ready","reason":"database_unavailable"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods("GET")

	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"status":"ok","service":"sentinelflow-gateway","version":"%s","auth_enabled":%v,"providers":%d,"models":%d,"traces":%d}`,
			version.String(), authEnabled, registry.Count(), len(modelRegistry.List()), traceStore.Size())))
	}).Methods("GET")

	r.HandleFunc("/version", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		info := version.Get()
		w.Write([]byte(`{"version":"` + info.Version + `","commit":"` + info.Commit + `"}`))
	}).Methods("GET")

	// ============================================================
	// PROTECTED ENDPOINTS (auth + optional scope)
	// ============================================================

	// --- /v1/providers ---
	r.Handle("/v1/providers",
		authMw.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"providers": manager.HealthStatus(),
			})
		})),
	).Methods("GET")

	// --- /v1/models ---
	r.Handle("/v1/models",
		authMw.Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			models := modelRegistry.List()
			data := make([]map[string]interface{}, 0, len(models))
			for _, m := range models {
				data = append(data, map[string]interface{}{
					"id":       m.ID,
					"object":   "model",
					"provider": m.Provider,
					"owned_by": m.Provider,
					"context":  m.ContextSize,
					"pricing":  map[string]float64{"input": m.InputPer1M, "output": m.OutputPer1M},
				})
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"data":   data,
			})
		})),
	).Methods("GET")

	// --- /v1/traces (auth + scope traces:read + tenant isolation) ---
	r.Handle("/v1/traces",
		authMw.Handler(
			middleware.RequireScope(rbac.ScopeReadTraces)(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					// IMPORTANTE: usar tenant del contexto, NO del query param
					tenantID := middleware.GetTenantID(req.Context())

					limit := 50
					if l := req.URL.Query().Get("limit"); l != "" {
						fmt.Sscanf(l, "%d", &limit)
					}

					traces := traceStore.ListByTenant(tenantID, limit)
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"traces": traces,
						"total":  traceStore.Size(),
					})
				}),
			),
		),
	).Methods("GET")

	// --- /v1/traces/{id} (auth + tenant isolation verification) ---
	r.Handle("/v1/traces/{id}",
		authMw.Handler(
			middleware.RequireScope(rbac.ScopeReadTraces)(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					id := mux.Vars(req)["id"]
					trace, ok := traceStore.Get(id)
					if !ok {
						writeError(w, http.StatusNotFound, "not_found", "trace not found")
						return
					}

					// Verificar aislamiento de tenant
					tenantID := middleware.GetTenantID(req.Context())
					if trace.TenantID != "" && trace.TenantID != tenantID {
						logger.Warnf("🚫 Tenant isolation violation: request=%s trace_tenant=%s user_tenant=%s",
							id, trace.TenantID, tenantID)
						writeError(w, http.StatusForbidden, "forbidden", "trace belongs to another tenant")
						return
					}

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(trace)
				}),
			),
		),
	).Methods("GET")

	// --- /v1/audit (auth + scope audit:read + tenant isolation) ---
	r.Handle("/v1/audit",
		authMw.Handler(
			middleware.RequireScope(rbac.ScopeReadAudit)(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					tenantID := middleware.GetTenantID(req.Context())
					limit := 50
					if l := req.URL.Query().Get("limit"); l != "" {
						fmt.Sscanf(l, "%d", &limit)
					}
					entries, err := pgClient.Audit().ListByTenant(req.Context(), tenantID, limit)
					if err != nil {
						writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
						return
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"entries": entries,
						"total":   len(entries),
					})
				}),
			),
		),
	).Methods("GET")

	// --- /v1/analytics/daily (auth + scope analytics:read + tenant isolation) ---
	r.Handle("/v1/analytics/daily",
		authMw.Handler(
			middleware.RequireScope(rbac.ScopeReadAnalytics)(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					tenantID := middleware.GetTenantID(req.Context())
					days := 30
					if d := req.URL.Query().Get("days"); d != "" {
						fmt.Sscanf(d, "%d", &days)
					}
					rows, err := pgClient.Analytics().ListByTenant(req.Context(), tenantID, days)
					if err != nil {
						writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
						return
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"tenant_id": tenantID,
						"days":      days,
						"daily":     rows,
					})
				}),
			),
		),
	).Methods("GET")

	// --- /v1/usage/stats (auth + scope usage:read + tenant isolation) ---
	r.Handle("/v1/usage/stats",
		authMw.Handler(
			middleware.RequireScope(rbac.ScopeReadUsage)(
				http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					// IMPORTANTE: usar tenant del contexto, NO del query param
					tenantID := middleware.GetTenantID(req.Context())

					days := 30
					if daysStr := req.URL.Query().Get("days"); daysStr != "" {
						fmt.Sscanf(daysStr, "%d", &days)
					}

					stats, err := accountingSvc.Stats(req.Context(), tenantID, time.Duration(days)*24*time.Hour)
					if err != nil {
						writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
						return
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(stats)
				}),
			),
		),
	).Methods("GET")

	// --- /v1/chat/completions (auth + scope chat:write + full pipeline) ---
	chatHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "error reading body")
			return
		}
		defer req.Body.Close()

		normReq, format, err := normalizerSvc.Normalize(body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		tenantID := middleware.GetTenantID(req.Context())
		userID := middleware.GetUserID(req.Context())
		apiKeyID := middleware.GetAPIKeyID(req.Context())

		logger.Infof("📨 Request: tenant=%s user=%s format=%s model=%s stream=%v",
			tenantID, userID, format, normReq.Model, normReq.Stream)

		routeReq := &routing.Request{
			RequestID: req.Header.Get("Idempotency-Key"),
			Model:     normReq.Model,
			Residency: middleware.GetResidency(req.Context()),
		}
		if routeReq.RequestID == "" {
			routeReq.RequestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		if normReq.Stream {
			routeReq.RequiredCapabilities = []string{"stream"}
		}
		if policyResult, ok := middleware.GetPolicyResult(req.Context()); ok && policyResult.Routing != nil {
			routeReq.RequiredCapabilities = append(routeReq.RequiredCapabilities, policyResult.Routing.RequiredCapabilities...)
			routeReq.PreferredProvider = policyResult.Routing.PreferredProvider
			routeReq.ExcludedProviders = policyResult.Routing.BlockedProviders
			routeReq.MaxCost = policyResult.Routing.MaxCostPer1M
		}

		candidates, err := routingEngine.OrderByScore(routeReq)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "no_provider", err.Error())
			return
		}

		logger.Infof("🎯 Routing: %s → %d candidate(s) ordered by score",
			normReq.Model, len(candidates))

		chatReq := &providers.ChatRequest{
			Model:       normReq.Model,
			Temperature: normReq.Temperature,
			MaxTokens:   normReq.MaxTokens,
			Stream:      normReq.Stream,
		}
		for _, m := range normReq.Messages {
			chatReq.Messages = append(chatReq.Messages, providers.Message{
				Role:    m.Role,
				Content: m.Content,
			})
		}

		if normReq.Stream {
			handleStreamWithFailover(w, req, exec, candidates, chatReq, tenantID, userID, apiKeyID)
			return
		}

		resp, attempts, err := exec.ExecuteChat(req.Context(), candidates, chatReq)
		if err != nil {
			logger.Errorf("❌ All providers failed: %v", err)
			writeError(w, http.StatusBadGateway, "all_providers_failed", err.Error())
			return
		}

		var winningProvider string
		for _, a := range attempts {
			if a.Success {
				winningProvider = a.ProviderID
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Provider", winningProvider)
		w.Header().Set("X-Attempts", fmt.Sprintf("%d", len(attempts)))
		w.Header().Set("X-Tenant-Id", tenantID)
		w.Header().Set("X-User-Id", userID)
		w.Header().Set("X-Api-Key-Id", apiKeyID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	// Pipeline completo: auth → scope → idempotency → tracing → policy → accounting → chat
	r.Handle("/v1/chat/completions",
		authMw.Handler(
			middleware.RequireScope(rbac.ScopeWriteChat)(
				idempotencyMw.Handler(
					tracingMw.Handler(
						policyMw.Handler(
							accountingMw.Handler(chatHandler),
						),
					),
				),
			),
		),
	).Methods("POST")

	// === Server ===
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("✅ Gateway en http://localhost:8080")
		log.Printf("   Auth: %v", authEnabled)
		log.Printf("   Pipeline: auth → scope → idempotency → tracing → policy → accounting → chat")
		log.Printf("   Failover: real (iterates candidates by score)")
		log.Printf("   Tenant isolation: enforced on /v1/traces, /v1/traces/{id}, /v1/usage/stats")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Gateway error: %v", err)
		}
	}()

	<-stop
	log.Println("🔄 Apagando (graceful shutdown)...")

	log.Println("⏸️  Drenando tráfico (3s para LB)...")
	time.Sleep(3 * time.Second)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	log.Println("🛑 Esperando requests en vuelo (max 30s)...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ Error en shutdown: %v", err)
	}

	log.Println("✅ Gateway detenido correctamente")
}

// handleStreamWithFailover maneja streaming con failover pre-primer-chunk
func handleStreamWithFailover(
	w http.ResponseWriter,
	req *http.Request,
	exec *executor.Executor,
	candidates []routing.Candidate,
	chatReq *providers.ChatRequest,
	tenantID, userID, apiKeyID string,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming_not_supported", "streaming not supported")
		return
	}

	chunks, attempts, err := exec.ExecuteStream(req.Context(), candidates, chatReq)
	if err != nil {
		logger.Errorf("❌ All stream providers failed: %v", err)
		writeError(w, http.StatusBadGateway, "all_providers_failed", err.Error())
		return
	}

	var winningProvider string
	for _, a := range attempts {
		if a.Success {
			winningProvider = a.ProviderID
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Provider", winningProvider)
	w.Header().Set("X-Attempts", fmt.Sprintf("%d", len(attempts)))
	w.Header().Set("X-Tenant-Id", tenantID)
	w.Header().Set("X-User-Id", userID)
	w.Header().Set("X-Api-Key-Id", apiKeyID)
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	start := time.Now()
	totalChunks := 0
	var ttft time.Duration

	for chunk := range chunks {
		select {
		case <-req.Context().Done():
			logger.Warnf("⚠️ Client disconnected after %d chunks", totalChunks)
			return
		default:
		}

		if totalChunks == 0 {
			ttft = time.Since(start)
			logger.Infof("⚡ TTFT: %dms", ttft.Milliseconds())
		}

		data, err := json.Marshal(map[string]interface{}{
			"id":      "chatcmpl-stream",
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   chatReq.Model,
			"choices": []map[string]interface{}{
				{
					"index":         chunk.Index,
					"delta":         map[string]string{"content": chunk.Delta},
					"finish_reason": chunk.FinishReason,
				},
			},
		})
		if err != nil {
			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", string(data))
		flusher.Flush()
		totalChunks++

		if chunk.FinishReason != "" {
			break
		}
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	logger.Infof("✅ Stream completed: %d chunks, TTFT=%dms, total=%dms (provider=%s)",
		totalChunks, ttft.Milliseconds(), time.Since(start).Milliseconds(), winningProvider)
}

func writeError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{"type": errType, "message": message},
	})
}
