// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	"database/sql"
	"fmt"
)

const (
	_ = iota
	TypeBox
	TypeAlias
)

type Dest struct {
	Id     uint64
	Type   int
	Name   string
	Domain string
	Enable bool
	Passwd string
	Forwrd string
}

func (d *Dest) String() string {
	var typ string
	if d.Type == TypeAlias {
		typ = " -> " + d.Forwrd
	}
	return fmt.Sprintf("%s@%s %s", d.Name, d.Domain, typ)
}

var CreateSql = `create table dest (
	id integer primary key autoincrement,
	type integer,
	name text,
	domain text,
	enable integer default 1,
	passwd text,
	forwrd text,
	unique (name, domain)
)`

var DomainsSql = `select
	distinct domain
	from dest %s order by domain
`

var DestsSql = `select
	id, type, name, domain, enable, passwd, forwrd
	from dest %s order by domain, name
`
var InsertSql = `insert into dest
	(type, name, domain, passwd, forwrd)
	values (?, ?, ?, ?, ?)
`
var DeleteSql = `delete from dest`

func Create(db *sql.DB) error {
	_, err := db.Exec(CreateSql)
	return err
}

func Domains(db *sql.DB, where string, args ...interface{}) ([]string, error) {
	rows, err := db.Query(fmt.Sprintf(DomainsSql, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []string
	for rows.Next() {
		var domain string
		err = rows.Scan(&domain)
		if err != nil {
			return nil, err
		}
		res = append(res, domain)
	}
	return res, rows.Err()
}

func Dests(db *sql.DB, where string, args ...interface{}) ([]Dest, error) {
	rows, err := db.Query(fmt.Sprintf(DestsSql, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []Dest
	for rows.Next() {
		var dest Dest
		err = rows.Scan(&dest.Id, &dest.Type, &dest.Name, &dest.Domain, &dest.Enable, &dest.Passwd, &dest.Forwrd)
		if err != nil {
			return nil, err
		}
		res = append(res, dest)
	}
	return res, rows.Err()
}

func NewBox(db *sql.DB, name, domain, passwd string) error {
	_, err := db.Exec(InsertSql, TypeBox, name, domain, passwd, "")
	return err
}

func NewAlias(db *sql.DB, name, domain, forwrd string) error {
	_, err := db.Exec(InsertSql, TypeAlias, name, domain, "", forwrd)
	return err
}

func Delete(db *sql.DB, where string, args ...interface{}) error {
	_, err := db.Exec(fmt.Sprintf(DeleteSql, where), args...)
	return err
}