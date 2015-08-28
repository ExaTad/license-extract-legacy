#
# Copyright © 2014-2015 Exablox Corporation.  All Rights Reserved.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions
# are met:
#
# 1. Redistributions of source code must retain the above copyright
#    notice, this list of conditions and the following disclaimer.
#
# 2. Redistributions in binary form must reproduce the above copyright
#    notice, this list of conditions and the following disclaimer in the
#    documentation and/or other materials provided with the distribution.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
# "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
# LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS
# FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE
# COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
# INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
# BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
# LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
# CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
# LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN
# ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
# POSSIBILITY OF SUCH DAMAGE.
#

#
# Change this every time you change and release the tool
#
VERSION	= 2.0
PROG	= license-extract
SCRIPTS	= extract.sh getallsrc.sh getsrc.sh mkarchive.sh mknotices.sh style.css overview-notice.html
PKGDEPS = Tagger

all: build install release

DTRACE_FLAGS	= -ldflags -linkmode=external
BUILD_FLAGS	= # ${DTRACE_FLAGS}

buildhost	= ${shell go version | awk '{h=$$NF; sub("/.*", "", h); print h}'}
buildarch	= ${shell go version | awk '{a=$$NF; sub(".*/", "", a); print a}'}

build:	mkversion
	GOOS=darwin GOARCH=amd64 GOPATH=${GOPATH}:$(PWD) go build ${BUILD_FLAGS} -o ${PROG}-${VERSION}-darwin-amd64 ${PROG}.go
	GOOS=linux GOARCH=amd64 GOPATH=${GOPATH}:$(PWD) go build -o ${PROG}-${VERSION}-linux-amd64 ${PROG}.go
	GOOS=linux GOARCH=386 GOPATH=${GOPATH}:$(PWD) go build -o ${PROG}-${VERSION}-linux-386 ${PROG}.go
	GOOS=windows GOARCH=amd64 GOPATH=${GOPATH}:$(PWD) go build -o ${PROG}-${VERSION}-win64.exe ${PROG}.go
	GOOS=windows GOARCH=386 GOPATH=${GOPATH}:$(PWD) go build -o ${PROG}-${VERSION}-win32.exe ${PROG}.go

install: uninstall
	cp ${PROG}-${VERSION}-${buildhost}-${buildarch} ${HOME}/bin

uninstall:
	rm -f ${HOME}/bin/${PROG}-${VERSION}-*

release:
	zip ${PROG}-${VERSION}-darwin.zip ${PROG}-${VERSION}-darwin-amd64
	zip ${PROG}-${VERSION}-linux-amd64.zip ${PROG}-${VERSION}-linux-amd64
	zip ${PROG}-${VERSION}-linux-386.zip ${PROG}-${VERSION}-linux-386
	zip ${PROG}-${VERSION}-win64.zip ${PROG}-${VERSION}-win64.exe
	zip ${PROG}-${VERSION}-win32.zip ${PROG}-${VERSION}-win32.exe

clean:
	rm -rf src/version
	rm -f ${PROG} ${PROG}-*-darwin-* ${PROG}-*-win*.exe ${PROG}-*-linux* ${PROG}-*-*.zip ${PROG} regextest
	rm -f opensrc-${VERSION}.tgz

nuke: clean uninstall

mkversion:
	@mkdir -p src/version
	@echo "package version"			> src/version/version.go
	@echo ""				>> src/version/version.go
	@echo 'const Version = "'${VERSION}'"'	>> src/version/version.go

#
# Run this target to setup the Go Windows and Linux cross compile tools
#
setup-xc:
	sudo GOROOT=${GOROOT} PATH=${PATH}:${GOROOT}/bin ./go-buildcmd
	sudo GOROOT=${GOROOT} PATH=${PATH}:${GOROOT}/bin ./go-buildpkg darwin amd64
	sudo GOROOT=${GOROOT} PATH=${PATH}:${GOROOT}/bin ./go-buildpkg linux amd64
	sudo GOROOT=${GOROOT} PATH=${PATH}:${GOROOT}/bin ./go-buildpkg linux 386
	sudo GOROOT=${GOROOT} PATH=${PATH}:${GOROOT}/bin ./go-buildpkg windows amd64
	sudo GOROOT=${GOROOT} PATH=${PATH}:${GOROOT}/bin ./go-buildpkg windows 386

test: mkversion
	GOPATH=$(PWD) go build ${PROG}.go

rtest:
	GOPATH=$(PWD) go build regextest.go

dist-linux-amd64: all
	rm -rf /tmp/opensrc-${VERSION}
	mkdir /tmp/opensrc-${VERSION}
	cp README /tmp/opensrc-${VERSION}
	cp $(PWD)/src/tagger/CopyrightCorpus.in /tmp/opensrc-${VERSION}
	cp ${SCRIPTS} /tmp/opensrc-${VERSION}
	cd /tmp/opensrc-${VERSION} \
		&& sed "s/EXTRACTVERSION/${VERSION}/g" < mknotices.sh > mknotices.sh.x \
		&& mv mknotices.sh.x mknotices.sh \
		&& chmod 755 mknotices.sh
	cp ${PROG}-${VERSION}-linux-amd64 /tmp/opensrc-${VERSION}
	cd /tmp && tar cvfz opensrc-${VERSION}-linux-amd64.tgz opensrc-${VERSION}
	mv /tmp/opensrc-${VERSION}-linux-amd64.tgz .



deps: ${PKGDEPS}

Tagger_Dir = src/tagger
Tagger_URL = https://github.com/EKnapik/goTagger.git
Tagger_Commit = 9e317e0815bdaf17d2a9f93ee6e380c10c1aa8d8 
Tagger:
	./mk-remote-git.sh ${${@}_URL} ${${@}_Commit} ${${@}_Dir}
