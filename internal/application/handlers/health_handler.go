package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"

	"my-go-app/pkg/config" // ✅ ИСПОЛЬЗУЕМ config пакет
	"my-go-app/pkg/middleware"
)

// HealthHandler предоставляет endpoint для мониторинга состояния приложения.
// Используется для health checks, мониторинга и диагностики системы.
//
// Проверяет:
//   - Состояние базы данных (подключение, пул соединений)
//   - Доступность streaming service
//   - Общее состояние приложения
type HealthHandler struct {
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	sqlDB, err := h.db.DB()
	var dbStatus string
	var dbStats interface{}

	if err != nil {
		dbStatus = "DISCONNECTED"
		dbStats = nil
	} else {
		pingCtx, pingCancel := context.WithTimeout(ctx, 2*time.Second)
		defer pingCancel()

		if err := sqlDB.PingContext(pingCtx); err != nil {
			dbStatus = "DISCONNECTED"
			dbStats = nil
		} else {
			dbStatus = "CONNECTED"
			stats := sqlDB.Stats()
			dbStats = map[string]interface{}{
				"open_connections": stats.OpenConnections,
				"in_use":           stats.InUse,
				"idle":             stats.Idle,
				"wait_count":       stats.WaitCount,
				"wait_duration":    stats.WaitDuration.String(),
			}
		}
	}

	streamingStatus := "DISCONNECTED"
	// ✅ ИСПОЛЬЗУЕМ config.GetEnv
	streamingServiceURL := config.GetEnv("STREAMING_SERVICE_URL", "http://streaming-service:8081") + "/api/health"

	httpClient := &http.Client{Timeout: 3 * time.Second}
	if resp, err := httpClient.Get(streamingServiceURL); err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			streamingStatus = "CONNECTED"
		}
	}

	response := middleware.Response{
		Message: "Server is running",
		Data: map[string]interface{}{
			"timestamp":         time.Now(),
			"status":            "OK",
			"database":          dbStatus,
			"database_pool":     dbStats,
			"streaming_service": streamingStatus,
		},
	}
	json.NewEncoder(w).Encode(response)
}
