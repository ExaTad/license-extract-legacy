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

if [ $# -ne 3 ] ; then
	echo $#
	echo "Usage: $0 URL CommitId LocalDirectory" 1>&2
	exit 1
fi

URL="$1"
commit="$2"
dir="$3"

cwd="$(pwd)"

if [ -d "${dir}" ] ; then
	cd "${dir}"
	curcommit="$(cat upstream.commit)"
	if [ "${curcommit}" = "${commit}" ] ; then
		echo "${dir}: already @ ${commit}: nothing to do"
		exit 0
	fi
	cd "${cwd}"
fi

rm -rf "${dir}"
mkdir -p "${dir}"
cd "${dir}"
git clone "${URL}" .
git checkout "${commit}"
echo "${commit}" > upstream.commit
rm -rf .git
git add -A .
