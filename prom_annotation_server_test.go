package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

/*
  for html coverage report run
  go test -coverprofile=coverage.out  && go tool cover -html=coverage.out
*/

type TestSetup struct {
	ServerURL string
	T         *testing.T
	Ctx       *ServerContext
}

func NewSetup(t *testing.T, storageConfig string) *TestSetup {
	ctx, err := NewServerContext(storageConfig)
	if err != nil {
		log.Fatalf(err.Error())
	}

	server := httptest.NewServer(ctx.router)
	s := &TestSetup{
		Ctx:       ctx,
		T:         t,
		ServerURL: fmt.Sprintf("%s/annotations", server.URL),
	}
	return s
}

func (s *TestSetup) put(msg, tag string, ts int) error {
	return s.putJSON(fmt.Sprintf(`{"created_at": %d, "message": "%s", "tags": ["%s"]}`, ts, msg, tag))
}

func (s *TestSetup) putJSON(msg string) error {
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

func (s *TestSetup) query(tag string, ts int) (p Posts, err error) {
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

func (s *TestSetup) TestServerAddAndQuery(t *testing.T) {

	ts := int(time.Now().Unix())
	if err := s.put("msg1", "tag1", ts-1); err != nil {
		t.Error(err)
	}
	if err := s.put("msg1", "tag1", ts-2); err != nil {
		t.Error(err)
	}
	if err := s.put("msg1", "tag2", ts-2); err != nil {
		t.Error(err)
	}

	if l, err := s.query("tag1", ts); err != nil || len(l.Posts) != 2 {
		t.Errorf("err: %s or Wrong l.Posts: %#v", err, l.Posts)
	}
	if l, err := s.query("tag2", ts); err != nil || len(l.Posts) != 1 {
		t.Errorf("err: %s or Wrong l.Posts: %#v", err, l.Posts)
	}
}

func (s *TestSetup) TestServerDefaultValues(t *testing.T) {

	tsLow := int(time.Now().Unix())
	if err := s.putJSON(`{"message": "test2", "tags": ["ts-test"]}`); err != nil {
		t.Error(err)
	}
	tsHigh := int(time.Now().Unix())

	l, err := s.query("ts-test", tsHigh)
	if err != nil || len(l.Posts) != 1 {
		t.Errorf("err: %s or Wrong len(l.Posts): %#v", err, l.Posts)
		return
	}

	if l.Posts[0].CreatedAt > tsHigh*1000 || l.Posts[0].CreatedAt < tsLow*1000 {
		t.Errorf("created_at out of bound, should be %d <= %d <= %d ", tsLow, l.Posts[0].CreatedAt, tsHigh)
	}
}

func TestServer(t *testing.T) {

	ts := int(time.Now().Unix())
	storageToTest := []string{fmt.Sprintf("local:./test-%d.db", ts), fmt.Sprintf("rethinkdb:localhost:28015/annotst%d", ts)}

	for _, storage := range storageToTest {

		log.Printf("storage: %s", storage)

		s := NewSetup(t, storage)
		defer s.Ctx.storage.Cleanup()

		s.TestServerAddAndQuery(t)
		s.TestServerDefaultValues(t)
	}
}
