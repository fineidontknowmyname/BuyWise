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
	Update(ctx context.Context, goal Goal) error
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
	return s.CreateWithDetails(ctx, Goal{Name: name})
}

func (s *Service) CreateWithDetails(ctx context.Context, g Goal) (Goal, error) {
	normalized, err := NormalizeName(g.Name)
	if err != nil {
		return Goal{}, err
	}
	g.Name = normalized
	var empty uuid.UUID
	if g.ID == empty {
		g.ID = s.newID()
	}
	g.CreatedAt = s.now().UTC()
	if g.Priority == "" {
		g.Priority = "medium"
	}
	if g.Status == "" {
		g.Status = "active"
	}

	if err := s.repo.Create(ctx, g); err != nil {
		return Goal{}, err
	}
	return g, nil
}

func (s *Service) Update(ctx context.Context, g Goal) (Goal, error) {
	normalized, err := NormalizeName(g.Name)
	if err != nil {
		return Goal{}, err
	}
	g.Name = normalized
	if g.Priority == "" {
		g.Priority = "medium"
	}
	if g.Status == "" {
		g.Status = "active"
	}

	if err := s.repo.Update(ctx, g); err != nil {
		return Goal{}, err
	}
	return g, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Goal, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]Goal, error) {
	return s.repo.List(ctx)
}
