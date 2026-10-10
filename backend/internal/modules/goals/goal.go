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
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

func NormalizeName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > MaxNameLength {
		return "", ErrInvalidName
	}
	return trimmed, nil
}
