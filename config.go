// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os/user"
	"path/filepath"
	"text/template"

	"github.com/mb0/vmail/feeds"
	"github.com/mb0/vmail/store"
)

type Config struct {
	*user.User
}

func NewConfig(name string) (*Config, error) {
	u, err := user.Lookup(name)
	if err != nil {
		return nil, err
	}
	return &Config{User: u}, nil
}

func (c *Config) Current() error {
	usr, err := user.Current()
	if err != nil {
		return err
	}
	if usr.Username != c.Username {
		return fmt.Errorf("current user is not %s", c.Username)
	}
	return nil
}

func (c *Config) DbFile() string {
	return filepath.Join(c.HomeDir, "vmail.sqlite")
}

func (c *Config) FeedsDir() string {
	return filepath.Join(c.HomeDir, "feeds") + "/"
}

func (c *Config) Fprint(w io.Writer, conf string) error {
	if conf == "sql" {
		var sqls []string
		sqls = append(sqls, store.CreateSql...)
		sqls = append(sqls, feeds.CreateSql...)
		for _, s := range sqls {
			_, err := fmt.Fprintln(w, s)
			if err != nil {
				return err
			}
		}
		return nil
	}
	tmpl := tmpls.Lookup(conf)
	if tmpl == nil {
		return fmt.Errorf("config requires valid option")
	}
	if conf != "" {
		_, err := fmt.Fprintf(w, "# generated with 'vmail config %s'", conf)
		if err != nil {
			return err
		}
	}
	return tmpl.Execute(w, c)
}

var tmpls = template.Must(template.New("").Parse(`vmail config
User: {{ .Username }}
Home: {{ .HomeDir }}
{{define "postfix_domain"}}
dbpath = {{ .HomeDir }}/vmail.sqlite
query = SELECT domain FROM dest WHERE name='%u' AND domain='%d' AND enable=1 AND type=1

{{end}}{{define "postfix_mailbox"}}
dbpath = {{ .HomeDir }}/vmail.sqlite
query = SELECT domain FROM dest WHERE name='%u' AND domain='%d' AND enable=1 AND type=1

{{end}}{{define "postfix_alias"}}
dbpath = {{ .HomeDir }}/vmail.sqlite
query = SELECT forwrd FROM dest WHERE name='%u' AND domain='%d' AND enable=1 AND type=2

{{end}}{{define "dovecot_auth"}}
mail_uid = {{ .Username }}
mail_gid = {{ .Username }}

mail_location = maildir:{{ .HomeDir }}/%u
mail_home = {{ .HomeDir }}/%u/home

userdb {
  driver = static
  args =
}

passdb {
    driver = sql
    args = /etc/dovecot/vmail-sql.conf.ext
}

{{end}}{{define "dovecot_sql"}}
driver = sqlite
connect = {{ .HomeDir }}/vmail.sqlite
default_pass_scheme = SHA512-CRYPT

password_query = \
    SELECT passwd as password, name||'@'||domain as user FROM dest \
    WHERE name = '%n' AND domain = '%d' AND type = 1 AND enable = 1

{{end}}`))
