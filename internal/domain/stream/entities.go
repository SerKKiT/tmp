package stream

import (
	//"context"
	"time"
)

type Stream struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"not null;index"`
	StreamID     string    `json:"stream_id" gorm:"uniqueIndex;not null"`
	StreamStatus Status    `json:"stream_status" gorm:"default:'stopped';index"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusError    Status = "error"
)

func (s Status) String() string {
	return string(s)
}

// Валидация статуса
func (s Status) IsValid() bool {
	switch s {
	case StatusStopped, StatusStarting, StatusRunning, StatusError:
		return true
	default:
		return false
	}
}

// ✅ ИСПРАВЛЕННАЯ логика переходов - более гибкая
func (s Status) CanTransitionTo(newStatus Status) bool {
	// Если новый статус невалидный, запрещаем
	if !newStatus.IsValid() {
		return false
	}

	// Если статусы одинаковые, разрешаем (идемпотентность)
	if s == newStatus {
		return true
	}

	switch s {
	case StatusStopped:
		// Из stopped можно перейти в starting или error
		return newStatus == StatusStarting || newStatus == StatusError

	case StatusStarting:
		// Из starting можно перейти в running, stopped или error
		return newStatus == StatusRunning || newStatus == StatusStopped || newStatus == StatusError

	case StatusRunning:
		// ✅ ИСПРАВЛЕНО: Из running можно перейти в stopped, starting или error
		return newStatus == StatusStopped || newStatus == StatusStarting || newStatus == StatusError

	case StatusError:
		// Из error можно перейти в любой статус (восстановление)
		return true

	default:
		// Для неизвестных статусов разрешаем переход
		return true
	}
}
