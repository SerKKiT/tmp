package stream

import "context"

type Repository interface {
	Create(ctx context.Context, stream *Stream) error
	GetByID(ctx context.Context, id uint) (*Stream, error)
	GetByStreamID(ctx context.Context, streamID string) (*Stream, error)
	List(ctx context.Context, filter *Filter) ([]*Stream, error)
	UpdateStatus(ctx context.Context, streamID string, status Status) error
	Update(ctx context.Context, id uint, stream *Stream) error
	Delete(ctx context.Context, id uint) error
	Count(ctx context.Context, filter *Filter) (int64, error)
}

type Filter struct {
	Status Status
	Limit  int
	Offset int
}
