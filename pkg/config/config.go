package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// ✅ НОВАЯ ФУНКЦИЯ: Загрузка .env файла
func init() {
	// Загружаем .env файл при инициализации пакета
	err := godotenv.Load()
	if err != nil {
		// В production .env может отсутствовать - это нормально
		log.Printf("⚠️ Предупреждение: .env файл не найден: %v", err)
	} else {
		log.Println("✅ .env файл успешно загружен")
	}
}

// GetEnv возвращает значение переменной окружения или значение по умолчанию
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Config содержит настройки приложения
type Config struct {
	DatabaseConfig *DatabaseConfig
	ServerConfig   *ServerConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ServerConfig struct {
	Port                string
	StreamingServiceURL string
	ServerIP            string
}

// NewConfig создает новую конфигурацию из переменных окружения
func NewConfig() *Config {
	return &Config{
		DatabaseConfig: &DatabaseConfig{
			Host:     GetEnv("DB_HOST", "localhost"),
			Port:     GetEnv("DB_PORT", "5432"),
			User:     GetEnv("DB_USER", "postgres"),
			Password: GetEnv("DB_PASSWORD", "postgres"),
			DBName:   GetEnv("DB_NAME", "goapp"),
			SSLMode:  GetEnv("DB_SSLMODE", "disable"),
		},
		ServerConfig: &ServerConfig{
			Port:                GetEnv("SERVER_PORT", ":8080"),
			StreamingServiceURL: GetEnv("STREAMING_SERVICE_URL", "http://streaming-service:8081"),
			ServerIP:            GetEnv("SERVER_IP", "192.168.3.55"),
		},
	}
}
