package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/leco-dev26/sentinelflow/internal/cache"
	"github.com/leco-dev26/sentinelflow/internal/config"
	"github.com/leco-dev26/sentinelflow/internal/logger"
)

// Engine es el motor de reglas que maneja el enrutamiento y failover
type Engine struct {
	config *config.Config
	cache  *cache.Cache
	client *http.Client
}

// NewEngine crea una nueva instancia del motor
func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		config: cfg,
		cache:  cache.NewCache(cfg.Cache.TTL),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Route procesa una petición y la enruta al proveedor adecuado
func (e *Engine) Route(path, method string, body []byte, headers http.Header) ([]byte, int, error) {
	logger.Infof("📨 Ruta: %s %s", method, path)

	// Buscar regla para esta ruta
	rule := e.config.GetRuleByPath(path, method)
	if rule == nil {
		return nil, http.StatusNotFound, fmt.Errorf("no rule found for %s %s", method, path)
	}

	// Verificar caché
	if e.config.Cache.Enabled && rule.Cache {
		cacheKey := fmt.Sprintf("%s:%s:%s", method, path, string(body))
		if cached, found := e.cache.Get(cacheKey); found {
			logger.Infof("✅ Respuesta desde caché: %s", cacheKey)
			return cached.([]byte), http.StatusOK, nil
		}
	}

	// Intentar con cada proveedor en orden
	var lastErr error
	for _, providerName := range rule.Providers {
		provider := e.config.GetProviderByName(providerName)
		if provider == nil {
			logger.Warnf("⚠️  Proveedor %s no encontrado", providerName)
			continue
		}

		logger.Infof("🔄 Intentando con proveedor: %s", provider.Name)

		resp, status, err := e.forwardRequest(provider, path, method, body, headers)
		if err == nil && status < 500 {
			// Éxito! Guardar en caché si aplica
			if e.config.Cache.Enabled && rule.Cache {
				cacheKey := fmt.Sprintf("%s:%s:%s", method, path, string(body))
				e.cache.Set(cacheKey, resp)
				logger.Infof("💾 Guardado en caché: %s", cacheKey)
			}
			return resp, status, nil
		}

		lastErr = err
		logger.Warnf("❌ Falló proveedor %s: %v", provider.Name, err)

		// Si tiene fallback, continuar
		if provider.Fallback != "" {
			logger.Infof("↩️  Fallback a: %s", provider.Fallback)
			continue
		}
	}

	return nil, http.StatusServiceUnavailable, fmt.Errorf("all providers failed: %v", lastErr)
}

// forwardRequest envía la petición a un proveedor específico
func (e *Engine) forwardRequest(provider *config.Provider, path, method string, body []byte, headers http.Header) ([]byte, int, error) {
	// Construir URL completa
	fullURL := provider.URL + path

	// Crear petición
	req, err := http.NewRequest(method, fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}

	// Añadir headers del proveedor
	for key, value := range provider.Headers {
		req.Header.Set(key, value)
	}

	// Añadir headers originales (excepto los de autenticación)
	for key, values := range headers {
		if key != "Authorization" && key != "X-API-Key" {
			for _, v := range values {
				req.Header.Set(key, v)
			}
		}
	}

	// Ejecutar petición
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	// Leer respuesta
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	// Verificar si la respuesta es exitosa
	if resp.StatusCode >= 500 {
		return respBody, resp.StatusCode, fmt.Errorf("provider returned %d", resp.StatusCode)
	}

	return respBody, resp.StatusCode, nil
}

// GetCacheStats retorna estadísticas de la caché
func (e *Engine) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"size":      e.cache.Size(),
		"enabled":   e.config.Cache.Enabled,
		"ttl":       e.config.Cache.TTL.String(),
		"max_size":  e.config.Cache.MaxSize,
	}
}
