package main

import (
	"testing"
)

/*
  for html coverage report run
  go test -coverprofile=coverage.out  && go tool cover -html=coverage.out
*/

func TestStorageConfig(t *testing.T) {

	// first, invalid ones
	invalid := []string{"INVALID", "INVALID:1234", "local:/proc/123.db", ""}
	for _, opt := range invalid {
		_, err := NewStorage(opt)
		if err == nil {
			t.Errorf("terrible: %s", err)
		}
	}

	// valid
	valid := []string{"local:/tmp/123.db", "local:./test-123.db", "rethinkdb:localhost:28015/annotations"}
	for _, opt := range valid {
		s, err := NewStorage(opt)
		if err != nil {
			t.Errorf("terrible: %s", err)
		} else {
			s.Close()
		}
	}
}
