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
package fileutils

import(
	"path/filepath"
	"os"
	"log"
)

//
// filepath.Walk callback to verify the dir, file, or symlink can be read
// Note that we don't want recurse into directories, thus the SkipDir return in certain cases.
//
func fileCheck(path string, info os.FileInfo, err error, verbose bool) error {
	if err != nil {
		return err
	}

	if verbose {
		log.Printf("[INFO] Checking accessibility of %s\n", path)
	}

	if info.Mode().IsDir() || info.Mode().IsRegular() || ((info.Mode() & os.ModeSymlink) == os.ModeSymlink) {
		infile, err := os.OpenFile(path, os.O_RDONLY, 0)
		defer infile.Close()

		if err == nil && info.Mode().IsDir() {
			return filepath.SkipDir
		}

		return err
	}
	return nil
}

func PathCheck(path string, verbose bool) error {
	return filepath.Walk(
			path,
			func(path string, info os.FileInfo, err error) error {
				return fileCheck(path, info, err, verbose)
			})
}
