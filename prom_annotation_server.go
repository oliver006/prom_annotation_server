package main

/*
	to add an annotation:
		curl -XPUT -d '{"message":"build: web server", "tags": ["build"] }'  "localhost:9119/annotations"
*/

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const VERSION = "0.4"

var (
	/*
		storage config is of format "type:type_specific_config"
		for local storage we use boltdb to store the data and "type_specific_config" sets the name of the DB file
		currently only "local" is available for storage type
		for rethinkdb use this format: rethinkdb:<HOST:PORT>/<DBNAME>
	*/
	storageConfig   = flag.String("storage", "local:/tmp/annotations.db", "Storage config, format is \"type:options\". \"local\" and \"rethinkdb\" are currently the only supported types.")
	listenAddress   = flag.String("listen-addr", ":9119", "Address to listen on for web interface")
	annoEndpoint    = flag.String("endpoint", "/annotations", "Path under which to expose the annotation server")
	metricsEndpoint = flag.String("metris", "/metrics", "Path under which to expose the metrics of the annotation server")
	showVersion     = flag.Bool("version", false, "Show version information")
)

type ServerContext struct {
	storage         Storage
	annotationStats *prometheus.GaugeVec
}

func newAnnotationStats() *prometheus.GaugeVec {

	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "annotations_total",
		Help: "Number of annotations per tag.",
	}, []string{"tag"})
}

func NewServerContext(storage string) (*ServerContext, error) {

	st, err := NewStorage(storage)
	if err != nil {
		return nil, err
	}
	srvr := ServerContext{
		storage:         st,
		annotationStats: newAnnotationStats(),
	}
	prometheus.MustRegister(&srvr)
	return &srvr, nil
}

func (s *ServerContext) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	log.Printf("Request: %s  %s", req.Method, req.URL.Path)

	switch req.URL.Path {
	case *metricsEndpoint:
		prometheus.Handler().ServeHTTP(w, req)
	case *annoEndpoint:
		prometheus.InstrumentHandlerFunc(*annoEndpoint, s.annotations)(w, req)
	default:
		http.Error(w, "Not found", 404)
	}
}

func (s *ServerContext) Describe(ch chan<- *prometheus.Desc) {
	s.annotationStats.Describe(ch)
}

func (s *ServerContext) Collect(ch chan<- prometheus.Metric) {
	s.annotationStats = newAnnotationStats()
	defer s.annotationStats.Collect(ch)

	stats, err := s.storage.TagStats()
	if err != nil {
		log.Printf("stats err: %s", err)
		return
	}

	for tag, count := range stats {
		s.annotationStats.WithLabelValues(tag).Set(float64(count))
	}
}

func writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	temp, _ := json.Marshal(data)
	fmt.Fprintln(w, string(temp))
}

func (s *ServerContext) annotations(w http.ResponseWriter, req *http.Request) {

	switch req.Method {

	case "GET":
		s.get(w, req)

	case "PUT":
		s.put(w, req)

	default:
		http.Error(w, "Not supported", 405)
	}
}

func (s *ServerContext) put(w http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()
	body, _ := ioutil.ReadAll(req.Body)
	var a Annotation
	if err := json.Unmarshal(body, &a); err == nil {
		if a.CreatedAt == 0 {
			a.CreatedAt = int(time.Now().Unix())
		}

		if err := s.storage.Add(a); err == nil {
			writeJSON(w, 200, map[string]string{"result": "ok"})
			return
		}
	}

	log.Printf("unmarshal annotion error or mad bad data: %s", body)
	writeJSON(w, 500, map[string]string{"result": "invalid_json"})
}

func (s *ServerContext) get(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	var err error
	var tags []string
	var r, until int

	all := req.Form.Get("all")
	if all != "" {
		tags = s.storage.AllTags()
		r = int(time.Now().Unix())
	} else {
		r, _ = strconv.Atoi(req.Form.Get("range"))
		if r == 0 {
			r = 3600
		}
		until, _ = strconv.Atoi(req.Form.Get("until"))
		tags, _ = req.Form["tags[]"]
	}
	if until == 0 {
		until = int(time.Now().Unix())
	}
	list, err := GetPosts(s.storage, tags, r, until)
	if err != nil {
		writeJSON(w, 500, map[string]string{"result": fmt.Sprintf("err: %s", err)})
		return
	}

	writeJSON(w, 200, list)
}

func main() {
	flag.Parse()
	fmt.Printf("prom_annotation_server version %s \n", VERSION)
	if *showVersion {
		return
	}

	ctx, err := NewServerContext(*storageConfig)
	if err != nil {
		log.Fatalf("storage config borked, err: %s", err)
	}
	defer ctx.storage.Close()

	http.Handle("/", ctx)

	log.Printf("Running server listening at %s, ", *listenAddress)
	go http.ListenAndServe(*listenAddress, nil)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Printf("Exiting")
}
