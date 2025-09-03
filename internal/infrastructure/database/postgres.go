package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"my-go-app/internal/domain/stream"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewConnection(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	var database *gorm.DB
	var err error

	// Подключение с повторными попытками
	for i := 0; i < 30; i++ {
		database, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger:      logger.Default.LogMode(logger.Info),
			PrepareStmt: true,
		})

		if err == nil {
			break
		}

		log.Printf("Database connection attempt %d/30 failed: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after 30 attempts: %v", err)
	}

	// Настройка пула подключений
	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Автомиграция
	if err := database.AutoMigrate(&stream.Stream{}); err != nil {
		return nil, fmt.Errorf("failed to migrate: %v", err)
	}

	log.Println("✅ Database connected successfully")
	return database, nil
}
