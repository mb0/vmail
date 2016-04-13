// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package email

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"

	"mime/quotedprintable"
)

var NoAddr = Addr{}
var Sender = Addr{"vmail", "vmail@localhost"}

type Addr mail.Address

func ParseAddr(addr string) (Addr, error) {
	addr = strings.TrimSpace(addr)
	maddr, err := mail.ParseAddress(addr)
	if err != nil {
		return NoAddr, err
	}
	return (Addr)(*maddr), nil
}

func ParseDest(addr string) (Addr, error) {
	addr = strings.TrimSpace(addr)
	catchall := len(addr) > 0 && addr[0] == '@'
	if catchall {
		addr = "~" + addr
	}
	a, err := ParseAddr(addr)
	if catchall && err == nil {
		a.Address = a.Address[1:]
	}
	return a, err
}

func (a Addr) String() string {
	if a.Name == "" {
		return a.Address
	}
	return (*mail.Address)(&a).String()
}

func (a Addr) User() string {
	split := strings.Split(a.Address, "@")
	if len(split) != 2 {
		return ""
	}
	return split[0]
}

func (a Addr) Domain() string {
	split := strings.Split(a.Address, "@")
	if len(split) != 2 {
		return ""
	}
	return split[1]
}

func (a Addr) HasDelimiter(d rune) bool {
	return strings.ContainsRune(a.User(), d)
}

// isQtext returns true if c is an RFC 5322 qtest character.
func isQtext(c byte) bool {
	// Printable US-ASCII, excluding backslash or quote.
	if c == '\\' || c == '"' {
		return false
	}
	return '!' <= c && c <= '~'
}

// isVchar returns true if c is an RFC 5322 VCHAR character.
func isVchar(c byte) bool {
	// Visible (printing) characters.
	return '!' <= c && c <= '~'
}

// rfc2047 Message Header Extensions for Non-ASCII Text
func mimestr(s string) string {
	ascii := true
	for i := 0; i < len(s); i++ {
		if !isVchar(s[i]) {
			ascii = false
			break
		}
	}
	var b bytes.Buffer
	if ascii {
		for i := 0; i < len(s); i++ {
			if !isQtext(s[i]) {
				b.WriteByte('\\')
			}
			b.WriteByte(s[i])
		}
	} else {
		b.WriteString("=?utf-8?q?")
		for i := 0; i < len(s); i++ {
			switch c := s[i]; {
			case c == ' ':
				b.WriteByte('_')
			case isVchar(c) && c != '=' && c != '?' && c != '_':
				b.WriteByte(c)
			default:
				fmt.Fprintf(&b, "=%02X", c)
			}
		}
		b.WriteString("?= ")
	}
	return b.String()
}

type Part struct {
	Header  textproto.MIMEHeader
	Content io.Reader
}

func NewPart(typ, enc string, r io.Reader) *Part {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", typ)
	h.Set("Content-Transfer-Encoding", enc)
	return &Part{h, r}
}

type Msg struct {
	Header textproto.MIMEHeader
	Parts  []Part
}

func NewMsg(from Addr, subject string, to ...Addr) *Msg {
	h := make(textproto.MIMEHeader)
	h.Set("From", from.String())
	list := make([]string, 0, len(to))
	for _, addr := range to {
		list = append(list, addr.String())
	}
	h.Set("To", strings.Join(list, ", "))
	h.Set("Subject", mimestr(subject))
	return &Msg{Header: h}
}

func (m *Msg) WriteTo(w io.Writer) error {
	if len(m.Parts) == 0 {
		return fmt.Errorf("no message content")
	}
	if len(m.Parts) > 1 {
		return m.writeMultipart(w)
	}
	p := m.Parts[0]
	for k, vv := range p.Header {
		m.Header[k] = vv
	}
	if m.Header.Get("Content-Type") == "" {
		m.Header.Set("Content-Type", `text/plain; charset="utf-8"`)
	}
	err := m.writeHeader(w)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, p.Content)
	return err
}

func (m *Msg) writeHeader(w io.Writer) error {
	var err error
	m.Header.Set("MIME-Version", `1.0`)
	for k, vv := range m.Header {
		for _, v := range vv {
			_, err = fmt.Fprintf(w, "%s: %s\r\n", k, v)
			if err != nil {
				return err
			}
		}
	}
	_, err = fmt.Fprintf(w, "\r\n")
	return err
}

func (m *Msg) writeMultipart(w io.Writer) error {
	mw := multipart.NewWriter(w)
	defer mw.Close()
	m.Header.Set("Content-Type", `multipart/mixed; boundary=`+mw.Boundary())
	err := m.writeHeader(w)
	if err != nil {
		return err
	}
	for _, p := range m.Parts {
		pw, err := mw.CreatePart(p.Header)
		if err != nil {
			return err
		}
		_, err = io.Copy(pw, p.Content)
		if err != nil {
			return err
		}
	}
	return nil
}
func (m *Msg) AddQuotedPrintable(typ string, r io.Reader) error {
	buf := &bytes.Buffer{}
	enc := quotedprintable.NewWriter(buf)
	_, err := io.Copy(enc, r)
	if err != nil {
		return err
	}
	p := NewPart(typ, `quoted-printable`, buf)
	m.Parts = append(m.Parts, *p)
	return nil
}

func (m *Msg) AddPlain(r io.Reader) error {
	return m.AddQuotedPrintable(`text/plain; charset="utf-8"`, r)
}

func (m *Msg) AddHtml(r io.Reader) error {
	return m.AddQuotedPrintable(`text/html; charset="utf-8"`, r)
}
