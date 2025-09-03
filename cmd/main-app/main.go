package main

import (
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	"my-go-app/internal/application/handlers"
	"my-go-app/internal/application/services"
	"my-go-app/internal/infrastructure/database"
	"my-go-app/pkg/config" // ✅ ИСПОЛЬЗУЕМ config пакет
	"my-go-app/pkg/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// ✅ ИСПОЛЬЗУЕМ централизованную конфигурацию
	cfg := config.NewConfig()

	dbConfig := &database.Config{
		Host:     cfg.DatabaseConfig.Host,
		Port:     cfg.DatabaseConfig.Port,
		User:     cfg.DatabaseConfig.User,
		Password: cfg.DatabaseConfig.Password,
		DBName:   cfg.DatabaseConfig.DBName,
		SSLMode:  cfg.DatabaseConfig.SSLMode,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	streamRepo := database.NewStreamRepository(db)
	streamService := services.NewStreamService(streamRepo)

	streamHandler := handlers.NewStreamHandler(streamService)
	healthHandler := handlers.NewHealthHandler(db)
	internalHandler := handlers.NewInternalHandler(streamService) // ✅ НОВЫЙ HANDLER

	timeoutShort := middleware.TimeoutMiddleware(5 * time.Second)
	timeoutMedium := middleware.TimeoutMiddleware(15 * time.Second)
	timeoutLong := middleware.TimeoutMiddleware(30 * time.Second)

	http.Handle("/api/tasks", timeoutLong(http.HandlerFunc(streamHandler.HandleStreams)))
	http.Handle("/api/tasks/", timeoutMedium(http.HandlerFunc(streamHandler.HandleStreamByID)))
	http.Handle("/api/streams/", timeoutLong(http.HandlerFunc(streamHandler.HandleStreamControl)))
	http.Handle("/api/health", timeoutShort(http.HandlerFunc(healthHandler.HandleHealth)))

	// ✅ НОВЫЙ ENDPOINT для внутренних обновлений
	http.Handle("/api/internal/stream-status", timeoutShort(http.HandlerFunc(internalHandler.HandleStreamStatusUpdate)))
	// ✅ НОВЫЙ ENDPOINT: Proxy для streaming service
	http.Handle("/api/streaming-proxy/", timeoutMedium(http.HandlerFunc(streamHandler.HandleStreamingServiceProxy)))

	http.Handle("/", http.FileServer(http.Dir("./static/")))

	log.Printf("🚀 Server starting on %s", cfg.ServerConfig.Port)
	log.Fatal(http.ListenAndServe(cfg.ServerConfig.Port, nil))
}
