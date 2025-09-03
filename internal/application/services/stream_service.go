package services

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"

	"my-go-app/internal/domain/stream"
)

type StreamService struct {
	repo stream.Repository
}

func NewStreamService(repo stream.Repository) *StreamService {
	return &StreamService{
		repo: repo,
	}
}

type CreateStreamRequest struct {
	Name string `json:"name"`
}

type StreamActionRequest struct {
	Action string `json:"action"`
}

func (s *StreamService) CreateStream(ctx context.Context, req *CreateStreamRequest) (*stream.Stream, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}

	newStream := &stream.Stream{
		Name:         req.Name,
		StreamID:     uuid.New().String(),
		StreamStatus: stream.StatusStopped,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, newStream); err != nil {
		return nil, err
	}

	return newStream, nil
}

func (s *StreamService) GetStreamByID(ctx context.Context, id uint) (*stream.Stream, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *StreamService) GetStreamByStreamID(ctx context.Context, streamID string) (*stream.Stream, error) {
	return s.repo.GetByStreamID(ctx, streamID)
}

func (s *StreamService) ListStreams(ctx context.Context, statusFilter string) ([]*stream.Stream, error) {
	filter := &stream.Filter{}

	if statusFilter != "" {
		status := stream.Status(statusFilter)
		if !status.IsValid() {
			return nil, errors.New("invalid status filter")
		}
		filter.Status = status
	}

	return s.repo.List(ctx, filter)
}

// В internal/application/services/stream_service.go
func (s *StreamService) UpdateStreamStatus(ctx context.Context, streamID string, newStatus stream.Status) error {
	// Получаем текущий stream для валидации
	currentStream, err := s.repo.GetByStreamID(ctx, streamID)
	if err != nil {
		return err
	}

	// ✅ ДОБАВЛЕНО: Детальное логирование
	log.Printf("🔄 Stream %s: attempting transition %s -> %s",
		streamID, currentStream.StreamStatus, newStatus)

	// Проверяем возможность перехода
	if !currentStream.StreamStatus.CanTransitionTo(newStatus) {
		log.Printf("❌ Stream %s: invalid transition %s -> %s",
			streamID, currentStream.StreamStatus, newStatus)
		return errors.New("invalid status transition")
	}

	log.Printf("✅ Stream %s: valid transition %s -> %s",
		streamID, currentStream.StreamStatus, newStatus)

	return s.repo.UpdateStatus(ctx, streamID, newStatus)
}

func (s *StreamService) DeleteStream(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
