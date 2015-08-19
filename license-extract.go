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

type WorkerNotice struct {
	path   string
	notice *notice.Notice
}

const LicenseDBNumBuckets = 1000000

var ldb *licensedb.LicenseDB
var verbose bool
var ignoreErrors bool
var showLic bool
var quiet bool
var periodic bool
var copyrightTagger *tagger.Tagger
var wg sync.WaitGroup
var licenseChan = make(chan WorkerNotice, runtime.NumCPU())
var workerChan = make(chan bool, runtime.NumCPU())
var doneChan = make(chan bool)

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

//
// Directories:	filepath.Walk() handles, we'll be called again for each member
// Files:	Read and process
// Others:	Skip
//
func FileParse(path string, info os.FileInfo, err error) error {
	if err != nil {
		return handleParseError(path, err)
	}

	if !info.Mode().IsRegular() {
		if verbose {
			log.Printf("[INFO] Skipping %s (not a file)\n", path)
		}
		return nil
	}

	workerChan <- true
	go func(path string) {
		lic, err := notice.NewNoticeFromFile(path, verbose, showLic, copyrightTagger)
		if err != nil {
			if ignoreErrors {
				if !quiet {
					log.Printf("[ERROR] %s: %s\n", path, err)
				}
			}
			log.Fatal(err)
		}

		licenseChan <- WorkerNotice{path: path, notice: lic} // throw the license (pointer to Notice) on the channel
		// <-workerChan
	}(path)

	return nil
}

func ProcessFile(path string) error {
	defer close(workerChan)
	go func() {
		for {
			_, more := <-workerChan // recieves the worker so another can start and if no more workers
			if more {
				workerNotice := <-licenseChan // recieves the notice
				ldb.Add(workerNotice.path, workerNotice.notice, verbose)
			} else {
				break
			}
		}
		doneChan <- true
	}()
	return filepath.Walk(path, FileParse)
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
	//	var dbInPath string
	//	var dbOutPath string
	var showVer bool
	var outPath string
	var zeroDelim bool

	flag.Usage = ExtraUsage

	// set GOMaxProcs
	// runtime.GOMAXPROCS(runtime.NumCPU())

	//	flag.StringVar(&dbInPath, "dbload", "", "Load license database from this file (default = don't load)")
	//	flag.StringVar(&dbOutPath, "dbsave", "", "Save license database to this file (default = don't save)")

	flag.StringVar(&inPath, "i", "", "File to read list of files and directories from (use '-' for stdin)")
	flag.StringVar(&outPath, "o", "", "File to write HTML formatted licensedb to (default = stdout)")
	flag.StringVar(&licenseDir, "ldir", "", "Directory to save licenses to (default = don't save) ")

	flag.StringVar(&stylePath, "style", "", "Use this css stylesheet (default = embed)")
	flag.StringVar(&corpusPath, "corpus", "", "The path the the corpus to use for training the tagger model")

	flag.BoolVar(&zeroDelim, "0", false, "Pathnames read from the input file (-i) are \\0 delimited (default is \\n delimited)")
	flag.BoolVar(&ignoreErrors, "continue", false, "Continue processing, ignoring errors (default is abort on error)")
	//	flag.BoolVar(&periodic, "periodic", false, "Periodically output statistics")
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
		fmt.Printf("Required path to input corpus for tagger module!\n")
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

	for _, path := range flag.Args() {
		err = ProcessFile(path)
		<-doneChan // will block the main program untill the go routine adding to ldb is done
		if err != nil {
			log.Fatal(err)
		}
	}

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

		for scanner.Scan() {
			err = ProcessFile(scanner.Text())
			<-doneChan // will block the main program untill the go routine adding to ldb is done
			if err != nil {
				log.Fatal(err)
			}
		}

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

	// wait for all Go Routines to finish as a precaution
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
