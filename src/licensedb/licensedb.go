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
package licensedb

import (
	"bufio"
	"path"
	"os"
	"bytes"
	"fmt"
	"html"
	"io"
	"log"
	"notice"
	"regexp"
	"time"
)

type LicenseDB struct {
	//
	// Content
	//
	Notices    []*notice.Notice // The license / copyright notices extracted from the files
	Licenses   map[string]int   // pathnames of files containing licenses
	LicenseDir string           // Directory to save license files into

	//
	// Statistics
	//
	CreateTime    time.Time // the time this LicenseDB was created (to compute ingest rate stats)
	NumBuckets    int       // Number of buckets in the Notices hash table
	IndexOffset   int       // offset within the SHA1 hash for the bytes to use in the hash function
	NumNotices    uint64    // Total number of Notices held in this db
	NumDupNotices uint64    // Total number of duplicate Notices held in this db
	SearchHist    []uint32  // Histogram of bucket sizes
	MaxSearch     int       // The most hashes that had to be compared to determine a dedup hit
}

//
// Note that the regex is open ended on purpose, to catch things
// like COPYING.GPL, etc
//
var rlicense = regexp.MustCompile(
	"(AUTHORS)|"+
	"(Artistic)|"+
	"(BSD)|"+
	"(CHANGES)|"+
	"(COPYING)|"+
	"(COPYING-CMAKE-SCRIPTS)|"+
	"(COPYING3)|"+
	"(COPYRIGHT)|"+
	"(Copying)|"+
	"(Copyright)|"+
	"(IMPORTING)|"+
	"(LIBGCJ_LICENSE)|"+
	"(LICENSE)|"+
	"(LICENSES)|"+
	"(LICENSE_BSD)|"+
	"(LICENSE_LGPL)|"+
	"(LICENSE_MIT)|"+
	"(License)|"+
	"(NOTICE)|"+
	"(PATENTS)|"+
	"(rcache/RELEASE)|"+
	"(THANKS)|"+
	"(copyright)|"+
	"(copyrights)")

func NewLicenseDB(licensedir string, nbuckets int, indexOffset int) *LicenseDB {
	ldb := &LicenseDB{
		CreateTime:  time.Now(),
		Notices:     make([]*notice.Notice, nbuckets),
		Licenses:    make(map[string]int),
		LicenseDir:  licensedir,
		NumBuckets:  nbuckets,
		IndexOffset: indexOffset,
		SearchHist:  make([]uint32, 1000),
	}

	return ldb
}

func IsLicense(path string) bool {
	return rlicense.MatchString(path)
}

func (ldb *LicenseDB) AddLicense(path string, verbose bool) {
	if verbose {
		log.Printf("[LICENSE] %s is a license\n", path)
	}

	ldb.Licenses[path]++

	if ldb.Licenses[path] > 1 {
		if verbose {
			log.Printf("[LICENSE] %s is a duplicate path\n", path)
		}
	}
}

func (ldb *LicenseDB) Add(path string, n *notice.Notice, verbose bool) {
	if IsLicense(path) {
		ldb.AddLicense(path, verbose)
		return
	}

	offset := ldb.IndexOffset
	index := (int(n.Sha1[offset]) | (int(n.Sha1[offset+1]) << 8) | (int(n.Sha1[offset+2]) << 16) | (int(n.Sha1[offset+3]) << 24)) % ldb.NumBuckets

	var ns int
	var c int
	var v *notice.Notice
	l := &ldb.Notices[index]
	for {
		ns++
		v = *l
		if v == nil {
			break
		}

		c = bytes.Compare(n.Sha1[:], v.Sha1[:])
		if c >= 0 {
			break
		}

		l = &v.Next
	}

	ldb.NumNotices++

	if ns > ldb.MaxSearch {
		ldb.MaxSearch = ns
		if verbose {
			log.Printf("[HIST] MaxSearch %d\n", n)
		}
	}
	if ns >= len(ldb.SearchHist) {
		ns = len(ldb.SearchHist) - 1
	}
	ldb.SearchHist[ns]++

	if v != nil {
		// searching found a record (v != nil) with a hash ≥ the new value
		if c == 0 {
			// dedup hit: the new hash matches exactly a hash previously added
			v.Count++
			ldb.NumDupNotices++
			found := false
			for _, p := range v.Files {
				if path == p {
					found = true
					break
				}
			}

			if !found {
				v.Files = append(v.Files, path)
			}

			if verbose {
				log.Printf("[LDB] %s: Duplicate Notice\n", path)
			}
			return
		}
	}

	// Dedup miss
	//
	// Reached in either of the following cases:
	// 	* exhaused the list
	//	* found a hash > the new one

	if verbose {
		log.Printf("[LDB] %s: New Notice\n", path)
	}

	n.Files = append(n.Files, path)
	n.Next = *l
	*l = n
}

func writeNotice(outb *bufio.Writer, n *notice.Notice, verbose bool) error {
	var err error

	_, err = fmt.Fprintf(outb, "<div class=\"notice\"> <!-- start notice %v -->\n", n.Sha1)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(outb, "<div class=\"notice-paths\"> <!-- start notice paths -->\n")
	if err != nil {
		return err
	}

	for _, path := range n.Files {
		epath := html.EscapeString(path)
		if verbose {
			log.Printf("[OUTPUT] %s\n", epath)
		}

		_, err = fmt.Fprintf(outb, "<div class=\"notice-path\">%s</div>\n", epath)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(outb, "</div> <!-- end notice-paths -->\n")
	if err != nil {
		return err
	}

	ltext := html.EscapeString(string(n.Text))

	_, err = fmt.Fprintf(outb, "<div class=\"notice-text\"> <!-- start notice-text -->\n")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(outb, "<pre>\n")
	if err != nil {
		return err
	}

	_, err = outb.WriteString(ltext)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(outb, "</pre>\n")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(outb, "</div> <!-- end notice-text -->\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(outb, "</div> <!-- end notice %v -->\n", n.Sha1)
	if err != nil {
		return err
	}

	return nil
}

func (ldb *LicenseDB) CopyLicense(src string) error {
	base := path.Base(src)
	dstdir := path.Join(ldb.LicenseDir, path.Dir(src))
	dst := path.Join(dstdir, base)

	err := os.MkdirAll(dstdir, 0755)
	if err != nil {
		return err
	}

	srcf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcf.Close()

	dstf, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstf.Close()

	_, err = io.Copy(dstf, srcf)
	if err != nil {
		return err
	}

	return nil
}

func (ldb *LicenseDB) SaveLicenses(outb *bufio.Writer, verbose bool) error {
	const start =	"<div class=\"licenses\"> <!-- start licenses -->\n"+
			"	<div class=\"license-paths\"> <!-- start license paths -->\n"+
			"		<h2>Licenses</h2>\n"+
			"		<table>\n"

	const end =	"		</table>\n"+
			"	</div> <!-- end license paths -->\n"+
			"</div> <!-- end licenses -->\n"

	_, err := outb.WriteString(start)
	if err != nil {
		return err
	}

	n := 1
	for path, _ := range ldb.Licenses {
		if ldb.LicenseDir == "" {
			_, err = fmt.Fprintf(outb, "			<tr><td>%d</td><td>%s</td></tr>\n", n, path)
		} else {
			_, err = fmt.Fprintf(outb, "			<tr><td>%d</td><td><a href=\"%s\">%s</a></td></tr>\n", n, path, path)
			if err != nil {
				return err
			}

			err = ldb.CopyLicense(path)
		}
		if err != nil {
			return err
		}

		n++
	}

	_, err = outb.WriteString(end)
	if err != nil {
		return err
	}

	return nil
}

func (ldb *LicenseDB) SaveNotices(outb *bufio.Writer, verbose bool) error {
	const start =	"<div class=\"notices\"> <!-- start notices -->\n"+
			"	<h2>Notices</h2>\n"

	const end =	"</div> <!-- end notices -->\n"

	_, err := outb.WriteString(start)
	if err != nil {
		return err
	}

	dosep := false
	for i := 0; i < len(ldb.Notices); i++ {
		for n := ldb.Notices[i]; n != nil; n = n.Next {
			if dosep {
				_, err = fmt.Fprintf(outb, "<hr>\n")
				if err != nil {
					return err
				}
			}
			err = writeNotice(outb, n, verbose)
			if err != nil {
				return err
			}
			dosep = true
		}
	}

	_, err = outb.WriteString(end)
	if err != nil {
		return err
	}

	return nil
}

func (ldb *LicenseDB) Save(outb *bufio.Writer, verbose bool) error {
	err := ldb.SaveLicenses(outb, verbose)
	if err != nil {
		return err
	}

	err = ldb.SaveNotices(outb, verbose)
	if err != nil {
		return err
	}

	return nil
}
