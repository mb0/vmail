// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"testing"
)

var FixtureSql = `insert into dest
	(id, type, name, domain, enable, passwd, forwrd) values
	(1, 1, 'mbnull', 'mbnull.org', 1, 'xxx', ''),
	(2, 2, 'mb0', 'mb0.org', 1, '', 'mbnull@mbnull.org'),
	(3, 2, 'mb0', 'mbnull.org', 0, '', 'mbnull@mbnull.org')
`

func TestStore(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	err = Create(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(FixtureSql)
	if err != nil {
		t.Fatal(err)
	}
	doms, err := Domains(db, "where enable=1")
	if err != nil {
		t.Fatal(err)
	}
	if len(doms) != 2 || doms[0] != "mb0.org" || doms[1] != "mbnull.org" {
		t.Logf("got unexpected domains %v\n", doms)
	}
	boxes, err := Dests(db, "where enable=1 and type=?", TypeBox)
	if err != nil {
		t.Fatal(err)
	}
	box1 := Dest{1, TypeBox, "mbnull", "mbnull.org", true, "xxx", ""}
	if len(boxes) != 1 || boxes[0] != box1 {
		t.Logf("expect %v got %v\n", box1, boxes)
	}
	aliases, err := Dests(db, "where enable=1 and type=?", TypeAlias)
	if err != nil {
		t.Fatal(err)
	}
	alias1 := Dest{2, TypeAlias, "mb0", "mb0.org", true, "", "mbnull@mbnull.org"}
	if len(aliases) != 1 || boxes[0] != alias1 {
		t.Logf("expect alias %v got %v\n", alias1, aliases)
	}
}
