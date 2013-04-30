// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

var username = flag.String("user", "vmail", "vmail username")

func main() {
	flag.Usage = usage
	flag.Parse()
	conf, err := NewConfig(*username)
	if err != nil {
		fail(err)
	}
	p := &prog{conf}
	switch flag.Arg(0) {
	case "setup":
		p.setup()
	case "config":
		err = p.conf.Fprint(os.Stdout, flag.Arg(1))
	case "list":
		domain := flag.Arg(1)
		res, err1 := p.list(domain)
		if err = err1; err == nil && len(res) == 0 {
			fmt.Fprintln(os.Stderr, "no results")
		}
		for _, dest := range res {
			fmt.Println(&dest)
		}
	case "passwd":
		err = p.passwd()
	case "create":
		mailbox, passwd := flag.Arg(1), flag.Arg(2)
		err = p.create(mailbox, passwd)
	case "alias":
		email, forward := flag.Arg(1), flag.Arg(2)
		err = p.alias(email, forward)
	case "remove":
		email := flag.Arg(1)
		err = p.remove(email)
	case "feed":
		name, url := flag.Arg(1), flag.Arg(2)
		err = p.feed(name, url)
	case "checkfeed":
		name := flag.Arg(1)
		err = p.checkFeed(name)
	default:
		failUsage("unknown command: ", flag.Arg(0))
	}
	if err != nil {
		fail(err)
	}
}

func fail(msgs ...interface{}) {
	fmt.Fprintln(os.Stderr, msgs...)
	os.Exit(1)
}

func failUsage(msgs ...interface{}) {
	fmt.Fprintln(os.Stderr, msgs...)
	usage()
	os.Exit(2)
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage of vmail:
	vmail --flag command [...opts]
flag:
`)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
command:
  setup:  initializes vmail setup
  list:   lists known destinations
  passwd: SHA512-CRYPTs input
  create: creates a mailbox
  alias:  creates an alias
  remove: removes an alias or mailbox
  config: prints configuration to stdout
      sql
      postfix_domain
      postfix_mailbox
      postfix_alias
      dovecot_auth
      dovecot_sql
`)
}

func open(conf *Config) *sql.DB {
	dbfile := conf.DbFile()
	_, err := os.Stat(dbfile)
	if err != nil {
		fail(dbfile, "does not exist")
	}
	db, err := sql.Open("sqlite3", dbfile)
	if err != nil {
		fail("could not connect to", dbfile, err)
	}
	return db
}
