package main

import (
	"log"
	"errors"
	"fmt"
	"strings"
)

type TagStats map[string]int

type Storage interface {
	Add(a Annotation) error
	ListForTag(tag string, r, until int, out *[]Annotation) (err error)
	TagStats() (TagStats, error)
	AllTags() []string
	Close()
	Cleanup() // after tests
}

type Annotation struct {
	CreatedAt int      `json:"created_at,omitempty"   gorethink:"created_at"`
	Message   string   `json:"message"                gorethink:"message"`
	Tags      []string `json:"tags,omitempty"         gorethink:"tags"`
}

type Posts struct {
	Posts []Annotation `json:"posts"`
}

func NewStorage(config string) (Storage, error) {
    log.Printf("Storage config: %s", config)
    
	parts := strings.SplitN(config, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid config format")
	}

	switch parts[0] {
	case "local":
		{
			return NewBoltDBStorage(parts[1])
		}
	case "rethinkdb":
		{
			return NewRethinkDBStorage(parts[1])
		}
	}
	return nil, fmt.Errorf("invalid config, type \"%s\" not supported", parts[0])
}

func GetPosts(s Storage, tags []string, ra, until int) (res Posts, err error) {
	res.Posts = make([]Annotation, 0)
	for _, tag := range tags {
		if s.ListForTag(tag, ra, until, &res.Posts) != nil {
			return res, err
		}
	}
	return res, nil
}
