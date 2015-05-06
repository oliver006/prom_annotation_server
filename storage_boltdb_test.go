package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

/*
  for html coverage report run
  go test -coverprofile=coverage.out  && go tool cover -html=coverage.out
*/

func TestBoltAnnotationAdd(t *testing.T) {

	ts := int(time.Now().Unix())
	dbFile := fmt.Sprintf("./test-%d.db", ts)
	s, err := NewBoltDBStorage(dbFile)
	defer os.Remove(dbFile)
	if err != nil {
		t.Errorf("no good: %s", err)
		return
	}
	defer s.Close()

	a := Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag1", "tag2"}}

	count := s.GetCount("tag1")
	err = s.Add(a)
	if err != nil {
		t.Errorf("no good: %s", err)
		return
	}

	if diff := s.GetCount("tag1") - count; err != nil || diff != 1 {
		t.Errorf("no good: %s", err)
		return
	}
}

func TestBoltGetList(t *testing.T) {

	ts := int(time.Now().Unix())

	dbFile := fmt.Sprintf("./test-%d.db", ts)
	s, err := NewBoltDBStorage(dbFile)
	defer os.Remove(dbFile)
	if err != nil {
		t.Errorf("no good: %s", err)
		return
	}
	defer s.Close()

	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag1", "tag2"}})
	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag2", "tag3"}})
	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag3", "tag4"}})

	if c := s.GetCount("tag2"); c != 2 {
		t.Errorf("no good, wrong count %d", c)
	}

	list, err := s.Posts([]string{"tag1"}, 1000, ts)
	if err != nil || len(list.Posts) != 1 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = s.Posts([]string{"tag2"}, 1000, ts)
	if err != nil || len(list.Posts) != 2 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = s.Posts([]string{"tag1", "tag2"}, 1000, ts)
	if err != nil || len(list.Posts) != 3 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = s.Posts([]string{"tag123"}, 1000, ts)
	if err != nil || len(list.Posts) != 0 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}
}

func TestBoltGetListFilters(t *testing.T) {

	ts := int(time.Now().Unix()) - 10

	dbFile := fmt.Sprintf("./test-%d.db", ts)
	s, err := NewBoltDBStorage(dbFile)
	defer os.Remove(dbFile)
	if err != nil {
		t.Errorf("no good: %s", err)
		return
	}
	defer s.Close()

	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag1"}})
	s.Add(Annotation{CreatedAt: ts + 5, Message: "Test message", Tags: []string{"tag1"}})
	s.Add(Annotation{CreatedAt: ts + 10, Message: "Test message", Tags: []string{"tag1"}})

	if c := s.GetCount("tag1"); c != 3 {
		t.Errorf("no good, wrong count %d", c)
	}

	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag2"}})
	s.Add(Annotation{CreatedAt: ts + 5, Message: "Test message", Tags: []string{"tag2"}})
	s.Add(Annotation{CreatedAt: ts + 10, Message: "Test message", Tags: []string{"tag2"}})

	if c := s.GetCount("tag2"); c != 3 {
		t.Errorf("no good, wrong count %d", c)
	}

	list, err := s.Posts([]string{"tag1"}, 1000, ts+10)
	if err != nil || len(list.Posts) != 3 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = s.Posts([]string{"tag1"}, 1000, ts+5)
	if err != nil || len(list.Posts) != 2 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = s.Posts([]string{"tag1"}, 1000, ts)
	if err != nil || len(list.Posts) != 1 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = s.Posts([]string{"tag1"}, 1000, ts-5)
	if err != nil || len(list.Posts) != 0 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = s.Posts([]string{"tag1", "tag2"}, 1000, ts+10)
	if err != nil || len(list.Posts) != 6 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = s.Posts([]string{"tag1", "tag2"}, 1000, ts+5)
	if err != nil || len(list.Posts) != 4 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}
}
