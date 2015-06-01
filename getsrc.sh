#!/bin/bash

#
# Copyright © 2015 Exablox Corporation.  All Rights Reserved.
#

set -o nounset
set -o errexit

if [ $# -ne 1 ] ; then
	echo "usage: $0 <package>" 1>&2
	exit 1
fi

package=$1

rm -rf Packages
mkdir Packages

#
# With "--show-installed" enabled, packages installed on the system will have color = honeydew
#
debtree --show-installed --no-skip --show-all $package | \
	awk '
		/^	"alt1":.*fillcolor=honeydew/ {
			pkg=$1
			sub(/"alt1":/, "", pkg)
			gsub(/"/, "", pkg)
			print pkg
			next
		}
		/^	".*fillcolor=honeydew/ {
			pkg=$1
			gsub(/"/, "", pkg)
			print pkg
			next
		}
		{
			next
		}
	' \
| (while read pkg; do
	apt-cache show $pkg | awk '/Source:/ {printf("apt-get --allow-unauthenticated --print-uris source %s | tee -a source.log\n", $2)}'
done) | sort | uniq > Packages/fetchit.sh

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
