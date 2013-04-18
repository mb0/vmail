// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"
	"unicode"
)

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
