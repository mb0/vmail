// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package email

import (
	"bytes"
	"net/mail"
	"testing"
)

func TestParseDest(t *testing.T) {
	tests := []struct {
		str  string
		addr Addr
	}{
		{
			`"Vmail" <vmail@localhost>`,
			Addr{"Vmail", "vmail@localhost"},
		},
		{
			`vmail@localhost`,
			Addr{"", "vmail@localhost"},
		},
		{
			`@localhost`,
			Addr{"", "@localhost"},
		},
	}
	for _, test := range tests {
		a, err := ParseDest(test.str)
		if err != nil {
			t.Errorf("err %s in %s\n", err, test.str)
			continue
		}
		if a != test.addr {
			t.Errorf("expect %v got %v\n", test.addr, a)
		}
		if a.String() != test.str {
			t.Errorf("expect %v got %v\n", test.str, a.String())
		}
	}
}

func TestMsg(t *testing.T) {
	addrs := []string{`"From" <from@localhost>`, `to@localhost`, `Other <to@localhost>`}
	addr := make([]Addr, len(addrs))
	for i, a := range addrs {
		a, err := ParseDest(a)
		if err != nil {
			t.Error(err)
		}
		addr[i] = a
	}
	m := NewMsg(addr[0], "test subject", addr[1:]...)
	err := m.AddPlain(bytes.NewReader([]byte("test body with Ümlötß\n")))
	if err != nil {
		t.Error(err)
	}
	var buf bytes.Buffer
	m.WriteTo(&buf)
	mm, err := mail.ReadMessage(&buf)
	if err != nil {
		t.Error(err)
	}
	expectHeaders := []struct{ name, expect string }{
		{"From", `"From" <from@localhost>`},
		{"Subject", "=?utf-8?q?test_subject?="},
		{"To", `to@localhost, "Other" <to@localhost>`},
		{"MIME-Version", "1.0"},
		{"Content-Type", `text/plain; charset="utf-8"`},
		{"Content-Transfer-Encoding", "quoted-printable"},
	}
	for _, eh := range expectHeaders {
		if got := mm.Header.Get(eh.name); got != eh.expect {
			t.Errorf("%s: expect %s got %s\n", eh.name, eh.expect, got)
		}
	}
	var body bytes.Buffer
	_, err = body.ReadFrom(mm.Body)
	if err != nil {
		t.Error(err)
	}
	expectBody := "test body with =C3=9Cml=C3=B6t=C3=9F\r\n"
	if got := body.String(); got != expectBody {
		t.Errorf("body: expect %s got %s\n", expectBody, got)
	}
}
