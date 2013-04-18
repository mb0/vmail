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
	"strings"
	"unicode"

	"github.com/mb0/vmail/store"
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
		var res []store.Dest
		res, err = p.list(domain)
		if err == nil && len(res) == 0 {
			fmt.Fprintln(os.Stderr, "no results")
		}
		for _, dest := range res {
			fmt.Println(&dest)
		}
	case "create":
		mailbox, passwd := flag.Arg(1), flag.Arg(2)
		err = p.create(mailbox, passwd)
	case "alias":
		email, forward := flag.Arg(1), flag.Arg(2)
		err = p.alias(email, forward)
	case "remove":
		email := flag.Arg(1)
		err = p.remove(email)
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
  create: creates a mailbox
  alias:  creates an alias
  remove: remove an alias or mailbox
  config: prints configuration to stdout
      sql
      postfix_domain
      postfix_mailbox
      postfix_alias
      dovecot_auth
      dovecot_sql
`)
}

type prog struct {
	conf *Config
}

func (p *prog) open() *sql.DB {
	dbfile := p.conf.DB()
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

func (p *prog) setup() {
	err := p.conf.Current()
	if err != nil {
		fail(err, "\nvmail init must be run as user", p.conf.Username)
	}
	_, err = os.Stat(p.conf.HomeDir)
	if err != nil {
		fail(p.conf.HomeDir, "does not exist")
	}
	fmt.Println("init vmail env in", p.conf.HomeDir)
	dbpath := p.conf.DB()
	_, err = os.Stat(dbpath)
	if err != nil {
		fmt.Println("creating", dbpath)
		f, err := os.OpenFile(dbpath, os.O_RDWR|os.O_CREATE, 0660)
		if err != nil {
			fail("could not create", dbpath, err)
		}
		defer f.Close()
		db := p.open()
		defer db.Close()
		err = store.Create(db)
		if err != nil {
			fail("could not create tables", err)
		}
	}
	fmt.Println("init successful!")
}

func (p *prog) list(domain string) (res []store.Dest, err error) {
	db := p.open()
	defer db.Close()
	if domain == "" {
		return store.Dests(db, "where enable=1")
	}
	return store.Dests(db, "where enable=1 and domain=?", domain)
}

type Email struct {
	Addr, Name, Domain string
}

func parseEmail(addr string, allowplus bool) (*Email, error) {
	addr = strings.TrimSpace(addr)
	if strings.IndexFunc(addr, unicode.IsSpace) > 0 {
		return nil, fmt.Errorf("email must not contain spaces")
	}
	res := strings.Split(addr, "@")
	if len(res) != 2 {
		return nil, fmt.Errorf("email must contain one '@'")
	}
	if res[1] == "" {
		return nil, fmt.Errorf("email domain must not be empty")
	}
	if !allowplus && strings.ContainsRune(res[0], '+') {
		return nil, fmt.Errorf("email name must not contain '+'")
	}
	return &Email{Addr: addr, Name: res[0], Domain: res[1]}, nil
}

func (p *prog) create(mailbox, passwd string) error {
	email, err := parseEmail(mailbox, false)
	if err != nil {
		return err
	}
	if email.Name == "" {
		return fmt.Errorf("mailbox name part must not be empty")
	}
	passwd = strings.TrimPrefix(passwd, "{SHA512-CRYPT}")
	if len(passwd) != 106 {
		return fmt.Errorf("invalid password length %d. create with 'doveadm pw -s SHA512-CRYPT'", len(passwd))
	}
	db := p.open()
	defer db.Close()
	return store.NewBox(db, email.Name, email.Domain, passwd)
}

func (p *prog) alias(addr, forward string) error {
	email, err := parseEmail(addr, false)
	if err != nil {
		return err
	}
	_, err = parseEmail(forward, true)
	if err != nil {
		return err
	}
	db := p.open()
	defer db.Close()
	return store.NewAlias(db, email.Name, email.Domain, forward)
}

func (p *prog) remove(addr string) error {
	email, err := parseEmail(addr, false)
	if err != nil {
		return err
	}
	db := p.open()
	defer db.Close()
	return store.Delete(db, "where name=?, domain=?", email.Name, email.Domain)
}