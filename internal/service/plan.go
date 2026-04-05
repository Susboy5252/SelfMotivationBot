package service

import (
	"context"
	"motivator/internal/models"
	"motivator/internal/repository"
)

// PlanService encapsulates plan-related business logic.
type PlanService struct {
	plans repository.PlanRepository
}

// NewPlanService creates a PlanService.
func NewPlanService(plans repository.PlanRepository) *PlanService {
	return &PlanService{plans: plans}
}

// Save creates a new plan entry.
func (s *PlanService) Save(ctx context.Context, userID int64, content string) (int64, error) {
	return s.plans.Create(ctx, userID, content)
}

// GetAll returns all plans for a user.
func (s *PlanService) GetAll(ctx context.Context, userID int64) ([]models.Plan, error) {
	return s.plans.GetByUserID(ctx, userID)
}
