package items

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

const (
	minNameLength = 1
	maxNameLength = 200
)

var ErrNotFound = errors.New("item not found")

type Item struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}

	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func validateName(name string) error {
	switch length := utf8.RuneCountInString(name); {
	case length < minNameLength:
		return ValidationError{Field: "name", Message: "name is required"}
	case length > maxNameLength:
		return ValidationError{Field: "name", Message: "name must be at most 200 characters"}
	default:
		return nil
	}
}

func validateID(id string) error {
	if !IsValidUUID(id) {
		return ValidationError{Field: "id", Message: "id must be a valid UUID"}
	}

	return nil
}
