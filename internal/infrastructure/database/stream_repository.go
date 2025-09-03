package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"my-go-app/internal/domain/stream"
)

type StreamRepository struct {
	db *gorm.DB
}

func NewStreamRepository(db *gorm.DB) *StreamRepository {
	return &StreamRepository{db: db}
}

func (r *StreamRepository) Create(ctx context.Context, s *stream.Stream) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *StreamRepository) GetByID(ctx context.Context, id uint) (*stream.Stream, error) {
	var s stream.Stream
	err := r.db.WithContext(ctx).First(&s, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("stream not found")
		}
		return nil, err
	}
	return &s, nil
}

func (r *StreamRepository) GetByStreamID(ctx context.Context, streamID string) (*stream.Stream, error) {
	var s stream.Stream
	err := r.db.WithContext(ctx).Where("stream_id = ?", streamID).First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("stream not found")
		}
		return nil, err
	}
	return &s, nil
}

func (r *StreamRepository) List(ctx context.Context, filter *stream.Filter) ([]*stream.Stream, error) {
	var streams []*stream.Stream

	query := r.db.WithContext(ctx)

	if filter != nil {
		if filter.Status != "" {
			query = query.Where("stream_status = ?", filter.Status)
		}
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	err := query.Find(&streams).Error
	return streams, err
}

func (r *StreamRepository) UpdateStatus(ctx context.Context, streamID string, status stream.Status) error {
	return r.db.WithContext(ctx).Model(&stream.Stream{}).
		Where("stream_id = ?", streamID).
		Update("stream_status", status).Error
}

func (r *StreamRepository) Update(ctx context.Context, id uint, s *stream.Stream) error {
	return r.db.WithContext(ctx).Model(&stream.Stream{}).Where("id = ?", id).Updates(s).Error
}

func (r *StreamRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&stream.Stream{}, id).Error
}

func (r *StreamRepository) Count(ctx context.Context, filter *stream.Filter) (int64, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&stream.Stream{})

	if filter != nil && filter.Status != "" {
		query = query.Where("stream_status = ?", filter.Status)
	}

	err := query.Count(&count).Error
	return count, err
}
