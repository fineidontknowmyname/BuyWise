package goals

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"
)

const MaxNameLength = 200

var (
	ErrInvalidName = errors.New("goal name must be 1 to 200 characters")
	ErrNotFound    = errors.New("goal not found")
)

type Goal struct {
	AllocatedPaise   int64      `json:"allocatedPaise"`
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	TargetPricePaise int64      `json:"targetPricePaise"`
	SavedAmountPaise int64      `json:"savedAmountPaise"`
	TargetDate       *string    `json:"targetDate,omitempty"`
	Priority         string     `json:"priority"`
	ProductID        *uuid.UUID `json:"productId,omitempty"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
}

func NormalizeName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > MaxNameLength {
		return "", ErrInvalidName
	}
	return trimmed, nil
}
