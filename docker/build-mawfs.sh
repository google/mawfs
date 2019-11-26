#!/bin/bash

VERSION=0.1.1

apt-get update
apt-get upgrade -y
apt-get install -y git libfuse-dev fuse

# Build the fuse extension.
git clone https://github.com/crack-lang/crack
g++ -fPIC -o _fuse.so -shared -I crack -D_FILE_OFFSET_BITS=64 \
  crack/opt/_fuse.cc /usr/lib/x86_64-linux-gnu/libfuse.so
mv _fuse.so /usr/local/lib/crack-1.5/crack/ext

git clone https://github.com/google/mawfs
git checkout rel-$VERSION
cd mawfs
echo -e "\033[33mbuilding mawfs...\033[0m"
crackc -l lib mawfs
mv mawfs.bin /usr/bin/mawfs
cp mfs /usr/bin/mfs

# cleanup (note that we don't want to remove "fuse")
apt-get -y remove git
apt -y autoremove
rm -rf mawfs crack
