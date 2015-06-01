#!/bin/bash

#
# Copyright © 2015 Exablox Corporation.  All Rights Reserved.
#

set -o nounset
set -o errexit

rm -rf Packages
mkdir Packages

dpkg -l | awk '/^ii/ {print $2} {next}' | while read package ; do
	echo "Running $package"

	apt-cache show "$package" | \
		awk '/Source:/ {printf("apt-get --allow-unauthenticated --print-uris source %s | tee -a source.log\n", $2)}' \
		>> Packages/fetchit.log
done

sort Packages/fetchit.log | uniq > Packages/fetchit.sh

cat > Packages/runit.sh << EOF
#!/bin/bash

set -o nounset
set -o errexit

touch source.log

source ./fetchit.sh

grep "^'" source.log | sed 's/ .*//g' | sed "s/'//g" | while read uri; do
	wget \$uri
done
EOF

chmod 755 Packages/runit.sh Packages/fetchit.sh
echo "To fetch the source:"
echo "cd Packages && ./runit.sh"
