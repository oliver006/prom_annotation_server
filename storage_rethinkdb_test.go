package main

import (
	"fmt"
	"testing"
	"time"
)

/*
  for html coverage report run
  go test -coverprofile=coverage.out  && go tool cover -html=coverage.out
*/

func TestRethinkAnnotationAdd(t *testing.T) {
	ts := int(time.Now().Unix())
	dbName := fmt.Sprintf("anno%d", ts)
	s, err := NewRethinkDBStorage("localhost:28015/" + dbName)
	if err != nil {
		t.Errorf("no good: %s", err)
		return
	}
	defer s.Cleanup()

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

func TestRethinkGetList(t *testing.T) {
	ts := int(time.Now().Unix())
	dbName := fmt.Sprintf("anno%d", ts)
	s, err := NewRethinkDBStorage("localhost:28015/" + dbName)
	if err != nil {
		t.Errorf("no good: %s", err)
		return
	}
	defer s.Cleanup()

	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag1", "tag2"}})
	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag2", "tag3"}})
	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag3", "tag4"}})

	if c := s.GetCount("tag2"); c != 2 {
		t.Errorf("no good, wrong count %d", c)
	}

	list, err := GetPosts(s, []string{"tag1"}, 1000, ts)
	if err != nil || len(list.Posts) != 1 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = GetPosts(s, []string{"tag2"}, 1000, ts)
	if err != nil || len(list.Posts) != 2 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = GetPosts(s, []string{"tag1", "tag2"}, 1000, ts)
	if err != nil || len(list.Posts) != 3 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = GetPosts(s, []string{"tag123"}, 1000, ts)
	if err != nil || len(list.Posts) != 0 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}
}

func TestRethinkGetListFilters(t *testing.T) {
	ts := int(time.Now().Unix())
	dbName := fmt.Sprintf("anno%d", ts)
	s, err := NewRethinkDBStorage("localhost:28015/" + dbName)
	if err != nil {
		t.Errorf("no good: %s", err)
		return
	}
	defer s.Cleanup()

	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag1"}})
	s.Add(Annotation{CreatedAt: ts - 5, Message: "Test message", Tags: []string{"tag1"}})
	s.Add(Annotation{CreatedAt: ts - 10, Message: "Test message", Tags: []string{"tag1"}})

	if c := s.GetCount("tag1"); c != 3 {
		t.Errorf("no good, wrong count %d != 3", c)
	}

	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag2"}})
	s.Add(Annotation{CreatedAt: ts - 5, Message: "Test message", Tags: []string{"tag2"}})
	s.Add(Annotation{CreatedAt: ts - 10, Message: "Test message", Tags: []string{"tag2"}})

	if c := s.GetCount("tag2"); c != 3 {
		t.Errorf("no good, wrong count %d != 3", c)
	}

	list, err := GetPosts(s, []string{"tag1"}, 1000, ts)
	if err != nil || len(list.Posts) != 3 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = GetPosts(s, []string{"tag1"}, 1000, ts-4)
	if err != nil || len(list.Posts) != 2 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = GetPosts(s, []string{"tag1"}, 1000, ts-9)
	if err != nil || len(list.Posts) != 1 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = GetPosts(s, []string{"tag1"}, 1000, ts-11)
	if err != nil || len(list.Posts) != 0 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = GetPosts(s, []string{"tag1", "tag2"}, 1000, ts)
	if err != nil || len(list.Posts) != 6 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}

	list, err = GetPosts(s, []string{"tag1", "tag2"}, 1000, ts-4)
	if err != nil || len(list.Posts) != 4 {
		t.Errorf("no good, wrong count, list: %#v, err: %s", list, err)
	}
}

func TestRethinkTagStats(t *testing.T) {
	ts := int(time.Now().Unix())
	dbName := fmt.Sprintf("anno%d", ts)
	s, err := NewRethinkDBStorage("localhost:28015/" + dbName)
	if err != nil {
		t.Errorf("no good: %s", err)
		return
	}
	defer s.Cleanup()

	statsPre, _ := s.TagStats()

	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag1", "tag2"}})
	s.Add(Annotation{CreatedAt: ts, Message: "Test message", Tags: []string{"tag2", "tag3"}})

	statsPost, _ := s.TagStats()

	if statsPre["tag1"] != statsPost["tag1"]-1 || statsPre["tag2"] != statsPost["tag2"]-2 {
		t.Errorf("no good, stats counts not as expected")
		return
	}
}
