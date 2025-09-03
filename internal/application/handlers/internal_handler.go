package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"my-go-app/internal/application/services"
	"my-go-app/internal/domain/stream"
	"my-go-app/pkg/middleware"
)

// InternalHandler обрабатывает внутренние API запросы для обновления статусов потоков.
// Предназначен для использования другими сервисами системы, а не внешними клиентами.
//
// Типичные сценарии использования:
//   - Streaming service уведомляет об изменении статуса потока
//   - Внутренние мониторинговые системы обновляют статусы
//   - Batch jobs корректируют статусы потоков
type InternalHandler struct {
	streamService *services.StreamService
}

type StatusUpdateRequest struct {
	StreamID string `json:"stream_id"`
	Status   string `json:"status"`
}

func NewInternalHandler(streamService *services.StreamService) *InternalHandler {
	return &InternalHandler{
		streamService: streamService,
	}
}

// ✅ НОВЫЙ ENDPOINT для внутренних обновлений статуса
func (h *InternalHandler) HandleStreamStatusUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := middleware.Response{
			Message: "Invalid request format",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Преобразуем строку в тип Status
	var newStatus stream.Status
	switch req.Status {
	case "starting":
		newStatus = stream.StatusStarting
	case "running":
		newStatus = stream.StatusRunning
	case "stopped":
		newStatus = stream.StatusStopped
	case "error":
		newStatus = stream.StatusError
	default:
		response := middleware.Response{
			Message: "Invalid status",
			Error:   "Unknown status: " + req.Status,
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("🔄 Internal status update: %s -> %s", req.StreamID, req.Status)

	// Обновляем статус в базе данных
	if err := h.streamService.UpdateStreamStatus(r.Context(), req.StreamID, newStatus); err != nil {
		log.Printf("❌ Failed to update stream status internally: %v", err)
		response := middleware.Response{
			Message: "Failed to update stream status",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("✅ Stream %s status updated internally to %s", req.StreamID, req.Status)

	response := middleware.Response{
		Message: "Status updated successfully",
		Data: map[string]interface{}{
			"stream_id": req.StreamID,
			"status":    req.Status,
		},
	}
	json.NewEncoder(w).Encode(response)
}
