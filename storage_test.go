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
	_, err := NewStorage("INVALID")
	if err == nil {
		t.Errorf("terrible, this shouldn't succeed")
	}

	_, err = NewStorage("INVALID:1234")
	if err == nil {
		t.Errorf("terrible, this shouldn't succeed")
	}

	_, err = NewStorage("local:/proc/123.db")
	if err == nil {
		t.Errorf("terrible: %s", err)
	}

	// valid ones
	s, err := NewStorage("local:/tmp/123.db")
	if err != nil {
		t.Errorf("terrible: %s", err)
	}
	defer s.Close()

}
