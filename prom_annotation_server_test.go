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

	"github.com/prometheus/client_golang/prometheus"
)

/*
  for html coverage report run
  go test -coverprofile=coverage.out  && go tool cover -html=coverage.out
*/

type TestSetup struct {
	T      *testing.T
	Ctx    *ServerContext
	Server *httptest.Server
}

func NewSetup(t *testing.T, storageConfig string) *TestSetup {
	ctx, err := NewServerContext(storageConfig)
	if err != nil {
		log.Fatalf(err.Error())
	}

	server := httptest.NewServer(ctx)
	s := &TestSetup{
		Ctx:    ctx,
		T:      t,
		Server: server,
	}
	return s
}

func (s *TestSetup) put(msg, tag string, ts int) error {
	if ts == 0 {
		ts = int(time.Now().Unix())
	}
	return s.putJSON(fmt.Sprintf(`{"created_at": %d, "message": "%s", "tags": ["%s"]}`, ts, msg, tag), 200)
}

func (s *TestSetup) putJSON(msg string, expectedStatus int) error {
	request, err := http.NewRequest("PUT", s.Server.URL+"/annotations", strings.NewReader(msg))
	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	txt, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != expectedStatus {
		s.T.Errorf("Expected code of %d, not: %d      txt: %s", expectedStatus, res.StatusCode, txt)
		return fmt.Errorf("Expected code of %d, not: %d      txt: %s", expectedStatus, res.StatusCode, txt)
	}
	return err
}

func (s *TestSetup) query(tag string, ts int) (p Posts, err error) {
	queryURL := fmt.Sprintf("%s/annotations?until=%d&range=3600&tags[]=%s", s.Server.URL, ts, tag)
	return s.queryURL(queryURL)
}

func (s *TestSetup) queryURL(queryURL string) (p Posts, err error) {
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

func (s *TestSetup) testAddAndQuery(t *testing.T) {
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

func (s *TestSetup) testDefaultValues(t *testing.T) {
	tsLow := int(time.Now().Unix())
	if err := s.putJSON(`{"message": "test2", "tags": ["defaults"]}`, 200); err != nil {
		t.Error(err)
	}
	tsHigh := int(time.Now().Unix())

	queryURL := fmt.Sprintf("%s/annotations?tags[]=%s", s.Server.URL, "defaults")
	l, err := s.queryURL(queryURL)
	if err != nil || len(l.Posts) != 1 {
		t.Errorf("err: %s or Wrong len(l.Posts): %#v", err, l.Posts)
		return
	}

	if l.Posts[0].CreatedAt > tsHigh*1000 || l.Posts[0].CreatedAt < tsLow*1000 {
		t.Errorf("created_at out of bound, should be %d <= %d <= %d ", tsLow, l.Posts[0].CreatedAt, tsHigh)
	}
}

func (s *TestSetup) testBrokenJSON(t *testing.T) {
	if err := s.putJSON(`{ BROKEN_JSON }`, 500); err != nil {
		t.Error("This shouldn't have failed")
	}
}

func (s *TestSetup) testTagStats(t *testing.T) {
	statsPre, _ := s.Ctx.storage.TagStats()

	if err := s.put("msg1", "tag1", 0); err != nil {
		t.Error(err)
	}
	if err := s.put("msg1", "tag2", 0); err != nil {
		t.Error(err)
	}
	if err := s.put("msg1", "tag2", 0); err != nil {
		t.Error(err)
	}

	statsPost, _ := s.Ctx.storage.TagStats()

	if statsPre["NOT-SET"] != 0 || statsPost["NOT-SET"] != 0 || statsPre["tag1"] != statsPost["tag1"]-1 || statsPre["tag2"] != statsPost["tag2"]-2 {
		t.Errorf("no good, stats counts not as expected")
		return
	}
}

func (s *TestSetup) testMetrics(t *testing.T) {
	if err := s.put("msg1", "tag1", 0); err != nil {
		t.Error(err)
	}

	queryURL := fmt.Sprintf("%s%s", s.Server.URL, *metricsEndpoint)
	res, err := http.Get(queryURL)
	if err != nil {
		s.T.Errorf("err: %s", err)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		s.T.Errorf("err: %s", err)
		return
	}
	if !strings.Contains(string(body), "http_requests_total") {
		t.Errorf("missing http_requests_total from metrics")
	}

	if err := s.put("msg1", "tag2", 0); err != nil {
		t.Error(err)
	}

	res, err = http.Get(queryURL)
	if err != nil {
		s.T.Errorf("err: %s", err)
		return
	}
	body, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		s.T.Errorf("err: %s", err)
		return
	}

	if !strings.Contains(string(body), `http_requests_total{code="200",handler="annotations",method="put"}`) {
		t.Errorf(`missing "http_requests_total{code="200",handler="annotations",method="put"}" from metrics`)
	}

	if !strings.Contains(string(body), `annotations_total{tag="tag2"}`) {
		t.Errorf(`missing "annotations_total{tag="tag2"}" from metrics`)
	}
}

func TestServer(t *testing.T) {
	ts := int(time.Now().Unix())
	storageToTest := []string{
		fmt.Sprintf("local:./test-%d.db", ts),
		fmt.Sprintf("rethinkdb:localhost:28015/annotst%d", ts),
	}

	for _, storage := range storageToTest {
		log.Printf("testing storage: %s", storage)

		s := NewSetup(t, storage)

		s.testAddAndQuery(t)
		s.testDefaultValues(t)
		s.testTagStats(t)
		s.testBrokenJSON(t)
		s.testMetrics(t)

		s.Server.Close()
		s.Ctx.storage.Cleanup()
		prometheus.Unregister(s.Ctx)
	}
}
