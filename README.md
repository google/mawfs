
MAWFS - A Personal Encrypted Distributed Filesystem
===================================================

MAWFS aims to be an encrypted, distributed, branching filesystem.  It is
currently in early beta.  You can create an encrypted filesystem and
replicate with merging among a set of peers.

If you want to play with it, the easiest way to do so is to pull the docker
image `crack/mawfs` from docker hub.  Alternately, you can build from source.

MAWFS is written in [Crack.](http://crack-lang.org).  If you want to build it
from source, you'll need at least Crack 1.4, and preferably HEAD on the master
branch, to do so.

Usage Example
-------------

    $ mkdir backing fs  # Create a directory for the backing store and a
                        # mountpoint for the filesystem
    $ export CRACK_LIB_PATH=mawfs/lib
    $ echo 's3cr37-p455w0rd!' | mawfs/mawfs run backing fs
    $ echo 'this will be encrypted!' >fs/myfile.txt
    $ cat fs/myfile.txt
    $ mawfs/mawfs commit fs  # generate a commit record.
    $ fusermount -u fs  # To unmount.

Every directory in the filesytem includes a special maintenance directory
(".mawfs", which is invisible to a directory listing) that allows you to
interact with the filesystem.  This current contains a "README" file
containing information about MAWFS and a "branch" file containing the current
branch:

    $ cat fs/.mawfs/branch
    master

This code is still likely to have a few bugs in it and we will likely
introduce incompatibilities, so it is not recommended that you use MAWFS
for anything important yet.

Current Status
--------------

The system is currently in early beta.  While it is largely usable for
experimental purposes, you are advised against relying on the security and
integrity of the system at this time.

If you do use it for anything important, you are encouraged to make local
backups and perform regular verification.  You are also encouraged to
[subscribe to the mailing list](https://groups.google.com/forum/#!forum/mawfs-dev)
to stay notified of any security issues that emerge.

Bug reports and code contribuutions gladly accepted.  Please sign Google's
[Contributor License
Agreement](https://groups.google.com/forum/#!forum/mawfs-dev) before sending
any pull requests.

Contacts
--------

-   [Mailing List](https://groups.google.com/forum/#!forum/mawfs-dev)
    mawfs-dev@googlegroups.com

Related Projects
----------------

-   [Peergos](http://peergos.org) - this is a very similar project which MAWFS
    will hopefully be interoperable with at some point.
-   [IPFS](http://ipfs.io/).
-   [ORI](http://ori.scs.stanford.edu/)

"MAWFS" stands for "Mike's Awesome Filesystem".  It started out that way and
we have since been unable to come up with a better name :-)

MAWFS is not an official Google project.
