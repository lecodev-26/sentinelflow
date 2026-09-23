package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	cpv3 "github.com/lecodev-26/sentinelflow/internal/controlplane/v3"
	oidcv3 "github.com/lecodev-26/sentinelflow/internal/enterprise/v3/oidc"
	scimv3 "github.com/lecodev-26/sentinelflow/internal/enterprise/v3/scim"
	"github.com/lecodev-26/sentinelflow/internal/gateway/v3/middleware"
	"github.com/lecodev-26/sentinelflow/internal/identity"
	"github.com/lecodev-26/sentinelflow/internal/logger"
	"github.com/lecodev-26/sentinelflow/internal/storage/postgres"
	"github.com/lecodev-26/sentinelflow/internal/version"
)

func main() {
	log.Printf("🛡️ SentinelFlow Control Plane v%s", version.Full())

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
	logger.Infof("📊 Pool stats: %v", pgClient.Stats())

	if err := pgClient.Migrate(ctx); err != nil {
		log.Fatalf("❌ Error aplicando migraciones: %v", err)
	}
	log.Printf("✅ Migraciones aplicadas")

	identitySvc := identity.NewService(pgClient)
	log.Printf("✅ Identity service iniciado")

	// === Auth middleware ===
	authEnabled := os.Getenv("SENTINELFLOW_ENV") == "production"
	authMw := middleware.NewAuth(identitySvc, authEnabled)
	if authEnabled {
		log.Printf("🔒 Control Plane auth: ENABLED (production)")
	} else {
		log.Printf("🔓 Control Plane auth: DISABLED (development)")
	}

	// === OIDC setup ===
	oidcMgr := oidcv3.NewManager()
	oidcFlow := oidcv3.NewFlow(oidcMgr)
	jwksCache := oidcv3.NewJWKSCache()
	oidcValidator := oidcv3.NewValidator(jwksCache)

	if cid := os.Getenv("GOOGLE_CLIENT_ID"); cid != "" {
		_ = oidcMgr.RegisterProvider(&oidcv3.Config{
			Provider:     oidcv3.ProviderGoogle,
			ClientID:     cid,
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
			Issuer:       "https://accounts.google.com",
		})
		log.Printf("✅ OIDC provider registrado: google")
	}
	if cid := os.Getenv("GITHUB_CLIENT_ID"); cid != "" {
		_ = oidcMgr.RegisterProvider(&oidcv3.Config{
			Provider:     oidcv3.ProviderGitHub,
			ClientID:     cid,
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
		})
		log.Printf("✅ OIDC provider registrado: github")
	}

	// === SCIM handler ===
	scimHandler := scimv3.NewHandler(identitySvc)
	scimAuth := middleware.NewSCIMAuthMiddleware()
	if scimAuth.Enabled() {
		log.Printf("🔒 SCIM auth: ENABLED (SCIM_TOKEN set)")
	} else {
		log.Printf("⚠️  SCIM auth: DISABLED (SCIM_TOKEN not set)")
	}

	// === Router ===
	r := mux.NewRouter()
	r.Use(corsMiddleware)

	// ============================================================
	// PUBLIC ENDPOINTS
	// ============================================================
	r.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		if err := pgClient.HealthCheck(req.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"sentinelflow-controlplane","version":"` + version.String() + `"}`))
	}).Methods("GET")

	r.HandleFunc("/version", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		info := version.Get()
		w.Write([]byte(`{"version":"` + info.Version + `","commit":"` + info.Commit + `"}`))
	}).Methods("GET")

	// === OIDC: providers + login + callback (público) ===
	r.HandleFunc("/auth/oidc/providers", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"providers": oidcMgr.ListProviders(),
		})
	}).Methods("GET")

	r.HandleFunc("/auth/oidc/{provider}/login", func(w http.ResponseWriter, req *http.Request) {
		provider := oidcv3.Provider(mux.Vars(req)["provider"])
		redirect := req.URL.Query().Get("redirect")

		authURL, err := oidcFlow.BuildAuthURL(provider, redirect)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		http.Redirect(w, req, authURL, http.StatusTemporaryRedirect)
	}).Methods("GET")

	// === Callback REAL (E.6) ===
	r.HandleFunc("/auth/oidc/{provider}/callback", func(w http.ResponseWriter, req *http.Request) {
		providerName := mux.Vars(req)["provider"]
		provider := oidcv3.Provider(providerName)

		// 1. Extraer code + state
		code := req.URL.Query().Get("code")
		state := req.URL.Query().Get("state")

		if code == "" || state == "" {
			writeOIDCError(w, http.StatusBadRequest, "missing_code_or_state",
				"missing code or state in callback")
			return
		}

		// 2. Validar state → obtener redirect + nonce
		redirectAfter, nonce, ok := oidcFlow.ValidateState(state)
		if !ok {
			writeOIDCError(w, http.StatusBadRequest, "invalid_state",
				"invalid or expired state parameter")
			return
		}

		// 3. Intercambiar code por tokens
		tokens, err := oidcFlow.ExchangeCode(req.Context(), provider, code)
		if err != nil {
			logger.Errorf("❌ OIDC token exchange failed: %v", err)
			writeOIDCError(w, http.StatusBadGateway, "token_exchange_failed", err.Error())
			return
		}

		// 4. Validar ID token (firma + claims)
		claims, err := oidcValidator.Validate(req.Context(), oidcMgr, provider, tokens.IDToken, nonce)
		if err != nil {
			logger.Errorf("❌ OIDC ID token validation failed: %v", err)
			writeOIDCError(w, http.StatusUnauthorized, "invalid_id_token", err.Error())
			return
		}

		// 5. Buscar o crear usuario en la DB
		user, err := identitySvc.GetUserByEmail(req.Context(), claims.Email)
		if err != nil {
			// Usuario no existe → crearlo
			// Necesitamos una org. Buscamos la primera por defecto.
			orgs, orgErr := identitySvc.ListOrganizations(req.Context())
			if orgErr != nil || len(orgs) == 0 {
				writeOIDCError(w, http.StatusInternalServerError, "no_organization",
					"no organization available to assign new user")
				return
			}
			newOrgID := orgs[0].ID

			user, err = identitySvc.CreateUser(req.Context(), newOrgID, claims.Email, claims.Name, "viewer")
			if err != nil {
				logger.Errorf("❌ Failed to create user from OIDC: %v", err)
				writeOIDCError(w, http.StatusInternalServerError, "user_creation_failed", err.Error())
				return
			}
			logger.Infof("👤 User created via OIDC: %s (%s)", user.Email, user.ID)
		}

		// 6. Crear sesión
		session := oidcFlow.CreateSession(
			user.ID,
			user.Email,
			user.Name,
			provider,
			tokens.AccessToken,
			tokens.RefreshToken,
			24*time.Hour,
		)

		// 7. Set cookie + redirect
		http.SetCookie(w, &http.Cookie{
			Name:     "sf_session",
			Value:    session.ID,
			Path:     "/",
			HttpOnly: true,
			Secure:   authEnabled, // true solo en producción (HTTPS)
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int((24 * time.Hour).Seconds()),
		})

		logger.Infof("✅ OIDC login OK: user=%s provider=%s", user.Email, provider)

		// Redirigir al cliente (o devolver JSON si no hay redirect)
		if redirectAfter != "" {
			http.Redirect(w, req, redirectAfter, http.StatusFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ok",
			"session_id": session.ID,
			"user": map[string]string{
				"id":    user.ID,
				"email": user.Email,
				"name":  user.Name,
			},
			"expires_at": session.ExpiresAt,
		})
	}).Methods("GET")

	// === SCIM (bearer token propio) ===
	scimRouter := r.PathPrefix("/scim/v2").Subrouter()
	if scimAuth.Enabled() {
		scimRouter.Use(scimAuth.Handler)
	}
	scimHandler.Register(scimRouter)

	// ============================================================
	// CONTROL PLANE V3 (auth + scopes)
	// ============================================================
	handlers := cpv3.NewWithAuth(identitySvc, authMw)
	handlers.Register(r)

	// === Server ===
	srv := &http.Server{
		Addr:         ":8081",
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("✅ Control Plane en http://localhost:8081")
		log.Printf("   Auth: %v", authEnabled)
		log.Printf("   OIDC providers: %d", len(oidcMgr.ListProviders()))
		log.Printf("   OIDC callback: real (token exchange + JWKS validation)")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Control Plane error: %v", err)
		}
	}()

	<-stop
	log.Println("🔄 Apagando...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	srv.Shutdown(shutdownCtx)
	log.Println("✅ Control Plane detenido")
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

func writeOIDCError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"type":    errType,
			"message": message,
		},
	})
}

// ensure fmt is used (para futuras mejoras)
var _ = fmt.Sprintf
