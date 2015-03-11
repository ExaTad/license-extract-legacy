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
package main

import (
	"fmt"
	"log"
	"regexp"
)

var rcopyright	= regexp.MustCompile("[ \t\r\n]+[Cc][Oo][Pp][Yy][Rr][Ii][Gg][Hh][Tt][ \t\r\n]+")

var rcomment	= regexp.MustCompile(
			"(/\\*([^*]|[\\r\\n]|(\\*+([^*/]|[\\r\\n])))*\\*+/)|" +			// C Style
			"(([ \\t]*\\.\\\\\"[^\\r\\n]*[\\r\\n])+)|" +				// Troff Style
			"(([ \\t]*//[^\\r\\n]*[\\r\\n])+)|" +					// C++ Style
			"(([ \\t]*#[^\\r\\n]*[\\r\\n])+)|" +					// Shell Style
			"(((dnl[ \\t][^\\r\\n]*[\\r\\n])|(dnl[\\r\\n]))+)")			// Autoconf Style

func main() {
	raw := []byte(`
/* First Copyright Comment 
 first comment line two*/

start_code();
/* Second comment */
more_code(); 
/* Third comment */
end_code();

/****
 * Fourth Common multi-line comment style.
 ****/

/****
 * Fifth Another common multi-line comment style.
 */	

// Sixth bleh1
 // bleh2
// bleh3

# Seventh shell comment line 1
 # Shell comment line 2 with space prefix
  	# Shell comment line 3 with tab prefix

#
# Eigth comment: python / shell comment
#

.\" Access Control Lists manual pages Comment Nine
.\"
.\" (C) 2002 Andreas Gruenbacher, <a.gruenbacher@bestbits.at>
.\"
.\" This is free documentation; you can redistribute it and/or
.\" modify it under the terms of the GNU General Public License as
.\" published by the Free Software Foundation; either version 2 of
.\" the License, or (at your option) any later version.
.\"
.\" The GNU General Public License's references to "object code"
.\" and "executables" are to be interpreted as the output of any
.\" document formatting or typesetting system, including
.\" intermediate and printed output.
.\"
.\" This manual is distributed in the hope that it will be useful,
.\" but WITHOUT ANY WARRANTY; without even the implied warranty of
.\" MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
.\" GNU General Public License for more details.
.\"
.\" You should have received a copy of the GNU General Public
.\" License along with this manual.  If not, see
.\" <http://www.gnu.org/licenses/>.
.\"

dnl Copyright (C) 2003  Silicon Graphics, Inc.	Comment Ten
dnl
dnl This program is free software: you can redistribute it and/or modify it
dnl under the terms of the GNU General Public License as published by
dnl the Free Software Foundation, either version 2 of the License, or
dnl (at your option) any later version.
dnl
dnl This program is distributed in the hope that it will be useful,
dnl but WITHOUT ANY WARRANTY; without even the implied warranty of
dnl MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
dnl GNU General Public License for more details.
dnl
dnl You should have received a copy of the GNU General Public License
dnl along with this program.  If not, see <http://www.gnu.org/licenses/>.

\"\"\" Eleventh comment docstring line 1
    docstring line 2
docstring line 3
	\"\"\"


"""Comment Twelve 1 line docstring #1"""
`)

	ncomments := 12

	matches := rcomment.FindAllIndex(raw, -1)

	if matches == nil {
		log.Fatalf("no comments found expected %d!\n", ncomments)
	}

	for i := 0; i < len(matches); i++ {
		m := matches[i]
		fmt.Printf("Comment %d: %v\n", i+1, m)
		s := string(raw[m[0]:m[1]])
		fmt.Println(s)
	}

	if len(matches) != ncomments {
		log.Fatalf("got %d matches expected %d\n", len(matches), ncomments)
	}

}
