package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

/*
  for html coverage report run
  go test -coverprofile=coverage.out  && go tool cover -html=coverage.out
*/

type Setup struct {
	DBFile    string
	TS        int
	ServerURL string
	T         *testing.T
}

func NewSetup(t *testing.T) *Setup {
	ts := int(time.Now().Unix())
	dbFile := fmt.Sprintf("./test-%d.db", ts)
	os.Remove(dbFile)
	
	ctx, err := NewServerContext("local:" + dbFile)
	if err != nil {
		log.Fatalf(err.Error())
	}

	server := httptest.NewServer(ctx.router)
	s := &Setup{
		T:         t,
		ServerURL: fmt.Sprintf("%s/annotations", server.URL),
		TS:        ts,
		DBFile:    dbFile,
	}
	return s
}

func (s *Setup) put(msg, tag string, ts int) error {
	return s.putJSON(fmt.Sprintf(`{"created_at": %d, "message": "%s", "tags": ["%s"]}`, ts, msg, tag))
}

func (s *Setup) putJSON(msg string) error {
	request, err := http.NewRequest("PUT", s.ServerURL, strings.NewReader(msg))
	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	txt, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.HasPrefix(string(txt), `{"result":"ok"`) {
		s.T.Errorf("Expected code of 200, not: %d      txt: %s", res.StatusCode, txt)
	}

	return err
}

func (s *Setup) query(tag string, ts int) (p Posts, err error) {
	queryURL := fmt.Sprintf("%s?until=%d&range=3600&tags[]=%s", s.ServerURL, ts, tag)
	res, err := http.Get(queryURL)
	if err != nil {
		s.T.Errorf("err: %s", err)
		return
	}

	posts, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		s.T.Errorf("err: %s", err)
		return
	}

	if err = json.Unmarshal(posts, &p); err != nil {
		s.T.Errorf("err: %s", err)
		return
	}

	if res.StatusCode != 200 {
		s.T.Errorf("Request failed, code:%d  ", res.StatusCode)
	}
	return
}

func TestServerAddAndQuery(t *testing.T) {
	s := NewSetup(t)
	defer os.Remove(s.DBFile)

	if err := s.put("msg1", "tag1", s.TS-1); err != nil {
		t.Error(err)
	}
	if err := s.put("msg1", "tag1", s.TS-2); err != nil {
		t.Error(err)
	}
	if err := s.put("msg1", "tag2", s.TS-2); err != nil {
		t.Error(err)
	}

	if l, err := s.query("tag1", s.TS); err != nil || len(l.Posts) != 2 {
		t.Errorf("err: %s or Wrong l.Posts: %#v", err, l.Posts)
	}
	if l, err := s.query("tag2", s.TS); err != nil || len(l.Posts) != 1 {
		t.Errorf("err: %s or Wrong l.Posts: %#v", err, l.Posts)
	}
}

func TestServerDefaultValues(t *testing.T) {

	s := NewSetup(t)
	defer os.Remove(s.DBFile)

	tsLow := int(time.Now().Unix()) * 1000
	if err := s.putJSON(`{"message": "test2", "tags": ["tag1"]}`); err != nil {
		t.Error(err)
	}
	tsHigh := int(time.Now().Unix()) * 1000

	l, err := s.query("tag1", s.TS+1)
	if err != nil || len(l.Posts) != 1 {
		t.Errorf("err: %s or Wrong l.Posts: %#v", err, l.Posts)
	}

	if l.Posts[0].CreatedAt > tsHigh || l.Posts[0].CreatedAt > tsHigh {
		t.Errorf("created_at out of bound, should be %d <= %d <= %d ", tsLow, l.Posts[0].CreatedAt, tsHigh)
	}
}
