#!/bin/bash -x

#
# Copyright © 2015 Exablox Corporation.  All Rights Reserved.
#

#
# Note: This must be run on a debian linux box, as dpkg-source doesn't work
# reliably elsewhere.
#

set -o errexit
set -o nounset

cd Packages

for i in *.dsc ; do
	dpkg-source -x --no-check $i
done
