//
// Copyright © 2014-2015 Exablox Corporation,  All Rights Reserved.
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
package main

import (
	"bufio"
	"fileutils"
	"flag"
	"fmt"
	"licensedb"
	"log"
	"notice"
	"os"
	"path/filepath"
	"runtime"
	"strutils"
	"sync"
	"tagger"
	"version"
)

type FileInfo struct {
	path string
	info os.FileInfo
}

type NoticeMsg struct {
	path   string
	notice *notice.Notice
}

const LicenseDBNumBuckets = 1000000

var ldb *licensedb.LicenseDB
var verbose bool
var ignoreErrors bool
var showLic bool
var quiet bool
var copyrightTagger *tagger.Tagger
var wg sync.WaitGroup
var workerChan chan FileInfo
var noticeChan chan NoticeMsg
var doneChan chan bool

const headHead = "<!DOCTYPE html>\n" +
	"<html>\n" +
	"<head>\n"

const headStyle = "	<meta charset=\"UTF-8\">\n" +
	"	<style>\n" +
	"	.license {\n" +
	"		background-color: #EAFFFF;\n" +
	"		#border-style: solid;\n" +
	"		#border-width: 1px;\n" +
	"	}\n" +
	"	.license-paths {\n" +
	"		background-color: linen;\n" +
	"		#border-width: 1px;\n" +
	"		#border-style: solid;\n" +
	"		#border-color: black;\n" +
	"	}\n" +
	"	.license-path {\n" +
	"		background-color: #FFFFEA;\n" +
	"		#border-width: 1px;\n" +
	"		#border-style: solid;\n" +
	"		#border-color: black;\n" +
	"	}\n" +
	"	.license-text {\n" +
	"		background-color: #EAFFFF;\n" +
	"		margin-left: 5em;\n" +
	"	}\n" +
	"	</style>\n"

const headFooter = "</head>\n" +
	"<body>"

const footer = "</body>\n" +
	"</html>\n"

func ExtraUsage() {
	fmt.Printf("Usage: %s [options] [path] ...\n", os.Args[0])
	fmt.Printf("\n")
	fmt.Printf("Version %s. © 2014-2015 Exablox Corporation.  All Rights Reserved.\n", version.Version)
	fmt.Printf("\n")
	fmt.Printf("Options:\n")
	fmt.Printf("\n")
	flag.PrintDefaults()
	fmt.Printf("\n")
	fmt.Printf("Description:\n")
	fmt.Printf(`
  The purpose of this tool is to generate find and emit the licenses
  and copyright notices in a sourcetree for compliance purposes.
  For example, it can be used to help comply with the various open
  source licenses that require attribution in documentation.

  It recursively searches through the given set of files and
  directories, extracts copyright notices (de-duplicating against
  notices it has already seen), and extracts the licenses.

  At the end of the search, it outputs (-o) an HTML document which
  contains per file copyright notices and licenses.  If the -ldir
  option is given in combination with -o, it copies the licenses
  it finds into the specified directory, and makes the licenses
  viewable / downloadable via a link in the HTML document it emits.

  For directories provided on the command line, all contents will
  be recursively scanned.  For more complex searches (such as
  excluding certin file types), a list of files may also be provided
  via a file or stdin.  This makes it easy to build complex query
  pipelines with tools such as find(1).  See the '-i' and '-0'
  command line options for details.
`)
}

func handleParseError(path string, err error) error {
	if ignoreErrors {
		if !quiet {
			log.Printf("[ERROR] %s: %s\n", path, err)
		}
		return nil
	}

	return err
}

// Starts N workers determined by GOMAXPROCS
// And start a secretary worker
func setupWorkers() {
	var numWorkers int = runtime.GOMAXPROCS(-1)
	workerChan = make(chan FileInfo, numWorkers)
	noticeChan = make(chan NoticeMsg, numWorkers)
	doneChan = make(chan bool)

	// start up the one secretary that will record the notices
	go noticeHandler(noticeChan, doneChan)

	// start up the workers that make notices
	for i := 0; i < numWorkers; i++ {
		go fileHandler(workerChan, noticeChan, doneChan)
	}
}

// The worker that keeps trying to pull work from the worker channel then attempts
// to parse that info to create a notice then if a notice was created send that onto a
// channel to be logged in ldb
func fileHandler(workerChan chan FileInfo, noticeChan chan NoticeMsg, doneChan chan bool) {
	for {
		fileInfo, ok := <-workerChan // recieve the work to do and if work was recieved
		if !ok {                     // ok is set to false when workerChan is closed
			break
		}

		notice := FileParse(fileInfo.path, fileInfo.info)
		if notice != nil {
			noticeChan <- NoticeMsg{path: fileInfo.path, notice: notice}
		}
	}
	doneChan <- true
}

// A Secretart function that attempts to continually take notices and add them to the ldb
// once there are no more notices to take this stops
func noticeHandler(noticeChan chan NoticeMsg, doneChan chan bool) {
	for {
		noticeMsg, ok := <-noticeChan // recieve the notice made and if a notice was recieved
		if !ok {                      // ok is set to false when noticeChan is closed
			break
		}

		ldb.Add(noticeMsg.path, noticeMsg.notice, verbose)
	}
	doneChan <- true
}

// Gathers all the workers ensuring all work is done and all go routines have stopped before allowing
// the program to continue
func shutdownWorkers() {
	var numWorkers int = runtime.GOMAXPROCS(-1)
	for i := 0; i < numWorkers; i++ {
		<-doneChan
	}
	// I have revieved my N workers so no more notices will be comming in
	// close the notice chan so the Notice Handler can stop
	close(noticeChan)
	<-doneChan
	close(doneChan)
}

func sendWork(path string, info os.FileInfo, err error) error {
	if err != nil {
		return handleParseError(path, err)
	}

	workerChan <- FileInfo{path: path, info: info}

	return nil
}

//
// Directories:	filepath.Walk() handles, we'll be called again for each member
// Files:	Read and process
// Others:	Skip
//
func FileParse(path string, info os.FileInfo) *notice.Notice {
	if !info.Mode().IsRegular() {
		if verbose {
			log.Printf("[INFO] Skipping %s (not a file)\n", path)
		}
		return nil
	}

	lic, err := notice.NewNoticeFromFile(path, verbose, showLic, copyrightTagger)
	if err != nil {
		if ignoreErrors {
			if !quiet {
				log.Printf("[ERROR] %s: %s\n", path, err)
			}
		}
		log.Fatal(err)
	}

	return lic
}

// walks all files sending the path to sendWork
func ProcessFile(path string) error {

	err := filepath.Walk(path, sendWork)
	return err
}

//
// Workaround an OSX issue "regexec error 17, (illegal byte sequence)"
//
func fixenv() {
	v := os.Getenv("LANG")
	if v != "" && v != "C" {
		log.Printf("[WARN]: Overriding 'LANG=%s'\n", v)
	}

	err := os.Setenv("LANG", "C")
	if err != nil {
		log.Fatal(err)
	}

	if verbose {
		log.Printf("[WARN]: setenv 'LANG=C'\n")
	}
}

func main() {
	var inPath string
	var stylePath string
	var licenseDir string
	var corpusPath string
	var showVer bool
	var outPath string
	var zeroDelim bool

	flag.Usage = ExtraUsage

	flag.StringVar(&inPath, "i", "", "File to read list of files and directories from (use '-' for stdin)")
	flag.StringVar(&outPath, "o", "", "File to write HTML formatted licensedb to (default = stdout)")
	flag.StringVar(&licenseDir, "ldir", "", "Directory to save licenses to (default = don't save) ")

	flag.StringVar(&stylePath, "style", "", "Use this css stylesheet (default = embed)")
	flag.StringVar(&corpusPath, "corpus", "", "The path the the corpus to use for training the tagger model")

	flag.BoolVar(&zeroDelim, "0", false, "Pathnames read from the input file (-i) are \\0 delimited (default is \\n delimited)")
	flag.BoolVar(&ignoreErrors, "continue", false, "Continue processing, ignoring errors (default is abort on error)")
	flag.BoolVar(&quiet, "quiet", false, "Don't output errors (use in conjunction with '-continue')")
	flag.BoolVar(&verbose, "verbose", false, "Turn on verbose debug output (default is off)")
	flag.BoolVar(&showLic, "showlic", false, "show licenses found during processing")
	flag.BoolVar(&showVer, "version", false, "show version and exit")
	flag.Parse()

	if showVer {
		fmt.Printf("Version %s\n", version.Version)
		return
	}

	fixenv()

	var err error

	// Initialize the Tagger Model
	if corpusPath == "" {
		log.Fatal("corpus required")
		return
	}
	copyrightTagger = tagger.New(corpusPath)

	for _, path := range flag.Args() {
		err = fileutils.PathCheck(path, verbose)
		if err != nil {
			log.Fatal(err)
		}
	}

	ldb = licensedb.NewLicenseDB(licenseDir, LicenseDBNumBuckets, 0)

	setupWorkers()
	for _, path := range flag.Args() {
		err = ProcessFile(path)
		if err != nil {
			log.Fatal(err)
		}
	}
	close(workerChan) //done sending work
	shutdownWorkers()

	if inPath != "" {
		var infile *os.File
		var err error
		if inPath == "-" {
			infile = os.Stdin
		} else {
			infile, err = os.OpenFile(inPath, os.O_RDONLY, 0)
			if err != nil {
				log.Fatal(err)
			}
		}

		scanner := bufio.NewScanner(infile)

		if zeroDelim {
			scanner.Split(strutils.ScanZeros)
		}

		setupWorkers()
		for scanner.Scan() {
			err = ProcessFile(scanner.Text())
			if err != nil {
				log.Fatal(err)
			}
		}
		close(workerChan) //done sending work
		shutdownWorkers()

		err = scanner.Err()
		if err != nil {
			log.Fatal(err)
		}
		infile.Close()
	}

	outfile := os.Stdout
	if outPath != "" {
		outfile, err = os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer outfile.Close()
	}

	outb := bufio.NewWriter(outfile)

	_, err = outb.WriteString(headHead)
	if err != nil {
		goto fail
	}
	if stylePath == "" {
		_, err = outb.WriteString(headStyle)
	} else {
		_, err = fmt.Fprintf(outb, "	<link rel=\"stylesheet\" type=\"text/css\" href=\"%s\">\n", stylePath)
	}
	if err != nil {
		goto fail
	}

	_, err = outb.WriteString(headFooter)
	if err != nil {
		goto fail
	}

	err = ldb.SortedSave(outb, verbose)

	if err != nil {
		goto fail
	}
	_, err = outb.WriteString(footer)
	if err != nil {
		goto fail
	}
	err = outb.Flush()
	if err != nil {
		goto fail
	}
	return

fail:
	outb.Flush()
	log.Fatal(err)
}
