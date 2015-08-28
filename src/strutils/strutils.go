//
// Copyright © 2015 Exablox Corporation,  All Rights Reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
//
// 1. Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS
// FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE
// COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
// INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
// BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
// LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN
// ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.
//
package strutils

import (
	"bytes"
	"strings"
)

//
// UTF-8 safe string reversing (with the exception of combining characters)
// from Russ Cox @ https://groups.google.com/forum/#!topic/golang-nuts/oPuBaYJ17t4
// (with local changes)
//
func Reverse(s string) string {
	n := 0
	runes := make([]rune, len(s))
	for _, r := range s {
		runes[n] = r
		n++
	}
	runes = runes[0:n]
	// Reverse
	for i := 0; i < n/2; i++ {
		runes[i], runes[n-1-i] = runes[n-1-i], runes[i]
	}
	// Convert back to UTF-8.
	return string(runes)
}

//
// Guts of Reverse(), operating on Runes instead of strings
//
func ReverseRunes(runes []rune) []rune {
	n := len(runes)
	// Reverse
	for i := 0; i < n/2; i++ {
		runes[i], runes[n-1-i] = runes[n-1-i], runes[i]
	}
	return runes
}

//
// Inserts thousands separators in decimal and floating point numbers
// UTF-8 safe, with the exception of combining characters
//
func PrettyNum(v string) string {
	a := strings.Split(v, ".")

	n := 0
	runes := make([]rune, len(a[0])) // len(a[0]) may be an over-estimate
	for _, r := range a[0] {
		runes[n] = r
		n++
	}
	runes = runes[0:n] // get rid of unused runes

	n = 0
	var out []rune

	for i := len(runes) - 1; i > 0; i-- {
		c := runes[i]

		out = append(out, c)

		if c < '0' || c > '9' {
			n = 0
			continue
		}
		n++

		if n < 3 {
			continue
		}
		out = append(out, rune(','))
		n = 0
	}
	out = append(out, runes[0])

	a[0] = string(ReverseRunes(out))

	return strings.Join(a, ".")
}

//
// ScanZeros is a split function for a Scanner that returns each record of
// text, The returned record may be empty. The end-of-record marker is one
// mandatory \0 (NUL) byte.
//
// The last non-empty line of input will be returned even if it has no
// newline.
//
// This is a hacked up version of ScanLines from the standard library.
// It has been modified to return records by splitting at  \0 (NUL) bytes
// instead of newlines.
//
// The purpose is to handle "find -print0" output in conjunction with the "-i"
// cmdline arg, so that more complex file filtering can be done.
//
func ScanZeros(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\000'); i >= 0 {
		// We have a full \0-terminated record.
		return i + 1, data[0:i], nil
	}

	// If we're at EOF, we have a final, non-terminated record. Return it.
	if atEOF {
		return len(data), data, nil
	}

	// Request more data.
	return 0, nil, nil
}
