package main

import (
	"fmt"
	"errors"
	"strings"
)

type Storage interface {
	Add(a Annotation) error
	Posts(tagsFilter []string, r, until int) (res Posts, err error)
	Close()
}

type Annotation struct {
	CreatedAt int      `json:"created_at,omitempty"`
	Message   string   `json:"message"                binding:"required"`
	Tags      []string `json:"tags,omitempty"         binding:"required"`
}

type Posts struct {
	Posts []Annotation `json:"posts"`
}

func NewStorage(config string) (Storage, error) {
	parts := strings.SplitN(config, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid config format")
	}

	switch parts[0] {
	case "local":
		{
			return NewBoltDBStorage(parts[1])
		}
	}
	return nil, errors.New(fmt.Sprintf("invalid config, type \"%s\" not supported", parts[0]))
}
