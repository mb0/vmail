// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"antihe.ro/pwhash/sha512_crypt"
	"code.google.com/p/gopass"
	"github.com/mb0/vmail/email"
	"github.com/mb0/vmail/feeds"
	"github.com/mb0/vmail/store"
	"github.com/sloonz/go-maildir"
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
	dbpath := p.conf.DbFile()
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
			fail("could not create vmail tables", err)
		}
		err = feeds.Create(db)
		if err != nil {
			fail("could not create feeder tables", err)
		}
	}
	feedsdir := p.conf.FeedsDir()
	_, err = os.Stat(feedsdir)
	if err != nil {
		fmt.Println("creating", feedsdir)
		err = os.Mkdir(feedsdir, 0770)
		if err != nil {
			fail("could not create", feedsdir, err)
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
	e, err := email.ParseAddr(mailbox)
	if err != nil {
		return err
	}
	if e.HasDelimiter('+') {
		fmt.Errorf("mailbox user must not contain '+'")
	}
	if passwd == "" {
		passwd, err = readPasswd()
		if err != nil {
			return err
		}
	} else {
		passwd = strings.TrimPrefix(passwd, "{SHA512-CRYPT}")
	}
	if len(passwd) != 4+16+86 { // $6$[16 chars salt]$[86 chars encrypted]
		return fmt.Errorf("invalid password length %d. create with 'vmail passwd'", len(passwd))
	}
	db := open(p.conf)
	defer db.Close()
	return store.NewBox(db, e.User(), e.Domain(), passwd)
}

func (p *prog) alias(addr, forward string) error {
	e, err := email.ParseDest(addr)
	if err != nil {
		return err
	}
	_, err = email.ParseAddr(forward)
	if err != nil {
		return err
	}
	db := open(p.conf)
	defer db.Close()
	return store.NewAlias(db, e.User(), e.Domain(), forward)
}

func (p *prog) remove(addr string) error {
	e, err := email.ParseDest(addr)
	if err != nil {
		return err
	}
	db := open(p.conf)
	defer db.Close()
	return store.Delete(db, "where name=? and domain=?", e.User(), e.Domain())
}

func (p *prog) feed(name, url string) error {
	db := open(p.conf)
	defer db.Close()
	if name == "" {
		// list feeds
		feeders, err := feeds.Feeders(db, "")
		if err != nil {
			return err
		}
		for _, f := range feeders {
			fmt.Printf("%s\t[%s]\n", f.Name, f.Url)
		}
		return nil
	}
	feeders, err := feeds.Feeders(db, "where name=?", name)
	if err != nil {
		return err
	}
	if len(feeders) > 0 {
		// update feed
		f := feeders[0]
		if f.Url == url {
			fmt.Println("feed %s has already url %s", name, url)
			return nil
		}
		return feeds.UpdateFeeder(db, name, url)
	}
	// create new feed
	_, err = feeds.NewFeeder(db, name, url)
	return err
}

func (p *prog) checkFeed(name string) error {
	feeders, err := p.getFeeders(name)
	if err != nil {
		return err
	}
	if len(feeders) < 1 {
		return fmt.Errorf("no feeder named '%s'", name)
	}
	mdir := &maildir.Maildir{p.conf.FeedsDir()}
	for _, f := range feeders {
		fmt.Printf("check feeder %s\n", f.Name)
		err = p.checkEntries(mdir, f)
		if err != nil {
			return err
		}
	}
	return nil
}
func (p *prog) getFeeders(name string) (fs []feeds.Feeder, err error) {
	db := open(p.conf)
	defer db.Close()
	if name == "*" {
		return feeds.Feeders(db, "where name=?", name)
	}
	return feeds.Feeders(db, "")
}

func (p *prog) checkEntries(mdir *maildir.Maildir, f feeds.Feeder) error {
	// check entries
	entries, err := f.Entries()
	if err != nil {
		return err
	}
	db := open(p.conf)
	defer db.Close()
	entries, err = f.Filter(db, entries)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		fmt.Println("\tno new entries")
		return nil
	}
	sub, err := mdir.Child(f.Name, false)
	if os.IsNotExist(err) {
		sub, err = mdir.Child(f.Name, true)
		if err == nil {
			uid, _ := strconv.Atoi(p.conf.Uid)
			gid, _ := strconv.Atoi(p.conf.Gid)
			os.Chown(sub.Path, uid, gid)
		}
	}
	if err != nil {
		return err
	}
	addr, err := email.ParseAddr(fmt.Sprintf("feeds+%s@mb0.org", f.Name))
	if err != nil {
		return err
	}
	var written []feeds.Entry
	for _, e := range entries {
		m := email.NewMsg(addr, e.Title, addr)
		err = m.AddHtml(strings.NewReader(e.Html()))
		if err != nil {
			fmt.Println(err)
			continue
		}
		var buf bytes.Buffer
		m.WriteTo(&buf)
		_, err := sub.CreateMail(&buf)
		if err != nil {
			fmt.Println(err)
			continue
		}
		written = append(written, e)
	}
	err = f.Fed(db, written, time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("\tgot %d entries %d of them are new\n", len(entries), len(written))
	return f.Prune(db, 256)
}
