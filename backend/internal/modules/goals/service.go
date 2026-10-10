package goals

import (
	"context"
	"time"
	"uuid"
)

type Repository interface {
	Create(ctx context.Context, goal Goal) error
	Get(ctx context.Context, id uuid.UUID) (Goal, error)
	List(ctx context.Context) ([]Goal, error)
}

type Service struct {
	repo  Repository
	now   func() time.Time
	newID func() uuid.UUID
}

func NewService(repo Repository, now func() time.Time) *Service {
	return &Service{repo: repo, now: now, newID: uuid.NewV7}
}

func (s *Service) Create(ctx context.Context, name string) (Goal, error) {
	normalized, err := NormalizeName(name)
	if err != nil {
		return Goal{}, err
	}
	goal := Goal{ID: s.newID(), Name: normalized, CreatedAt: s.now().UTC()}
	if err := s.repo.Create(ctx, goal); err != nil {
		return Goal{}, err
	}
	return goal, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Goal, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Goal, error) {
	return s.repo.List(ctx)
}
