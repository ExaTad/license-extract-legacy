#!/bin/bash

#
# Copyright © 2015 Exablox Corporation.  All Rights Reserved.
#

set -o errexit
set -o nounset

rm -rf Archive
mkdir Archive

cd Packages

for pkg in $(ls -l | grep '^d' | awk '{print $NF}') ; do
	tar cvfz ../Archive/$pkg.tgz $pkg
done
