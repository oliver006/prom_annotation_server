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

func (s *TestSetup) testAddAndQuery() {
	ts := int(time.Now().Unix())
	if err := s.put("msg1", "tag1", ts-1); err != nil {
		s.T.Error(err)
	}
	if err := s.put("msg1", "tag1", ts-2); err != nil {
		s.T.Error(err)
	}
	if err := s.put("msg1", "tag2", ts-2); err != nil {
		s.T.Error(err)
	}

	if l, err := s.query("tag1", ts); err != nil || len(l.Posts) != 2 {
		s.T.Errorf("err: %s or Wrong l.Posts: %#v", err, l.Posts)
	}
	if l, err := s.query("tag2", ts); err != nil || len(l.Posts) != 1 {
		s.T.Errorf("err: %s or Wrong l.Posts: %#v", err, l.Posts)
	}
}

func (s *TestSetup) testDefaultValues() {
	tsLow := int(time.Now().Unix())
	if err := s.putJSON(`{"message": "test2", "tags": ["defaults"]}`, 200); err != nil {
		s.T.Error(err)
	}
	tsHigh := int(time.Now().Unix())

	queryURL := fmt.Sprintf("%s/annotations?tags[]=%s", s.Server.URL, "defaults")
	l, err := s.queryURL(queryURL)
	if err != nil || len(l.Posts) != 1 {
		s.T.Errorf("err: %s or Wrong len(l.Posts): %#v", err, l.Posts)
		return
	}

	if l.Posts[0].CreatedAt > tsHigh*1000 || l.Posts[0].CreatedAt < tsLow*1000 {
		s.T.Errorf("created_at out of bound, should be %d <= %d <= %d ", tsLow, l.Posts[0].CreatedAt, tsHigh)
	}
}

func (s *TestSetup) testAll() {
	s.put("msg1", "all1", 0) // defaults to ts=now()
	s.put("msg1", "all2", 0) // defaults to ts=now()
	s.put("msg1", "all3", 0) // defaults to ts=now()
	s.put("msg1", "early", 1)

	queryURL := fmt.Sprintf("%s/annotations?all=true", s.Server.URL)
	l, err := s.queryURL(queryURL)
	if err != nil || len(l.Posts) < 4 {
		s.T.Errorf("err: %s or Wrong len(l.Posts): %#v", err, l.Posts)
		return
	}
}

func (s *TestSetup) testBrokenJSON() {
	if err := s.putJSON(`{ BROKEN_JSON }`, 500); err != nil {
		s.T.Error("This shouldn't have failed")
	}
}

func (s *TestSetup) testTagStats() {
	statsPre, _ := s.Ctx.storage.TagStats()

	if err := s.put("msg1", "tag1", 0); err != nil {
		s.T.Error(err)
	}
	if err := s.put("msg1", "tag2", 0); err != nil {
		s.T.Error(err)
	}
	if err := s.put("msg1", "tag2", 0); err != nil {
		s.T.Error(err)
	}

	statsPost, _ := s.Ctx.storage.TagStats()

	if statsPre["NOT-SET"] != 0 || statsPost["NOT-SET"] != 0 || statsPre["tag1"] != statsPost["tag1"]-1 || statsPre["tag2"] != statsPost["tag2"]-2 {
		s.T.Errorf("no good, stats counts not as expected")
		return
	}
}

func (s *TestSetup) testMetrics() {
	if err := s.put("msg1", "tag1", 0); err != nil {
		s.T.Error(err)
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
		s.T.Errorf("missing http_requests_total from metrics")
	}

	if err := s.put("msg1", "tag2", 0); err != nil {
		s.T.Error(err)
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

	want := fmt.Sprintf(`http_requests_total{code="200",handler="%s",method="put"}`, *annoEndpoint)
	if !strings.Contains(string(body), want) {
		s.T.Errorf(`missing "%s" from metrics`, want)
	}

	if !strings.Contains(string(body), `annotations_total{tag="tag2"}`) {
		s.T.Errorf(`missing "annotations_total{tag="tag2"}" from metrics`)
	}
}

func (s *TestSetup) testAllTags() {
	tagsPre := s.Ctx.storage.AllTags()
	if err := s.put("msg1", "xxxtag1", 0); err != nil {
		s.T.Error(err)
	}
	if err := s.put("msg2", "xxxtag2", 0); err != nil {
		s.T.Error(err)
	}
	if err := s.put("msg3", "xxxtag3", 0); err != nil {
		s.T.Error(err)
	}
	tagsPost := s.Ctx.storage.AllTags()
	if len(tagsPre) != len(tagsPost)-3 {
		s.T.Errorf("no good, tags count not as expected: %#v %#v", tagsPre, tagsPost)
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

		s.testAddAndQuery()
		s.testDefaultValues()
		s.testTagStats()
		s.testBrokenJSON()
		s.testMetrics()
		s.testAllTags()
		s.testAll()

		s.Server.Close()
		s.Ctx.storage.Cleanup()
		prometheus.Unregister(s.Ctx)
	}
}
