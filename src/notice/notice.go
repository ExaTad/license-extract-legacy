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

package notice

import (
	"crypto/sha1"
	"filemagic"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"tagger"
)

const (
	SRC = iota
	BIN
	UNK
	ERR
)

type Notice struct {
	Sha1 [sha1.Size]byte // Unique identifier for this Notice
	Type int             // Best guess as to the type of object this notice applies to
	Text []byte          // The Notice text itself

	//
	// XXX - Tad: Interface Violation: These are LicenseDB specific things, not Notice specific things
	//
	Count int
	Files []string
	Next  *Notice // next in the database bucket
}

// Define the different comment styles as strings
const cStyle string = "(/\\*([^*]|(\\*+([^*/])))*\\*+/)|(([ \\t]*//[^\r\n]*[\r\n])+)"
const htmlStyle string = "(<!--([^-]|(-*([^->])|->))*-+->)"
const pythonStyle string = "(([ \\t]*#[^\r\n]*[\r\n])+)|(\"\"\"([^\"]|(\"[^\"]|\"\"[^\"]))*\"\"+\")"
const shellStyle string = "(([ \\t]*#[^\r\n]*[\r\n])+)"
const m4Style string = "(([ \\t]*#[^\r\n]*[\r\n])+)|(([ \\t]*dnl[^\r\n]*[\r\n])+)"
const pascalStyle string = "(([ \\t]*//[^\r\n]*[\r\n])+)"
const preproccessStyle string = "(([ \\t]*\\.\\\\\"[^\r\n]*[\r\n])+)|(([ \\t]*\\#[^\r\n]*[\r\n])+)|(([ \\t]*\\\"[^\r\n]*[\r\n])+)"

//
// This regex originated with the "Solution" @  http://blog.ostermiller.org/find-comment
// and has been tweaked and extended.
//
var rcomment = regexp.MustCompile(cStyle + "|" + htmlStyle + "|" + pythonStyle + "|" +
	shellStyle + "|" + m4Style + "|" + pascalStyle + "|" +
	preproccessStyle)

const noNotice = "No copyright notice found"

func mkNotice(path string, ltype int, ltext []byte, showNotice bool) (*Notice, error) {
	if ltext == nil {
		ltext = []byte(noNotice + "\n")
	}

	notice := &Notice{
		Text: ltext,
		Type: ltype,
		Sha1: sha1.Sum(ltext),
	}

	if showNotice {
		log.Printf("[LICENSE %s] Signature %v\n", path, notice.Sha1)
	}

	return notice, nil
}

func extractCopyrightNotices(path string, raw []byte, verbose bool, showNotice bool, copyrightTagger *tagger.Tagger) ([]byte, error) {
	if showNotice {
		log.Printf("[LIC %s]: found copyright outside of comments\n", path)
	}

	cindex := copyrightTagger.FindAllIndex(raw)
	if cindex == nil {
		return nil, fmt.Errorf("%s: matched a copyright but couldn't find it", path)
	}

	var ltext []byte
	for i := 0; i < len(cindex); i++ {
		start := cindex[i][0]
		end := cindex[i][1]

		if showNotice {
			log.Printf("[COPYRIGHT %s] %s\n", path, string(raw[start:end]))
		}

		ltext = append(ltext, raw[start:end]...)
		ltext = append(ltext, '\n')
	}

	return ltext, nil
}

func skipFile(path string) (*filemagic.Magic, int, error) {
	magic, err := filemagic.New(path)

	if err != nil {
		return nil, ERR, err
	}

	if magic.IsCompressed() {
		return magic, BIN, fmt.Errorf("%s is compressed", path)
	}
	if magic.IsBinary() {
		return magic, BIN, fmt.Errorf("%s is binary", path)
	}
	if magic.IsUnknown() {
		return magic, UNK, fmt.Errorf("%s is unknown type", path)
	}

	return nil, SRC, nil
}

//
// Strategy / Heuristics:
//
// 1. If this is an unsupported filetype, return a notice to that effect, including identifying the type of file
//
// 2. Look for a copyright notice, if none was found, return a canonical "unknown copyright" notice.
//
// 3. If a copyright notice was found, but no comments were found, just include the copyright notice.
//
// 4. If a copyright notice was found, create the notice text by including all of the
//    text from all comment blocks which have copyright notices.
//
// 5. As a last resort, if a copyright notice was found, and comments were found, but the copyright
//    notice wasn't found in a comment, just include the copyright notice.
//
func NewNoticeFromFile(path string, verbose bool, showNotice bool, copyrightTagger *tagger.Tagger) (*Notice, error) {

	if verbose {
		log.Printf("[LIC] Process %s\n", path)
	}

	m, ltype, err := skipFile(path)
	if err != nil {
		if m == nil {
			return nil, err
		}
		return mkNotice(path, ltype, append([]byte("Unsupported Filetype: "), m.Magic...), showNotice)
	}

	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Check to see if any copyright notice exists in this file within or not inside of comments
	if !copyrightTagger.Match(raw) {
		if showNotice {
			log.Printf("[LIC %s] %s\n", path, noNotice)
		}
		return mkNotice(path, ltype, nil, showNotice)
	}

	cindex := rcomment.FindAllIndex(raw, -1)
	var ltext []byte

	if cindex == nil {
		ltext, err = extractCopyrightNotices(path, raw, verbose, showNotice, copyrightTagger)
		if err != nil {
			return nil, err
		}
		return mkNotice(path, ltype, ltext, showNotice)
	}

	for i := 0; i < len(cindex); i++ {
		start := cindex[i][0]
		end := cindex[i][1]

		// If the comment does contain something I want I take it if not I skip and go back to the top
		if !copyrightTagger.Match(raw[start:end]) {
			continue
		}

		if showNotice {
			log.Printf("[LICENSE %s] %s\n", path, string(raw[start:end]))
		}

		ltext = append(ltext, raw[start:end]...)
		ltext = append(ltext, '\n')
	}

	if ltext == nil {
		ltext, err = extractCopyrightNotices(path, raw, verbose, showNotice, copyrightTagger)
		if err != nil {
			return nil, err
		}
		return mkNotice(path, ltype, ltext, showNotice)
	}

	return mkNotice(path, ltype, ltext, showNotice)
}
