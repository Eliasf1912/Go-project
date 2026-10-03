package domain

import (
	"strings"
	"time"
)

type Category struct {
	ID        int
	Name      string
	CreatedAt time.Time
}

func NormalizeCategoryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	if name == "" {
		return "", ErrInvalidCategoryName
	}
	return name, nil
}
