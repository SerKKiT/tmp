package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"my-go-app/internal/application/services"
	"my-go-app/internal/domain/stream"
	"my-go-app/pkg/config" // ✅ ДОБАВЛЕН ИМПОРТ
	"my-go-app/pkg/middleware"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// StreamHandler обрабатывает все HTTP запросы связанные с управлением потоками.
// Реализует полный CRUD (Create, Read, Update, Delete) функционал для стримов.
// Также обеспечивает интеграцию с внешним streaming service через HTTP вызовы.
type StreamHandler struct {
	streamService *services.StreamService
}

// StreamingResponse - стандартизированный формат ответа для операций со streaming service.
// Используется для унификации коммуникации между Go API и внешним streaming сервисом.
type StreamingResponse struct {
	Message  string      `json:"message"`
	StreamID string      `json:"stream_id,omitempty"`
	Status   string      `json:"status,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// NewStreamHandler создает новый экземпляр обработчика потоков.
// Принимает сервис для работы с бизнес-логикой и возвращает готовый к использованию handler.
//
// Параметры:
//
//	streamService - сервис содержащий бизнес-логику для работы с потоками
//
// Возвращает:
//
//	*StreamHandler - новый экземпляр обработчика
func NewStreamHandler(streamService *services.StreamService) *StreamHandler {
	return &StreamHandler{
		streamService: streamService,
	}
}

// HandleStreams обрабатывает HTTP запросы к endpoint /api/tasks
//
// Поддерживаемые методы:
//
//	GET  - получение списка всех потоков с опциональной фильтрацией по статусу
//	POST - создание нового потока на основе данных из request body
//
// GET параметры:
//
//	stream_status (query) - фильтр по статусу потоков (starting, running, stopped, error)
//
// POST body (JSON):
//
//	{"name": "stream_name", "stream_id": "unique_id"}
//
// Возвращает:
//
//	GET:  200 + список потоков в JSON
//	POST: 201 + данные созданного потока
//	Ошибки: 400/500 + описание ошибки
func (h *StreamHandler) HandleStreams(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	switch r.Method {
	case "GET":
		statusFilter := r.URL.Query().Get("stream_status")

		streams, err := h.streamService.ListStreams(ctx, statusFilter)
		if err != nil {
			response := middleware.Response{
				Message: "Failed to get streams",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := middleware.Response{
			Message: "Streams retrieved successfully",
			Data:    streams,
		}
		json.NewEncoder(w).Encode(response)

	case "POST":
		var req services.CreateStreamRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response := middleware.Response{
				Message: "Invalid request format",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		stream, err := h.streamService.CreateStream(ctx, &req)
		if err != nil {
			response := middleware.Response{
				Message: "Failed to create stream",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := middleware.Response{
			Message: "Stream created successfully",
			Data:    stream,
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *StreamHandler) HandleStreamByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	idStr := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response := middleware.Response{
			Message: "Invalid ID",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	switch r.Method {
	case "GET":
		stream, err := h.streamService.GetStreamByID(ctx, uint(id))
		if err != nil {
			response := middleware.Response{
				Message: "Stream not found",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := middleware.Response{
			Message: "Stream found",
			Data:    stream,
		}
		json.NewEncoder(w).Encode(response)

	case "DELETE":
		err := h.streamService.DeleteStream(ctx, uint(id))
		if err != nil {
			response := middleware.Response{
				Message: "Failed to delete stream",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		response := middleware.Response{
			Message: "Stream deleted successfully",
		}
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *StreamHandler) HandleStreamControl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	streamID := strings.TrimPrefix(r.URL.Path, "/api/streams/")
	if streamID == "" {
		response := middleware.Response{
			Message: "StreamID is required",
			Error:   "StreamID is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	streamEntity, err := h.streamService.GetStreamByStreamID(ctx, streamID)
	if err != nil {
		response := middleware.Response{
			Message: "Stream not found",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	switch r.Method {
	case "POST":
		var req struct {
			Action string `json:"action"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response := middleware.Response{
				Message: "Invalid request format",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}

		log.Printf("🎬 Stream action requested: %s for stream %s", req.Action, streamID)

		// Отправляем запрос в streaming service
		streamingResp, err := h.callStreamingService(streamID, req.Action)
		if err != nil {
			log.Printf("❌ Failed to communicate with streaming service: %v", err)
			response := middleware.Response{
				Message: "Failed to communicate with streaming service",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		// ✅ ИСПРАВЛЕНА ЛОГИКА: правильное определение статуса
		var newStatus stream.Status

		if req.Action == "start" {
			if streamingResp.Error == "" {
				// Успешный запуск
				newStatus = stream.StatusStarting
				log.Printf("✅ Stream %s starting successfully", streamID)
			} else {
				// Ошибка запуска
				newStatus = stream.StatusError
				log.Printf("❌ Stream %s failed to start: %s", streamID, streamingResp.Error)
			}
		} else if req.Action == "stop" {
			newStatus = stream.StatusStopped
			log.Printf("🛑 Stream %s stopped", streamID)
		} else {
			// Неизвестное действие
			newStatus = stream.StatusError
			log.Printf("⚠️ Unknown action %s for stream %s", req.Action, streamID)
		}

		// Обновляем статус в базе данных
		if err := h.streamService.UpdateStreamStatus(ctx, streamID, newStatus); err != nil {
			log.Printf("❌ Failed to update stream status to %s: %v", newStatus, err)
			// Не прерываем выполнение, просто логируем
		} else {
			log.Printf("✅ Stream %s status updated to %s", streamID, newStatus)
		}

		// Формируем ответ
		response := middleware.Response{
			Message: streamingResp.Message,
			Data: map[string]interface{}{
				"stream_id":      streamID,
				"stream_status":  newStatus,
				"action":         req.Action,
				"streaming_data": streamingResp.Data,
			},
		}

		// Если была ошибка в streaming service, возвращаем ошибку
		if streamingResp.Error != "" {
			response.Error = streamingResp.Error
			w.WriteHeader(http.StatusInternalServerError)
		}

		json.NewEncoder(w).Encode(response)

	case "GET":
		response := middleware.Response{
			Message: "Stream status retrieved",
			Data:    streamEntity,
		}
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *StreamHandler) callStreamingService(streamID, action string) (*StreamingResponse, error) {
	streamingServiceURL := config.GetEnv("STREAMING_SERVICE_URL", "http://streaming-service:8081") + "/api/streams"

	requestBody := map[string]string{
		"stream_id": streamID,
		"action":    action,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	log.Printf("📡 Calling streaming service: %s with %s", streamingServiceURL, string(jsonData))

	resp, err := http.Post(streamingServiceURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ HTTP request to streaming service failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Failed to read response body: %v", err)
		return nil, err
	}

	log.Printf("📡 Streaming service response: %s", string(body))

	var streamingResponse StreamingResponse
	if err := json.Unmarshal(body, &streamingResponse); err != nil {
		log.Printf("❌ Failed to parse streaming service response: %v", err)
		return nil, err
	}

	log.Printf("📡 Parsed streaming response: Message=%s, Status=%s, Error=%s",
		streamingResponse.Message, streamingResponse.Status, streamingResponse.Error)

	return &streamingResponse, nil
}

// ✅ НОВАЯ ФУНКЦИЯ: Proxy для streaming service
func (h *StreamHandler) HandleStreamingServiceProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Извлекаем путь после /api/streaming-proxy/
	proxyPath := strings.TrimPrefix(r.URL.Path, "/api/streaming-proxy")

	// Формируем URL для streaming service
	streamingServiceURL := config.GetEnv("STREAMING_SERVICE_URL", "http://streaming-service:8081")
	targetURL := streamingServiceURL + proxyPath

	log.Printf("🔄 Proxy request to: %s", targetURL)

	// Создаем запрос к streaming service
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Копируем заголовки
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Выполняем запрос
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ Proxy request failed: %v", err)
		http.Error(w, "Streaming service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Копируем статус и заголовки ответа
	w.WriteHeader(resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Копируем тело ответа
	io.Copy(w, resp.Body)

	log.Printf("✅ Proxy request completed: %d", resp.StatusCode)
}
