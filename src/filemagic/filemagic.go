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
package filemagic

import (
	"log"
	"os/exec"
	"regexp"
	"strings"
)

var rbinary = regexp.MustCompile(
	"(^data$)|" +
		"(binary)|" +
		"(PDF)|" +
		"((([Ll][Ii][Tt][Tt][Ll][Ee])|([Bb][Ii][Gg]))[ -]*[Ee][Nn][Dd][Ii][Aa][Nn])|" +
		"(TrueType)|" +
		"(TIFF image)|" +
		"(GIF image)|" +
		"(PostScript document)|" +
		"(ELF 32-bit)|" +
		"(ELF 64-bit)|" +
		"(Mach-O)|" +
		"(PE.*executable)|" +
		"(compiled Java)|" +
		"(Vim swap)|" +
		"(80386 COFF executable)|" +
		"(Berkeley DB)|" +
		"(ACB archive data)|" +
		"(Big-endian)|" +
		"(CDF V2 Document)|" +
		"(lif)")

var runk = regexp.MustCompile(
	"(unknown)|" +
		"(none)")

var rcompressed = regexp.MustCompile(
	"(compressed)|" +
		"(archive)")

type Magic struct {
	Magic []byte
}

func (m *Magic) String() string {
	return string(m.Magic)
}

func (m *Magic) IsUnknown() bool {
	return runk.Match(m.Magic)
}

func (m *Magic) IsCompressed() bool {
	return rcompressed.Match(m.Magic)
}

func (m *Magic) IsBinary() bool {
	return rbinary.Match(m.Magic)
}

func (m *Magic) IsASCII() bool {
	return strings.Contains(m.String(), "ASCII")
}

func New(path string) (*Magic, error) {
	cmd := exec.Command("file", "-b", path)

	magic, err := cmd.Output()
	if err != nil {
		log.Printf("%v: %s\n", cmd.Args, err)
		magic = []byte("unknown")
	} else {
		if magic == nil || len(magic) == 0 {
			log.Printf("%v: empty magic\n", cmd.Args)
			magic = []byte("none")
		}
	}

	if magic[len(magic)-1] == '\n' {
		magic = magic[:len(magic)-1]
	}

	return &Magic{Magic: magic}, nil
}
