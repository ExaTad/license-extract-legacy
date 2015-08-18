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

bindir="$(cd $(dirname "$0"); pwd)"

if [ $# -ne 2 ] ; then
	echo "usage: $0 <path to Packages directory> <outdir>"
	exit 1
fi

dir="$1"
outdir="$2"

if [ ! -e ${outdir} ] ; then
	mkdir -p ${outdir}
else
	if [ ! -d ${outdir} ] ; then
		echo "ERROR: ${outdir}: not a directory" 1>&2
		exit 1
	else
		rm -rf ${outdir}
		mkdir -p ${outdir}
	fi
fi

aoutdir=$(cd $outdir && pwd)

cp ${bindir}/style.css ${aoutdir}

cat > ${aoutdir}/index.html << EOF
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<link rel="stylesheet" type="text/css" href="style.css">
</head>
<body>
<div class="overview">
EOF

cat ${bindir}/overview-notice.html >> ${aoutdir}/index.html

cat >> ${aoutdir}/index.html << EOF
</div>
<div class="package-list">
<H2>Packages</H2>
<table>
	<tr>
		<td></td>
		<td><h3>Package</h3></td>
		<td><h3>Notices & Licenses</h3></td>
		<td><h3>Source Code</h3></td>
	</tr>
EOF

case $(uname) in 
Darwin)
	host=osx
	;;
Linux)
	host=linux64
	;;
esac


#
# Pick a verbosity level
#
# verbose=true
verbose=false
#
n=1
for pkgpath in $(find ${dir} -type f -print) ; do
	pkg="$(basename ${pkgpath})"

	echo "[mknotices] Creating Notices for ${pkg}" 1>&2

	rm -rf pkg.tmp
	mkdir pkg.tmp

	cd pkg.tmp
	echo "[mknotices] Extracting ${pkg}" 1>&2
	tar xfz "${pkgpath}"


	echo "[mknotices] Generating Notices for ${pkg}" 1>&2

	pkgdir="$(ls -l | grep '^d' | awk '{print $NF}' | head -1)"
	mkdir -p "${aoutdir}/${pkgdir}"
	cd ${pkgdir}
	license-extract-EXTRACTVERSION-$host \
		-style ../style.css \
		-ldir "${aoutdir}/${pkgdir}" \
		-verbose=${verbose} \
		-o "${aoutdir}/${pkgdir}/${pkgdir}.html" \
		"."

	echo '	<tr>'									>> ${aoutdir}/index.html
	echo '		<td>'${n}'</td>'						>> ${aoutdir}/index.html
	echo '		<td>'${pkgdir}'</td>'						>> ${aoutdir}/index.html
	echo '		<td><a href="'${pkgdir}/${pkgdir}.html'">Browse</a></td>'	>> ${aoutdir}/index.html
	echo '		<td><a href="'${pkgdir}/${pkg}'">Download </a></td>'		>> ${aoutdir}/index.html
	echo '	</tr>'									>> ${aoutdir}/index.html

	echo "[mknotices] Copying Package ${pkg}" 1>&2
	cp ${pkgpath} ${aoutdir}/${pkgdir}

	cd ..
	n=$(( $n + 1 ))
done

cat >> "${aoutdir}/index.html" << EOF
</table>
</div> <!-- end "package-list" -->
<p><i>Created with opensrc toolchain release EXTRACTVERSION</i></p>
</body>
</html>
EOF

name="$(basename ${aoutdir})"
cd "${aoutdir}/.."

echo "[mknotices] Creating Archive $(pwd)/${name}.tgz" 1>&2
rm -f ${name}.tgz
tar cvfz ${name}.tgz ${name}

