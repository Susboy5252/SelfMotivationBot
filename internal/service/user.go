package service

import (
	"context"
	"errors"
	"fmt"
	"motivator/internal/models"
	"motivator/internal/repository"
)

var (
	// ErrUserNotFound is returned when a referenced user doesn't exist.
	ErrUserNotFound = errors.New("user not found")
)

// UserService encapsulates user-related business logic.
type UserService struct {
	users repository.UserRepository
	geo   *GeoService
}

// NewUserService creates a UserService.
func NewUserService(users repository.UserRepository, geo *GeoService) *UserService {
	return &UserService{users: users, geo: geo}
}

// EnsureUser creates the user if they don't exist yet, otherwise does nothing.
func (s *UserService) EnsureUser(ctx context.Context, id int64, username string) error {
	return s.users.Create(ctx, id, username)
}

// GetUser returns a user by ID. Returns nil if not found (no error).
func (s *UserService) GetUser(ctx context.Context, id int64) (*models.User, error) {
	return s.users.GetByID(ctx, id)
}

// ResolveUserByUsername finds a user by their @username.
// Returns ErrUserNotFound if the user is not registered.
func (s *UserService) ResolveUserByUsername(ctx context.Context, username string) (*models.User, error) {
	u, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("userService.ResolveUserByUsername: %w", err)
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// SetCity updates the user's city and resolves their timezone from it.
// Returns the detected timezone identifier and the local time string.
func (s *UserService) SetCity(ctx context.Context, userID int64, city string) (timezone string, err error) {
	coords, err := s.geo.GetCoordinates(city)
	if err != nil {
		return "", fmt.Errorf("userService.SetCity: %w", err)
	}

	tz, err := s.geo.GetTimezone(coords.Lat, coords.Lng)
	if err != nil {
		return "", fmt.Errorf("userService.SetCity: %w", err)
	}

	if err := s.users.UpdateCity(ctx, userID, city); err != nil {
		return "", err
	}
	if err := s.users.UpdateTimezone(ctx, userID, tz); err != nil {
		return "", err
	}

	return tz, nil
}

func (s *UserService) SetLanguage(ctx context.Context, userID int64, language string) error {
	if err := s.users.UpdateLanguage(ctx, userID, language); err != nil {
		return err
	}
	return nil
}

// GetTimezone returns the user's timezone.
func (s *UserService) GetTimezone(ctx context.Context, userID int64) (string, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", ErrUserNotFound
	}
	return u.TimeZone, nil
}
