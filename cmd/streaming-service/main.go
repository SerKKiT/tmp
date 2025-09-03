package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"

	//    "io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type StreamRequest struct {
	StreamID string `json:"stream_id"`
	Action   string `json:"action"` // start, stop, status
}

type StreamResponse struct {
	Message  string      `json:"message"`
	StreamID string      `json:"stream_id,omitempty"`
	Status   string      `json:"status,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Error    string      `json:"error,omitempty"`
}

type StreamInstance struct {
	StreamID    string     `json:"stream_id"`
	Status      string     `json:"status"` // starting, running, stopped, error
	StartTime   time.Time  `json:"start_time"`
	StreamStart *time.Time `json:"stream_start,omitempty"` // время начала потока
	Process     *exec.Cmd  `json:"-"`
	LogFile     string     `json:"log_file"`
	HLSPath     string     `json:"hls_path"`
	SRTPort     int        `json:"srt_port"`
	ServerIP    string     `json:"server_ip"`
}

type StreamManager struct {
	streams  map[string]*StreamInstance
	mutex    sync.RWMutex
	basePort int
}

// ✅ ДОБАВИТЬ после существующих структур
type StatusUpdateRequest struct {
	StreamID string `json:"stream_id"`
	Status   string `json:"status"`
}

// ✅ НОВЫЕ СТРУКТУРЫ для API ответов
type StreamInfo struct {
	ID           uint   `json:"id"`
	StreamID     string `json:"stream_id"`
	Name         string `json:"name"`
	StreamStatus string `json:"stream_status"`
}

// ✅ ДОБАВЬТЕ недостающие структуры
type StreamInstanceResponse struct {
	StreamID  string    `json:"stream_id"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	SRTPort   int       `json:"srt_port"`
	ServerIP  string    `json:"server_ip"`
	SRTURL    string    `json:"srt_url"`
	HLSURL    string    `json:"hls_url"`
}

type TasksResponse struct {
	Message string       `json:"message"`
	Data    []StreamInfo `json:"data"`
}

var manager *StreamManager

func main() {
	// Создание директорий
	if err := os.MkdirAll("/app/hls", 0o755); err != nil {
		log.Printf("Error creating hls directory: %v", err)
	}
	if err := os.MkdirAll("/app/logs", 0o755); err != nil {
		log.Printf("Error creating logs directory: %v", err)
	}

	manager = &StreamManager{
		streams:  make(map[string]*StreamInstance),
		basePort: 10000,
	}

	// ✅ НОВОЕ: Восстановление активных потоков при старте
	go func() {
		time.Sleep(5 * time.Second) // Ждем инициализации основного приложения
		restoreActiveStreams()
	}()

	// API endpoints
	http.HandleFunc("/api/streams", handleStreams)
	http.HandleFunc("/api/streams/", handleStreamByID)
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/debug/", handlePlaylistDebug)
	http.HandleFunc("/api/hls/", handleHLSMetadata) // ✅ НОВЫЙ endpoint для HLS метаданных
	// Serve HLS files
	//http.Handle("/hls/", http.StripPrefix("/hls/", http.FileServer(http.Dir("/app/hls"))))

	log.Println("Streaming service запущен на порту :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

// ИСПРАВЛЕННАЯ функция мониторинга HLS активности
func monitorHLSActivity(streamID string, hlsPath string) {
	log.Printf("📁 Запуск мониторинга HLS активности для потока %s", streamID)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastModTime := time.Time{}
	consecutiveInactiveChecks := 0
	const maxInactiveChecks = 5 // 10 секунд без активности

	for range ticker.C {
		// Проверяем, что поток еще существует
		manager.mutex.RLock()
		stream, exists := manager.streams[streamID]
		manager.mutex.RUnlock()

		if !exists {
			log.Printf("🛑 Поток %s удален, завершаем мониторинг HLS", streamID)
			return
		}

		// Получаем информацию о НОВЕЙШИХ сегментах
		entries, err := os.ReadDir(hlsPath)
		if err != nil {
			consecutiveInactiveChecks++
			continue
		}

		newestModTime := time.Time{}

		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".ts") {
				if info, err := entry.Info(); err == nil {
					if info.ModTime().After(newestModTime) {
						newestModTime = info.ModTime()
					}
				}
			}
		}

		// Проверяем, есть ли НОВАЯ активность
		hasNewActivity := false
		if !newestModTime.IsZero() && newestModTime.After(lastModTime) {
			hasNewActivity = true
			lastModTime = newestModTime
			consecutiveInactiveChecks = 0
			log.Printf("🔄 Новая HLS активность для потока %s (время: %s)", streamID, newestModTime.Format("15:04:05"))
		} else if !newestModTime.IsZero() {
			// Если есть сегменты, но нет новых - проверяем свежесть
			timeSinceLastMod := time.Since(newestModTime)
			if timeSinceLastMod > 10*time.Second {
				consecutiveInactiveChecks++
			} else {
				consecutiveInactiveChecks = 0
			}
		} else {
			consecutiveInactiveChecks++
		}

		manager.mutex.Lock()

		// Логика перехода starting -> running (только при НОВОЙ активности)
		if hasNewActivity && stream.Status == "starting" {
			now := time.Now()
			stream.StreamStart = &now
			stream.Status = "running"
			log.Printf("🎬 Новые HLS сегменты для потока %s, статус: running", streamID)
			// ✅ ДОБАВИТЬ webhook уведомление
			go notifyMainApp(streamID, "running")
		}

		// Логика перехода running -> starting (только после продолжительной неактивности)
		if consecutiveInactiveChecks >= maxInactiveChecks && stream.Status == "running" {
			stream.Status = "starting"
			stream.StreamStart = nil
			log.Printf("⏹️  Длительная неактивность HLS для потока %s (%d проверок), статус: starting", streamID, consecutiveInactiveChecks)
			consecutiveInactiveChecks = 0 // Сбрасываем счетчик

			// ✅ ДОБАВИТЬ webhook уведомление
			go notifyMainApp(streamID, "starting")
		}

		manager.mutex.Unlock()
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	manager.mutex.RLock()
	totalStreams := len(manager.streams)
	runningStreams := 0
	for _, stream := range manager.streams {
		if stream.Status == "running" {
			runningStreams++
		}
	}
	manager.mutex.RUnlock()

	response := StreamResponse{
		Message: "Streaming service работает",
		Data: map[string]interface{}{
			"timestamp":       time.Now(),
			"total_streams":   totalStreams,
			"running_streams": runningStreams,
			"hls_path":        "/app/hls",
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func handleStreams(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		manager.mutex.RLock()
		streams := make([]*StreamInstance, 0, len(manager.streams))
		for _, stream := range manager.streams {
			streams = append(streams, stream)
		}
		manager.mutex.RUnlock()

		response := StreamResponse{
			Message: fmt.Sprintf("Найдено потоков: %d", len(streams)),
			Data:    streams,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding JSON: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	case "POST":
		var req StreamRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response := StreamResponse{
				Message: "Неверный формат данных",
				Error:   err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("Error encoding JSON: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			return
		}

		switch req.Action {
		case "start":
			response := startStream(req.StreamID)
			if response.Error != "" {
				w.WriteHeader(http.StatusInternalServerError)
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("Error encoding JSON: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

		case "stop":
			response := stopStream(req.StreamID)
			if response.Error != "" {
				w.WriteHeader(http.StatusInternalServerError)
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("Error encoding JSON: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

		default:
			response := StreamResponse{
				Message: "Неизвестное действие",
				Error:   "Поддерживаемые действия: start, stop",
			}
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("Error encoding JSON: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

func handleStreamByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	streamID := strings.TrimPrefix(r.URL.Path, "/api/streams/")
	if streamID == "" {
		response := StreamResponse{
			Message: "Не указан StreamID",
			Error:   "StreamID is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	manager.mutex.RLock()
	stream, exists := manager.streams[streamID]
	manager.mutex.RUnlock()

	if !exists {
		response := StreamResponse{
			Message: "Поток не найден",
			Error:   "Stream not found",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	serverIP := getServerIP()

	// ✅ ОБНОВЛЕНО: Получаем CDN домен из переменных окружения
	cdnDomain := os.Getenv("CDN_DOMAIN")
	if cdnDomain == "" {
		cdnDomain = serverIP
	}

	// ✅ ОБНОВЛЕНО: Информация о потоке с CDN URLs
	streamData := map[string]interface{}{
		"stream_id":  streamID,
		"status":     stream.Status,
		"start_time": stream.StartTime,
		"srt_port":   stream.SRTPort,
		"server_ip":  serverIP,
		"hls_path":   stream.HLSPath,
		"log_file":   stream.LogFile,
		"mode":       "repack_only",

		// ✅ ОБНОВЛЕННЫЕ URL через CDN/nginx
		"srt_url": fmt.Sprintf("srt://%s:%d?mode=caller&transtype=live&streamid=%s", serverIP, stream.SRTPort, streamID),
		"hls_url": fmt.Sprintf("https://%s/hls/%s/playlist.m3u8", cdnDomain, streamID),
		"hls_api": fmt.Sprintf("http://%s:8081/api/hls/%s", serverIP, streamID),

		"description": "Поток перепаковывается без перекодирования и раздается через CDN",
	}

	// Добавляем информацию о времени начала потока если есть
	if stream.StreamStart != nil {
		streamData["stream_start"] = *stream.StreamStart
		streamData["stream_duration"] = time.Since(*stream.StreamStart).String()
	}

	response := StreamResponse{
		Message:  "Информация о потоке",
		StreamID: streamID,
		Status:   stream.Status,
		Data:     streamData,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ✅ ДОБАВИТЬ перед func main():
func handleHLSMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")

	streamID := strings.TrimPrefix(r.URL.Path, "/api/hls/")
	if streamID == "" {
		response := StreamResponse{
			Message: "StreamID required",
			Error:   "streamId parameter is missing",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	manager.mutex.RLock()
	stream, exists := manager.streams[streamID]
	manager.mutex.RUnlock()

	if !exists {
		response := StreamResponse{
			Message: "Stream not found",
			Error:   "Stream not found or not active",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	cdnDomain := os.Getenv("CDN_DOMAIN")
	if cdnDomain == "" {
		cdnDomain = getServerIP()
	}

	accessToken := generateStreamToken(streamID)

	response := StreamResponse{
		Message:  "HLS stream metadata",
		StreamID: streamID,
		Status:   stream.Status,
		Data: map[string]interface{}{
			"hls_url":      fmt.Sprintf("https://%s/hls/%s/playlist.m3u8", cdnDomain, streamID),
			"stream_url":   fmt.Sprintf("https://%s/hls/%s/playlist.m3u8?token=%s", cdnDomain, streamID, accessToken),
			"stream_id":    streamID,
			"status":       stream.Status,
			"start_time":   stream.StartTime,
			"cdn_domain":   cdnDomain,
			"access_token": accessToken,
		},
	}

	if stream.StreamStart != nil {
		response.Data.(map[string]interface{})["stream_start"] = *stream.StreamStart
		response.Data.(map[string]interface{})["is_live"] = true
	} else {
		response.Data.(map[string]interface{})["is_live"] = false
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func generateStreamToken(streamID string) string {
	timestamp := time.Now().Unix()
	tokenData := fmt.Sprintf("%s:%d", streamID, timestamp)

	secret := os.Getenv("STREAM_TOKEN_SECRET")
	if secret == "" {
		secret = "default-secret-change-in-production"
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(tokenData+secret)))[:16]
}

func createWrapperScript(streamID string, port int, hlsPath string) (string, error) {
	scriptPath := filepath.Join("/tmp", fmt.Sprintf("ffmpeg_wrapper_%s.sh", streamID))

	script := fmt.Sprintf(`#!/bin/bash
STREAM_ID="%s"
SRT_PORT=%d
HLS_PATH="%s"
LOG_FILE="/app/logs/%s.log"

echo "$(date): 🚀 Запуск устойчивого wrapper для потока $STREAM_ID" >> "$LOG_FILE"

# ✅ ИСПРАВЛЕНО: Правильный синтаксис функции в bash
cleanup() {
    echo "$(date): 🛑 Получен сигнал завершения для потока $STREAM_ID" >> "$LOG_FILE"
    if [ ! -z "$FFMPEG_PID" ]; then
        kill -TERM "$FFMPEG_PID" 2>/dev/null
        wait "$FFMPEG_PID" 2>/dev/null
    fi
    exit 0
}

# Устанавливаем обработчики сигналов
trap cleanup TERM INT QUIT

# ✅ ОСНОВНОЙ ЦИКЛ: Бесконечный перезапуск ffmpeg
RESTART_COUNT=0
while true; do
    # Проверяем файл флага остановки
    if [ -f "/tmp/stop_$STREAM_ID" ]; then
        echo "$(date): 🔴 Получен флаг остановки для потока $STREAM_ID" >> "$LOG_FILE"
        rm -f "/tmp/stop_$STREAM_ID"
        break
    fi
    
    RESTART_COUNT=$((RESTART_COUNT + 1))
    echo "$(date): 🔄 Запуск FFmpeg для потока $STREAM_ID на порту $SRT_PORT (попытка #$RESTART_COUNT)" >> "$LOG_FILE"
    
    # ✅ ИСПРАВЛЕНО: Убрал проблемные reconnect параметры, добавил & для фонового запуска
    ffmpeg \
        -hide_banner \
        -loglevel warning \
        -f mpegts \
        -timeout 10000000 \
        -i "srt://0.0.0.0:$SRT_PORT?mode=listener&transtype=live&streamid=$STREAM_ID&latency=2000000&rcvbuf=100000000&sndbuf=100000000" \
        -c:v copy \
        -c:a copy \
        -avoid_negative_ts make_zero \
        -copyts \
        -start_at_zero \
        -f hls \
        -hls_time 4 \
        -hls_list_size 6 \
        -hls_delete_threshold 1 \
        -hls_flags delete_segments+append_list+omit_endlist \
        -hls_segment_type mpegts \
        -hls_segment_filename "$HLS_PATH/segment_%%03d.ts" \
        "$HLS_PATH/playlist.m3u8" \
        >> "$LOG_FILE" 2>&1 &
    
    # Сохраняем PID процесса FFmpeg
    FFMPEG_PID=$!
    echo "$(date): 📊 FFmpeg запущен с PID $FFMPEG_PID" >> "$LOG_FILE"
    
    # ✅ МОНИТОРИНГ: Ждем завершения ffmpeg
    while kill -0 "$FFMPEG_PID" 2>/dev/null; do
        # Проверяем флаг остановки каждую секунду
        if [ -f "/tmp/stop_$STREAM_ID" ]; then
            echo "$(date): ⏹️ Остановка по флагу, завершаем FFmpeg PID $FFMPEG_PID" >> "$LOG_FILE"
            kill -TERM "$FFMPEG_PID" 2>/dev/null
            wait "$FFMPEG_PID" 2>/dev/null
            rm -f "/tmp/stop_$STREAM_ID"
            exit 0
        fi
        sleep 1
    done
    
    # Получаем код завершения
    wait "$FFMPEG_PID" 2>/dev/null
    EXIT_CODE=$?
    
    # ✅ АНАЛИЗ ПРИЧИНЫ ЗАВЕРШЕНИЯ
    case $EXIT_CODE in
        0)
            echo "$(date): ✅ FFmpeg завершился корректно (код $EXIT_CODE)" >> "$LOG_FILE"
            sleep 2
            ;;
        1)
            echo "$(date): 🔌 FFmpeg завершился из-за EOF/потери соединения (код $EXIT_CODE)" >> "$LOG_FILE"
            echo "$(date): ⚡ Быстрый перезапуск через 1 секунду" >> "$LOG_FILE"
            sleep 1
            ;;
        255)
            echo "$(date): 📡 FFmpeg завершился из-за сетевой ошибки (код $EXIT_CODE)" >> "$LOG_FILE"
            sleep 2
            ;;
        *)
            echo "$(date): ❌ FFmpeg завершился с ошибкой $EXIT_CODE" >> "$LOG_FILE"
            sleep 5
            ;;
    esac
    
    # Ограничение на количество быстрых перезапусков
    if [ $RESTART_COUNT -gt 50 ]; then
        echo "$(date): ⚠️ Слишком много перезапусков, увеличиваем задержку" >> "$LOG_FILE"
        sleep 30
        RESTART_COUNT=0
    fi
done

echo "$(date): 🏁 Wrapper для потока $STREAM_ID завершен (перезапусков: $RESTART_COUNT)" >> "$LOG_FILE"
`, streamID, port, hlsPath, streamID)

	// Записываем скрипт в файл
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	if err != nil {
		return "", fmt.Errorf("не удалось создать wrapper-скрипт: %v", err)
	}

	log.Printf("✅ Создан устойчивый wrapper-скрипт: %s", scriptPath)
	return scriptPath, nil
}

// ОБНОВЛЕННАЯ функция startStream с новой механикой статусов
func startStream(streamID string) StreamResponse {
	manager.mutex.Lock()

	// Проверяем, не существует ли уже поток
	if _, exists := manager.streams[streamID]; exists {
		manager.mutex.Unlock()
		return StreamResponse{
			Message: "Поток уже запущен",
			Error:   "Stream already running",
		}
	}

	// Генерируем уникальный порт
	port := manager.basePort
	manager.basePort++

	manager.mutex.Unlock()

	// Создаем директории
	hlsPath := filepath.Join("/app/hls", streamID)
	err := os.MkdirAll(hlsPath, 0755)
	if err != nil {
		return StreamResponse{
			Message: "Ошибка создания директории HLS",
			Error:   err.Error(),
		}
	}
	// ✅ ДОБАВИТЬ: Немедленно уведомляем о starting
	go notifyMainApp(streamID, "starting")

	// Создаем wrapper-скрипт
	scriptPath, err := createWrapperScript(streamID, port, hlsPath)
	if err != nil {
		return StreamResponse{
			Message: "Ошибка создания wrapper-скрипта",
			Error:   err.Error(),
		}
	}

	// Создаем лог-файл
	logFile := fmt.Sprintf("/app/logs/%s.log", streamID)
	logFileHandle, err := os.Create(logFile)
	if err != nil {
		return StreamResponse{
			Message: "Ошибка создания лог-файла",
			Error:   err.Error(),
		}
	}
	defer logFileHandle.Close()

	// ✅ ИСПРАВЛЕНО: Запускаем wrapper-скрипт и НЕ ЖДЕМ его завершения
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdout = logFileHandle
	cmd.Stderr = logFileHandle

	log.Printf("Запуск wrapper-скрипта для перепаковки потока %s: %s", streamID, scriptPath)

	// ✅ КЛЮЧЕВОЕ ИЗМЕНЕНИЕ: Запускаем в фоне и НЕ ждем завершения
	err = cmd.Start()
	if err != nil {
		return StreamResponse{
			Message: "Ошибка запуска wrapper-скрипта",
			Error:   err.Error(),
		}
	}

	serverIP := os.Getenv("SERVER_IP")
	if serverIP == "" {
		serverIP = "localhost"
	}

	// Создаем объект потока
	stream := &StreamInstance{
		StreamID:    streamID,
		Status:      "starting",
		StartTime:   time.Now(),
		StreamStart: nil,
		Process:     cmd, // Сохраняем процесс для возможности остановки
		LogFile:     logFile,
		HLSPath:     hlsPath,
		SRTPort:     port,
		ServerIP:    serverIP,
	}

	manager.mutex.Lock()
	manager.streams[streamID] = stream
	manager.mutex.Unlock()

	// ✅ ВАЖНО: Запускаем мониторинг в отдельной горутине, которая НЕ ждет завершения процесса
	go func() {
		// Запускаем HLS мониторинг
		go monitorHLSActivity(streamID, hlsPath)

		log.Printf("🚀 Поток %s запущен с автоперезапуском, мониторинг активен", streamID)

		// ✅ НЕ ВЫЗЫВАЕМ cmd.Wait() - wrapper работает бесконечно в фоне
		// Если нужно дождаться завершения, это будет сделано в stopStream

	}()

	return StreamResponse{
		Message:  "Поток запущен",
		StreamID: streamID,
		Status:   "starting",
		Data: StreamInstanceResponse{
			StreamID:  streamID,
			Status:    "starting",
			StartTime: stream.StartTime,
			SRTPort:   port,
			ServerIP:  serverIP,
			SRTURL:    fmt.Sprintf("srt://%s:%d?streamid=%s", serverIP, port, streamID),
			HLSURL:    fmt.Sprintf("http://%s:8081/hls/%s/playlist.m3u8", serverIP, streamID),
		},
	}
}

func stopStream(streamID string) StreamResponse {
	manager.mutex.Lock()

	stream, exists := manager.streams[streamID]
	if !exists {
		manager.mutex.Unlock()
		return StreamResponse{
			Message: "Поток не найден",
			Error:   "Stream not found",
		}
	}

	log.Printf("🛑 Начинаем остановку потока %s", streamID)
	manager.mutex.Unlock()

	// ✅ Создаем файл флага для остановки wrapper-скрипта
	stopFlagFile := fmt.Sprintf("/tmp/stop_%s", streamID)
	err := os.WriteFile(stopFlagFile, []byte("stop"), 0644)
	if err != nil {
		log.Printf("⚠️ Не удалось создать флаг остановки: %v", err)
	}

	// ✅ Даем время wrapper-скрипту завершиться корректно
	time.Sleep(2 * time.Second)

	// ✅ Если процесс все еще работает, принудительно завершаем
	if stream.Process != nil && stream.Process.ProcessState == nil {
		log.Printf("🔄 Принудительное завершение процесса для потока %s", streamID)

		// Отправляем SIGTERM
		err = stream.Process.Process.Signal(os.Interrupt)
		if err != nil {
			log.Printf("⚠️ Ошибка отправки SIGTERM: %v", err)
		}

		// Ждем 5 секунд
		done := make(chan error, 1)
		go func() {
			done <- stream.Process.Wait()
		}()

		select {
		case <-done:
			log.Printf("✅ Процесс для потока %s завершен корректно", streamID)
		case <-time.After(5 * time.Second):
			log.Printf("⚠️ Принудительное завершение процесса для потока %s", streamID)
			stream.Process.Process.Kill()
		}
	}

	// ✅ Очищаем HLS файлы
	if stream.HLSPath != "" {
		log.Printf("🧹 Очистка HLS файлов для потока %s: %s", streamID, stream.HLSPath)
		os.RemoveAll(stream.HLSPath)
	}

	// Удаляем wrapper-скрипт
	scriptPath := filepath.Join("/tmp", fmt.Sprintf("ffmpeg_wrapper_%s.sh", streamID))
	os.Remove(scriptPath)

	// ✅ Удаляем флаг остановки
	os.Remove(stopFlagFile)

	// ✅ Удаляем поток из управления
	manager.mutex.Lock()
	delete(manager.streams, streamID)
	manager.mutex.Unlock()

	log.Printf("✅ Поток %s полностью остановлен и очищен", streamID)

	// Уведомляем основное приложение
	go notifyMainApp(streamID, "stopped")

	return StreamResponse{
		Message:  "Поток остановлен",
		StreamID: streamID,
		Status:   "stopped",
	}
}

func handlePlaylistDebug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	streamID := strings.TrimPrefix(r.URL.Path, "/api/debug/")
	if streamID == "" {
		http.Error(w, "StreamID required", http.StatusBadRequest)
		return
	}

	manager.mutex.RLock()
	stream, exists := manager.streams[streamID]
	manager.mutex.RUnlock()

	if !exists {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	// Читаем содержимое плейлиста
	playlists := make(map[string]string)
	playlistPath := filepath.Join(stream.HLSPath, "playlist.m3u8")

	if content, err := os.ReadFile(playlistPath); err == nil {
		playlists["playlist.m3u8"] = string(content)
	} else {
		playlists["playlist.m3u8"] = fmt.Sprintf("Error reading file: %v", err)
	}

	// Список сегментов
	var segments []string
	if entries, err := os.ReadDir(stream.HLSPath); err == nil {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".ts") {
				segments = append(segments, entry.Name())
			}
		}
	}

	response := map[string]interface{}{
		"stream_id":   streamID,
		"status":      stream.Status,
		"mode":        "repack_only",
		"playlists":   playlists,
		"segments":    segments,
		"hls_path":    stream.HLSPath,
		"start_time":  stream.StartTime,
		"description": "Поток перепаковывается без перекодирования",
	}

	if stream.StreamStart != nil {
		response["stream_start"] = *stream.StreamStart
		response["stream_duration"] = time.Since(*stream.StreamStart).String()
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func getServerIP() string {
	// Сначала проверяем переменную окружения
	if serverIP := os.Getenv("SERVER_IP"); serverIP != "" {
		return serverIP
	}

	// Пытаемся определить автоматически
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Printf("Ошибка определения IP: %v, используем localhost", err)
		return "localhost"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// ✅ ДОБАВИТЬ новую функцию webhook
func notifyMainApp(streamID, status string) {
	mainAppURL := os.Getenv("MAIN_APP_URL")
	if mainAppURL == "" {
		mainAppURL = "http://go-app:8080"
	}

	updateURL := fmt.Sprintf("%s/api/internal/stream-status", mainAppURL)

	requestBody := StatusUpdateRequest{
		StreamID: streamID,
		Status:   status,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("❌ Failed to marshal webhook data: %v", err)
		return
	}

	// ✅ ДОБАВЛЯЕМ RETRY-ЛОГИКУ
	maxRetries := 3
	client := &http.Client{Timeout: 5 * time.Second}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("📡 Webhook attempt %d/%d: %s -> %s", attempt, maxRetries, streamID, status)

		resp, err := client.Post(updateURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("❌ Webhook attempt %d failed: %v", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return
		}
		defer resp.Body.Close()

		responseBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == 200 {
			log.Printf("✅ Webhook success on attempt %d: %s", attempt, string(responseBody))
			return
		} else {
			log.Printf("❌ Webhook attempt %d failed with status %d: %s", attempt, resp.StatusCode, string(responseBody))
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
		}
	}

	log.Printf("❌ All webhook attempts failed for %s -> %s", streamID, status)
}

// ✅ НОВАЯ ФУНКЦИЯ: Восстановление активных потоков
func restoreActiveStreams() {
	log.Printf("🔄 Начинаем восстановление активных потоков...")

	// Получаем список активных потоков из основного приложения
	activeStreams, err := getActiveStreamsFromMainApp()
	if err != nil {
		log.Printf("❌ Ошибка получения активных потоков: %v", err)
		return
	}

	if len(activeStreams) == 0 {
		log.Printf("✅ Нет активных потоков для восстановления")
		return
	}

	log.Printf("🔄 Найдено %d активных потоков для восстановления", len(activeStreams))

	for _, stream := range activeStreams {
		log.Printf("🔄 Восстановление потока: %s (статус: %s)", stream.StreamID, stream.StreamStatus)

		// Запускаем поток заново
		response := startStream(stream.StreamID)
		if response.Error != "" {
			log.Printf("❌ Ошибка восстановления потока %s: %s", stream.StreamID, response.Error)
		} else {
			log.Printf("✅ Поток %s успешно восстановлен", stream.StreamID)
		}

		// Небольшая задержка между запусками
		time.Sleep(1 * time.Second)
	}

	log.Printf("✅ Восстановление потоков завершено")
}

// ✅ НОВАЯ ФУНКЦИЯ: Получение активных потоков из основного приложения
func getActiveStreamsFromMainApp() ([]StreamInfo, error) {
	mainAppURL := os.Getenv("MAIN_APP_URL")
	if mainAppURL == "" {
		mainAppURL = "http://go-app:8080"
	}

	urls := []string{
		fmt.Sprintf("%s/api/tasks?stream_status=starting", mainAppURL),
		fmt.Sprintf("%s/api/tasks?stream_status=running", mainAppURL),
	}

	var allStreams []StreamInfo

	for _, url := range urls {
		log.Printf("📡 Запрос активных потоков: %s", url)

		// ✅ ДОБАВЛЯЕМ RETRY-ЛОГИКУ
		maxRetries := 5
		client := &http.Client{Timeout: 10 * time.Second}

		for attempt := 1; attempt <= maxRetries; attempt++ {
			resp, err := client.Get(url)
			if err != nil {
				log.Printf("❌ Попытка %d/%d не удалась для %s: %v", attempt, maxRetries, url, err)
				if attempt < maxRetries {
					time.Sleep(time.Duration(attempt) * 2 * time.Second)
					continue
				}
				break
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				log.Printf("❌ Ошибка ответа от %s: %d", url, resp.StatusCode)
				if attempt < maxRetries {
					time.Sleep(time.Duration(attempt) * 2 * time.Second)
					continue
				}
				break
			}

			var response TasksResponse
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				log.Printf("❌ Ошибка парсинга ответа от %s: %v", url, err)
				if attempt < maxRetries {
					time.Sleep(time.Duration(attempt) * 2 * time.Second)
					continue
				}
				break
			}

			log.Printf("📡 Получено %d потоков со статусом из %s", len(response.Data), url)
			allStreams = append(allStreams, response.Data...)
			break // Успешно получили данные
		}
	}

	return allStreams, nil
}
