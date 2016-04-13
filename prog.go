// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mb0/vmail/email"
	"github.com/mb0/vmail/feeds"
	"github.com/mb0/vmail/store"
	"github.com/mewbak/gopass"
	"github.com/ncw/pwhash/sha512_crypt"
	maildir "github.com/sloonz/go-maildir"
)

type prog struct {
	conf *Config
}

func ensureFile(name string, mode os.FileMode, content io.Reader) (created bool, err error) {
	_, err = os.Stat(name)
	if err == nil || !os.IsNotExist(err) {
		return
	}
	var f *os.File
	f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE, mode)
	if err != nil {
		return
	}
	created = true
	defer f.Close()
	if content == nil {
		return
	}
	_, err = io.Copy(f, content)
	return
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
	createdb, err := ensureFile(dbpath, 0644, nil)
	if err != nil {
		fail("could not create", dbpath, err)
	}
	if createdb {
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
	if err != nil && !os.IsNotExist(err) {
		fail("could not stat", feedsdir, err)
	}
	if err != nil {
		fmt.Println("creating", feedsdir)
		err = os.Mkdir(feedsdir, 0770)
		if err != nil {
			fail("could not create", feedsdir, err)
		}
	}
	_, err = ensureFile(filepath.Join(feedsdir, "dovecot-shared"), 0660, nil)
	if err != nil {
		fail("could not ensure dovecot-shared", err)
	}
	_, err = ensureFile(filepath.Join(feedsdir, "dovecot-acl"), 0660, strings.NewReader("authenticated lrst\n"))
	if err != nil {
		fail("could not ensure dovecot-acl", err)
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
	for _, f := range feeders {
		fmt.Printf("check feeder %s\n", f.Name)
		err = p.checkEntries(f)
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
		return feeds.Feeders(db, "")
	}
	return feeds.Feeders(db, "where name=?", name)
}

func ensureMaildir(conf *Config, name string) (*maildir.Maildir, error) {
	root := &maildir.Maildir{conf.FeedsDir()}
	child, err := root.Child(name, false)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		child, err = root.Child(name, true)
		if err != nil {
			return nil, err
		}
		uid, _ := strconv.Atoi(conf.Uid)
		gid, _ := strconv.Atoi(conf.Gid)
		err = os.Chown(child.Path, uid, gid)
		if err != nil {
			return nil, err
		}
	}
	_, err = ensureFile(filepath.Join(child.Path, "dovecot-acl"), 0664, strings.NewReader("authenticated lrst\n"))
	if err != nil {
		return nil, err
	}
	return child, nil
}

func (p *prog) checkEntries(f feeds.Feeder) error {
	addr, err := email.ParseAddr(fmt.Sprintf(`"%s" <%s@feeds>`, f.Name, f.Name))
	if err != nil {
		return err
	}
	maildir, err := ensureMaildir(p.conf, f.Name)
	if err != nil {
		return err
	}
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
	filtered := make([]feeds.Entry, 0, len(entries))
	for _, e := range entries {
		if strings.Contains(e.Link, "sportschau.de") {
			continue
		}
		filtered = append(filtered, e)
	}
	entries = filtered
	if len(entries) == 0 {
		fmt.Println("\tno new entries")
		return nil
	}
	var written []feeds.Entry
	for _, e := range entries {
		r, err := e.Html()
		if err != nil {
			log.Println(err)
			continue
		}
		m := email.NewMsg(addr, e.Title, addr)
		dtime, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", e.PubDate)
		if err != nil {
			log.Println(err)
		} else {
			m.Header.Set("Date", dtime.Format(time.RFC822))
		}
		err = m.AddHtml(r)
		if err != nil {
			log.Println(err)
			continue
		}
		var buf bytes.Buffer
		m.WriteTo(&buf)
		_, err = maildir.CreateMail(&buf)
		if err != nil {
			log.Println(err)
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
