// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package feeds

import (
	"database/sql"
	"hash/fnv"
	"time"
)

const (
	_ = iota
	TypeRss
)

var CreateSql = []string{
	`create table if not exists feeder (
	id integer primary key autoincrement,
	type integer,
	name text unique,
	url  text unique,
	time timestamp
)`,
	`create table if not exists fedentry (
	id integer primary key autoincrement,
	feeder integer,
	link   integer,
	title  integer,
	time   timestamp,
	unique (feeder, link),
	unique (feeder, title)
)`}

func Create(db *sql.DB) error {
	for _, s := range CreateSql {
		_, err := db.Exec(s)
		if err != nil {
			return err
		}
	}
	return nil
}

type Feeder struct {
	Id   int64
	Type int
	Name string
	Url  string
	Time *time.Time
}

type FedEntry struct {
	Id    int64
	Feed  int64
	Link  uint32
	Title uint32
	Time  time.Time
}

func Feeders(db *sql.DB, where string, args ...interface{}) ([]Feeder, error) {
	rows, err := db.Query(`select * from feeder `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var fs []Feeder
	for rows.Next() {
		var f Feeder
		err = rows.Scan(&f.Id, &f.Type, &f.Name, &f.Url, &f.Time)
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}
	return fs, nil
}
func UpdateFeeder(db *sql.DB, name, url string) error {
	_, err := db.Exec(`update feeder set url=? where name=?`, url, name)
	return err
}
func NewFeeder(db *sql.DB, name, url string) (*Feeder, error) {
	r, err := db.Exec(`insert into feeder (type, name, url) values (?, ?, ?)`, TypeRss, name, url)
	if err != nil {
		return nil, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Feeder{id, TypeRss, name, url, nil}, nil
}

func (f *Feeder) Entries() ([]Entry, error) {
	feed, err := ReadHttp(f.Url)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(feed.Channel.Item))
	for _, item := range feed.Channel.Item {
		entries = append(entries, Entry{item})
	}
	return entries, nil
}

func hashfnv(str string) uint32 {
	h := fnv.New32()
	h.Write([]byte(str))
	return h.Sum32()
}

func (f *Feeder) Filter(db *sql.DB, entries []Entry) ([]Entry, error) {
	res := make([]Entry, 0, len(entries))
next:
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		for _, o := range res {
			if e.Title == o.Title || e.Link == o.Link {
				continue next
			}
		}
		res = append(res, e)
	}
	entries, res = res, make([]Entry, 0, len(res))
	stmt, err := db.Prepare(`select 1 from fedentry where feeder=? and (link=? or title=?) limit 1`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		// filter seen
		row := stmt.QueryRow(f.Id, hashfnv(e.Link), hashfnv(e.Title))
		if err := row.Scan(new(bool)); err == sql.ErrNoRows {
			res = append(res, e)
		} else if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (f *Feeder) Fed(db *sql.DB, entries []Entry, now time.Time) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	f.Time = &now
	_, err = tx.Exec("update feeder set time=? where id=?", f.Time, f.Id)
	stmt, err := tx.Prepare(`insert or ignore into fedentry (feeder, link, title, time) values (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, e := range entries {
		_, err := stmt.Exec(f.Id, hashfnv(e.Link), hashfnv(e.Title), now)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (f *Feeder) Prune(db *sql.DB, limit int) error {
	_, err := db.Exec(`delete from fedentry where feeder=? and id not in (
		select id from fedentry where feeder=? order by time, id desc limit ?
	)`, f.Id, f.Id, limit)
	return err
}
