package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	r "github.com/dancannon/gorethink"
)

type RethinkDBStorage struct {
	dbName  string
	session *r.Session
}

func NewRethinkDBStorage(conn string) (*RethinkDBStorage, error) {
	parts := strings.Split(conn, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid rethinkdb connection string: %s  expected format: <host:port>/<dbname>", conn)
	}

	addr := parts[0]
	db := parts[1]

	s, err := r.Connect(r.ConnectOpts{
		Addresses: []string{addr},
	})

	if err != nil {
		return nil, err
	}
	r.DbCreate(db).Run(s)
	r.Db(db).TableCreate("annotations").Run(s)
	r.Db(db).Table("annotations").IndexCreate("created_at").Run(s)

	s.Use(db)

	return &RethinkDBStorage{session: s, dbName: db}, nil
}

func (s *RethinkDBStorage) AllTags() (res []string) {

	q, _ := r.Table("annotations").Pluck("tags").Distinct().Run(s.session)
	var rows []struct {
		Tags []string
	}
	err := q.All(&rows)
	if err != nil {
		return []string{}
	}

	// collect all tags first
	temp := make(map[string]bool)
	for _, row := range rows {
		for _, s := range row.Tags {
			temp[s] = true
		}
	}

	res = make([]string, 0)
	for tag := range temp {
		res = append(res, tag)
	}
	return res
}

func (s *RethinkDBStorage) TagStats() (TagStats, error) {
	var res TagStats = make(map[string]int)

	q, err := r.Table("annotations").Pluck("tags").Distinct().Run(s.session)
	if err != nil {
		return res, err
	}
	var rows []struct {
		Tags []string
	}
	err = q.All(&rows)
	if err != nil {
		return res, err
	}

	// collect all tags first
	for _, row := range rows {
		for _, s := range row.Tags {
			res[s] = 0
		}
	}

	// now collect the tag count for each tag
	for tag := range res {
		res[tag] = s.GetCount(tag)
	}

	return res, nil
}

func (s *RethinkDBStorage) Add(a Annotation) error {
	_, err := r.Table("annotations").Insert(a).RunWrite(s.session)
	if err != nil {
		log.Printf("Saving annotation failed, err: %s", err)
	}
	return err
}

func (s *RethinkDBStorage) ListForTag(tag string, ra, until int, out *[]Annotation) (err error) {
	start := float64(until-ra) - 0.5
	end := float64(until) + 0.5

	res, err := r.Table("annotations").Between(start, end, r.BetweenOpts{Index: "created_at", RightBound: "open", LeftBound: "open"}).Filter(func(row r.Term) r.Term {
		return row.Field("tags").Contains(tag)
	}).Run(s.session)

	if err != nil {
		log.Printf("err geting annotations for tag %s err: %s", tag, err)
		return err
	}
	defer res.Close()

	var a Annotation
	for res.Next(&a) {
		*out = append(*out, Annotation{CreatedAt: a.CreatedAt * 1000, Message: a.Message, Tags: []string{tag}})
	}
	return err
}

func (s *RethinkDBStorage) Close() {
	s.session.Close()
	log.Printf("Closed RethinkDB storage")
}

func (s *RethinkDBStorage) Cleanup() {
	r.DbDrop(s.dbName).RunWrite(s.session)
	s.Close()
}

func (s *RethinkDBStorage) GetCount(tag string) (count int) {
	var temp []Annotation
	ts := int(time.Now().Unix())
	s.ListForTag(tag, ts, ts, &temp)
	return len(temp)
}
