#!/bin/bash

#
# Copyright © 2015 Exablox Corporation,  All Rights Reserved.
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

set -o nounset
set -o errexit

#
# ${dstdir} holds the path to the directory to place all output into.
# The default is normally OK
#
dstdir="/mnt/opensrc"

#
# ${sshport} is the "-p {portnum}" argument to sshfs, if specified on the cmdline
# or empty if not.
#
sshport=""

#
# ${sshinfo} holds the {user}@{host}[:/path] info for the remote system.  If possible,
# This remote path is mounted onto ${dstdir}
#
sshinfo=""

#
# ${rdeps} holds the list of pkgs this toolchain depends on.  Update this
# whenever the toolchain evolves to need anything new
#
rdeps="dpkg-dev file sshfs"

#
# ${ideps} contains the list of packages that were installed in order to
# to make all of the ${rdeps} available.
#
ideps=""

#
# ${cleanup_dstdir} is a boolean that indicates whether the dstdir
# should be recursively removed
#
cleanup_dstdir="no"

#
# ${remount_rw} is a boolean that indicates whether the root partition
# needs to be remounted readwrite in order to install the necessary packages
#
remount_rw="no"

deps_install()
{
	local pkgs=""
	local pkg=""

	for pkg in ${rdeps} ; do
		pkgs=$(apt-get -s install ${pkg} | awk '/^Inst / {printf("%s ", $2)} {next}')
		ideps="${ideps}${pkgs}"
	done

	if [ -z "${ideps}" ] ; then
		echo "Installing: nothing"
		return
	fi

	if mount | grep 'on / .*(ro' > /dev/null 2>/dev/null ; then
		remount_rw="yes"
		mount -o remount,rw /
	fi

	echo "Installing: ${ideps}"
	apt-get install ${ideps}
}

deps_uninstall()
{
	if [ -z "${ideps}" ] ; then
		echo "Removing: nothing"
		return
	fi

	echo "Removing: ${ideps}"
	apt-get remove ${ideps}
}

#
# pkg_skip returns 0 if the package should be skipped, nonzero if it should not be skipped
# this semantic enables expressions like "if pkg_skip foo ; then echo "skip foo" ; fi
#
pkg_skip()
{
	local pkg="$(echo $1 | sed 's/:.*//g')"		# Some packages have platform type in them...

	for p in ${ideps} ; do
		if [ "${pkg}" = "${p}" ] ; then
			return 0
		fi
	done

	return 10
}


fetch_distributions()
{
	rm -rf ${dstdir}/Packages
	mkdir ${dstdir}/Packages

	# get ruby distributed packages
	if ! which gem > /dev/null 2>&1 ; then
		echo "Skipping ruby: gem not found in $PATH"
	else
		fetchit_create_ruby
	fi

	# get python distributed packages
	if ! which pip > /dev/null 2>&1 ; then
		echo "Skipping python: pip not found in $PATH"
	else
		fetchit_create_python
	fi

	# create/get debian distributed packages
	fetchit_create_debian
	runit_create
	runit_execute
}


fetchit_create_python()
{
	rm -rf ${dstdir}/Packages/Python
	mkdir ${dstdir}/Packages/Python

	local package=""
	local version=""

	set +o errexit # This command will error when packages are unavialable for download these are acceptable results
	# will move .tgz python directories to the specified folder
	pip list | awk '{pkg=$1; ver=$2; gsub(/[()]/, "", ver); printf("%s %s\n", pkg, ver)}' | while read package version; do
		pip install -d ${dstdir}/Packages/Python $package==$version
	done
	set -o errexit
}

fetchit_create_ruby()
{
	rm -rf ${dstdir}/Packages/Ruby
	mkdir ${dstdir}/Packages/Ruby

	local package=""
	local version=""

	set +o errexit # This command will error when packages are unavialable for download these are acceptable results
	# will move unpacked ruby directories to the specified folder
	gem list | awk '{pkg=$1; ver=$2; gsub(/[()]/, "", ver); printf("%s %s\n", pkg, ver)}' | while read package version; do
		gem unpack $package --target=${dstdir}/Packages/Ruby -v $version;
	done
	set -o errexit
}


fetchit_create_debian()
{
	rm -rf ${dstdir}/Packages/Debian
	mkdir ${dstdir}/Packages/Debian

	local package=""

	dpkg -l | awk '/^ii/ {print $2} {next}' | while read package ; do
		if pkg_skip ${package} ; then
			echo "Skipping $package"
		else
			echo "Processing $package"
			apt-cache show "$package" | \
				awk '/Source:/ {printf("apt-get --allow-unauthenticated --print-uris source %s | tee -a source.log\n", $2)}' \
				| tee -a ${dstdir}/Packages/Debian/fetchit.log
		fi
	done

	sort ${dstdir}/Packages/Debian/fetchit.log | uniq > ${dstdir}/Packages/Debian/fetchit.sh
	chmod 755 ${dstdir}/Packages/Debian/fetchit.sh
}

runit_create()
{
	cat > ${dstdir}/Packages/Debian/runit.sh << EOF
#!/bin/bash

set -o nounset
set -o errexit

touch source.log

source ./fetchit.sh

grep "^'" source.log | sed 's/ .*//g' | sed "s/'//g" | while read uri; do
	wget \$uri
done
EOF
	chmod 755 ${dstdir}/Packages/Debian/runit.sh
}

sshfs_mount()
{
	if [ ! -e ${dstdir} ] ; then
		cleanup_dstdir=yes
		mkdir -p ${dstdir}
	fi
	sshfs ${sshport} ${sshinfo} ${dstdir}
}

runit_execute()
{
	(cd ${dstdir}/Packages/Debian && ./runit.sh)
}

cleanup()
{
	umount ${dstdir} || :
	if [ ${cleanup_dstdir} = "yes" ] ; then
		rm -rf ${dstdir}
	fi
	deps_uninstall
}

usage()
{
	echo "usage: ${1} [-p portnum] user@host:[/path] [mtpt]" 1>&2
}

args_parse()
{
	local prog=${0}
	shift

	while getopts "p:h" flag; do
		case $flag in
		p)
			sshport="-p ${OPTARG}"
			;;
		h)
			usage ${prog}
			exit 0
			;;
		\?)
			echo "unrecognized option" 1>&2
			usage ${prog}
			exit 1
			;;
		*)
			echo "unhandled flag" 1>&2
			usage ${prog}
			exit 1
		esac
	done

	shift $((OPTIND - 1))

	case $# in
	1)
		sshinfo="$1"
		;;
	2)
		sshinfo="$1"
		dstdir="$2"
		;;
	*)
		usage ${prog}
		exit 1
		;;
	esac
}

args_parse $0 $*

trap cleanup EXIT	# NOTE: must come before install to eliminate race condition
deps_install
sshfs_mount
fetch_distributions

# "cleanup" happens automatically when the shell exits
