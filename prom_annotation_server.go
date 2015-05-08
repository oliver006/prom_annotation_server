package main

/*
	to add an annotation:
		curl -XPUT -d '{"message":"build: web server", "tags": ["build"] }'  "localhost:9119/annotations"
*/

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

const VERSION = "0.2"

var (
	/*
		storage config is of format "type:type_specific_config"
		for local storage we use boltdb to store the data and "type_specific_config" sets the name of the DB file
		currently only "local" is available for storage type
		for rethinkdb use this format: rethinkdb:<HOST:PORT>/<DBNAME>
	*/
	storageConfig = flag.String("storage", "local:/tmp/annotations.db", "Storage config, format is \"type:options\". \"local\" is currently the only supported type with options being the location of the DB file.")
	listenAddress = flag.String("listen-addr", ":9119", "Address to listen on for web interface")
	endpoint      = flag.String("endpoint", "/annotations", "Path under which to expose the annotation server")
	showVersion   = flag.Bool("version", false, "Show version information")
)

type ServerContext struct {
	storage Storage
	router  http.Handler
}

func (s *ServerContext) put(c *gin.Context) {
	var a Annotation
	if ok := c.BindWith(&a, binding.JSON); ok && a.Message != "" && len(a.Tags) > 0 {
		if a.CreatedAt == 0 {
			a.CreatedAt = int(time.Now().Unix())
		}

		if err := s.storage.Add(a); err == nil {
			c.JSON(200, map[string]string{"result": "ok"})
			return
		}
	}

	log.Printf("unmarshal annotion error or mad bad data")
	c.JSON(200, map[string]string{"result": "invalid_json"})
	return
}

func (s *ServerContext) get(c *gin.Context) {
	c.Request.ParseForm()

	r, err := strconv.Atoi(c.Request.Form.Get("range"))
	if err != nil || r == 0 {
		r = 3600
	}

	until, err := strconv.Atoi(c.Request.Form.Get("until"))
	if err != nil || until == 0 {
		until = int(time.Now().Unix())
	}
	tags, _ := c.Request.Form["tags[]"]

	list, err := s.storage.Posts(tags, r, until)
	if err != nil {
		c.JSON(200, map[string]string{"result": fmt.Sprintf("err: %s", err)})
		return
	}

	c.JSON(200, list)
}

func NewServerContext(storage string) (*ServerContext, error) {

	st, err := NewStorage(storage)
	if err != nil {
		return nil, err
	}
	srvr := ServerContext{storage: st}

	r := gin.Default()
	r.GET(*endpoint, srvr.get)
	r.PUT(*endpoint, srvr.put)
	srvr.router = r
	return &srvr, nil
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

	s := &http.Server{
		Addr:           *listenAddress,
		Handler:        ctx.router,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Printf("Running server listening at %s, ", *listenAddress)
	go s.ListenAndServe()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Printf("Exiting")
}
