// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main


import (
	"fmt"
	"os"
	"strings"

	"antihe.ro/pwhash/sha512_crypt"
	"code.google.com/p/gopass"
	"github.com/mb0/vmail/store"
)

type prog struct {
	conf *Config
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
		db := open(p.conf)
		defer db.Close()
		err = store.Create(db)
		if err != nil {
			fail("could not create tables", err)
		}
	}
	fmt.Println("init successful!")
}

func (p *prog) list(domain string) (res []store.Dest, err error) {
	db := open(p.conf)
	defer db.Close()
	if domain == "" {
		return store.Dests(db, "where enable=1")
	}
	return store.Dests(db, "where enable=1 and domain=?", domain)
}

func readPasswd() (passwd string, err error) {
	for {
		passwd, err = gopass.GetPass("Passwd:")
		if err != nil {
			return "", err	
		}
		repeat, err := gopass.GetPass("Repeat:")
		if err != nil {
			return "", err	
		}
		if passwd == repeat {
			break
		}
		fmt.Println("Inputs don't match. Retry.")
	}
	return sha512_crypt.Crypt(passwd, sha512_crypt.RandomSalt), nil
}

func (p *prog) passwd() error {
	passwd, err := readPasswd()
	if err != nil {
		return err	
	}
	fmt.Println(passwd)
	return nil
}

func (p *prog) create(mailbox, passwd string) error {
	email, err := parseEmail(mailbox, false)
	if err != nil {
		return err
	}
	if email.Name == "" {
		return fmt.Errorf("mailbox name part must not be empty")
	}
	if passwd == "" {
		passwd, err = readPasswd()
		if err != nil {
			return err	
		}
	} else {
		passwd = strings.TrimPrefix(passwd, "{SHA512-CRYPT}")
	}
	if len(passwd) != 4 + 16 + 86 { // $6$[16 chars salt]$[86 chars encrypted]
		return fmt.Errorf("invalid password length %d. create with 'vmail passwd'", len(passwd))
	}
	db := open(p.conf)
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
	db := open(p.conf)
	defer db.Close()
	return store.NewAlias(db, email.Name, email.Domain, forward)
}

func (p *prog) remove(addr string) error {
	email, err := parseEmail(addr, false)
	if err != nil {
		return err
	}
	db := open(p.conf)
	defer db.Close()
	return store.Delete(db, "where name=?, domain=?", email.Name, email.Domain)
}