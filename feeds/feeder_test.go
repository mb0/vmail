// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package feeds

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
	"time"
)

func TestFeeder(t *testing.T) {
	s := testServer()
	defer s.Close()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	err = Create(db)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewFeeder(db, "xkcd", s.URL)
	if err != nil {
		t.Fatal(err)
	}
	if f.Id != 1 {
		t.Error("id: expect 1 got", f.Id)
	}
	if f.Time != nil {
		t.Error("time: expect time nil got", f.Time)
	}
	original, err := f.Entries()
	if err != nil {
		t.Error(err)
		return
	}
	entries, err := f.Filter(db, original)
	if err != nil {
		t.Error(err)
		return
	}
	if n := len(entries); n != 4 {
		t.Errorf("time: expect 4 entries got %d", n)
	}
	err = f.Fed(db, entries[:2], time.Now())
	if err != nil {
		t.Error(err)
		return
	}
	entries, err = f.Filter(db, entries)
	if err != nil {
		t.Error(err)
		return
	}
	if n := len(entries); n != 2 {
		t.Errorf("time: expect 2 entries got %d", n)
	}
	err = f.Fed(db, entries, time.Now())
	if err != nil {
		t.Error(err)
		return
	}
	var count int
	row := db.QueryRow(`select count(id) from fedentry`)
	if err := row.Scan(&count); err != nil {
		t.Error(err)
		return
	}
	if count != 4 {
		t.Errorf("count 1: expect 4 fedentries got %d", count)
	}
	f.Prune(db, 1)
	row = db.QueryRow(`select count(id) from fedentry`)
	if err := row.Scan(&count); err != nil {
		t.Error(err)
		return
	}
	if count != 1 {
		t.Errorf("count 2: expect 1 fedentries got %d", count)
	}
	var link uint32
	row = db.QueryRow(`select link from fedentry limit 1`)
	if err := row.Scan(&link); err != nil {
		t.Error(err)
		return
	}
	if expect := hashfnv(original[3].Link); link != expect {
		t.Errorf("remaining: expect %x linkhash got %x", expect, link)
	}
}
