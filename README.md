
MAWFS - A Personal Encrypted Distributed Filesystem
===================================================

MAWFS aims to be an encrypted, distributed, branching filesystem.  It is
currently very much in development, but still somewhat usable in that it
preseents a FUSE filesystem around an encrypted backing store.

MAWFS is written in [Crack.](http://crack-lang.org)  If you want to play with
it, you'll need to install crack 1.3.

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

Current Status and Ultimate Goal
--------------------------------

The system currently presents a FUSE based filesystem that stores its data as
encrypted, content addressable objects in a backing store.  There is a
git-like commit history that records the state of the filesystem at a sequence
of points in its history and an unencrypted branch file in the backing store
that points to the latest commit for the branch (the "branch head").

The backing store also contains a journal file consisting of a sequence of
mutations to the filesystem since the last commit.  This journal is erased
upon performing the next commit, which is simply an "fsync" applied to the
root directory that can be performed using the "mawfs commit" command.

The eventual goal is to provide a system (ideally interoperable with IPFS and
Peergos, see below) that allows a MAWFS filesystem to be automatically
replicated across multiple peers.  Since peers can be used offline, and
therefore can diverge, we will also support user directed conflict resolution.

Related Projects
----------------

-   [Peergos](http://peergos.org) - this is a very similar project which MAWFS
    will hopefully be interoperable with at some point.
-   [IPFS](http://ipfs.io/).
-   [ORI](http://ori.scs.stanford.edu/)

"MAWFS" stands for "Mike's Awesome Filesystem".  It started out that way and
we have since been unable to come up with a better name :-)

MAWFS is not an official Google project.
