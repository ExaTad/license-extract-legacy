#!/bin/bash

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
