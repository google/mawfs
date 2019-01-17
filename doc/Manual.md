
Installation
============

To run MAWFS, you'll need a fairy recent version of the Crack programming
language (realistically the latest checkin on master if you want to be able to
compile MAWFS AOT using crackc).  See the [crack
documentation](https://github.com/crack-lang/crack/blob/master/INSTALL) for
information on building and installing crack.

Having done this, you should be able to build the mawfs executable as follows:

-   Get the latest mawfs code:

    ```shell
    $ git clone https://github.com/google/mawfs.git
    ```

-   Build the executable:

    ```shell
    $ cd mawfs
    $ crackc -l lib mawfs
    ```

-   Copy the binary to wherever you like:

    ```shell
    $ sudo cp mawfs.bin /usr/local/bin/mawfs
    ```

-   Copy the "mfs" wrapper script, too:

    ```shell
    $ sudo cp mfs /usr/local/bin/mfs
    ```

"crackc" creates a an executable with a ".bin" extension if the source file
doesn't already have an extension, so the next to last step just copies the
file, removing the extension.

"mfs" is a wrapper script written in python, you'll need python 2 installed to
use it.

Now see [Creating a Fresh Instance](creating-a-fresh-instance) below to create an encrypted filesytem.

Getting Started
===============

Creating a Fresh Instance
-------------------------

To create a new instance and a new repository, just run:

```shell
$ mfs new mydir
```

This will prompt you for a password, so enter a new password for your
repository.

if successful, `mfs` will print out something like:

```
port is 9131
MAWFS Cell created
"mawfs run /home/user/.mawfs/XRB4uXqkuChYuponWjastg <mount-point>" to begin using it.
Your filesystem has been mounted on mydir
```

The command on the next to last line is the command that you would run to
mount your directory from the low-level "mawfs" command.
`/home/user/.mawfs/FY56evj1J6zLEboPGXjbbw` is the name of the "backing
store" directory, where mawfs stores all of the encrypted contents and
meta-data of your filesystem.  You won't normally have to deal with this, the
`mfs` command has created that directory based on the MD5 hash of the
mount-point yoy specified.  It has also started an instance that mounts the
directory you specified.  (Note that because it's based on the path name, you
will have to deal with it if you try to move the directory!)

You can verify that the instance is running by checking for the presence of
the special ".mawfs" admin directory:

```shell
$ cat mydir/.mawfs/README
```

This should display a brief copyright and license notice.

Now you should be able to deal with `mydir` like any other filesystem:

```shell
$ echo this is a test >mydir/myfile
$ cat mydir/myfile
this is a test
```

If you unmount it, it should all go away:

```shell
$ mfs umount mydir
$ cat mydir/myfile
cat: mydir/myfile: No such file or directory
```

As a FUSE filesystem, your MAWFS instance should be invisible to all users of
the system but you.

The `info` command can be used to obtain information about your instance:

```shell
$ mfs -i mydir info
Backing-dir: /home/user/.mawfs/XRB4uXqkuChYuponWjastg
Server-Iface: 127.0.0.1
Server-Port: 9131
```

Note that the instance is bound to port 9131 on 127.0.0.1 (the "localhost"
interface).  `mfs` doesn't configure instances to run on publicly visible
interfaces, mawfs doesn't have an authentication mechanism yet.  We'll discuss
tunneling connections over ssh below.

Cloning an Instance
-------------------

At this point, you have a single running MAWFS instance.  Now let's clone it
and create a "cell".  A MAWFS cell is a set of instances sharing a common
respository.  First let's remount the original instance (if it's currently
unmounted):

```shell
$ mfs mount mydir
```

We can clone our instance as follows:

```shell
$ mfs clone mydir clone
cloning into /home/mmuller/.mawfs/B7l+LACuug8qZ0AdbD5bEw (mount = clone)
password:
master is BUyVNk9R9qB418zvmUHe4F_MpaDgFxmwsM.4271_LTI
getting head
writing master...
"mawfs run /home/user/.mawfs/B7l+LACuug8qZ0AdbD5bEw <mount-point>" to begin using it.
or just run /usr/local/bin/mfs mount clone
```

As the output indicates, we can now mount our new clone:

```shell
$ mfs mount clone
```

We can also inspect the information on the new instance:

```shell
$ mfs -i clone info
Backing-dir: /home/user/.mawfs/B7l+LACuug8qZ0AdbD5bEw
Server-Iface: 127.0.0.1
Server-Port: 9132
peer-origin: host=127.0.0.1 port=9132
```

Note that in addition to the information we saw with "mydir", there is also a
"peer-origin" line.  Clones retain the address of the peer that they were
created from.  Note that the new peer has not been added to the original
instance:

```shell
$ mfs info mydir
Backing-dir: /home/user/.mawfs/XRB4uXqkuChYuponWjastg
Server-Iface: 127.0.0.1
Server-Port: 9131
```

We can add the new instance as a peer to the original instances with the
"add_peer" command:

```shell
$ mfs -i mydir add_peer clone 127.0.0.1:9132
```

When you clone an instance, you get the origin peer for free.  All other
peers must be added explicitly.

Changing and Pulling
--------------------

Now let's modify some files:

```shell
$ echo 'this is a test' >mydir/test.txt
$ cat mydir/test.txt
this is a test
```

We can pull this change to our clone:

```shell
$ mfs -i clone pull  master
```

And now we have the change in the clone:

```shell
$ cat clone/test.txt
this is a test
```

Likewise, we can add a change to the clone and propagate it to mydir:

```shell
$ echo from the clone >>clone/test.txt
$ mfs -i mydir pull master
Branch loaded into master
$ cat mydir/test.txt
this is a test
from the clone
```

What happens if we introduce a change to both instances?

```shell
$ echo added from mydir >> mydir/text.txt
$ echo added from clone >> clone/text.txt
$ mfs -i clone pull master
Branch loaded into 127.0.0.1:9131:master
```

In this case, things get a little more complicated.  We couldn't just append
the changes to "master", so we had to create a new branch:
"127.0.0.1:9131:master".  This is the "master" branch as known by "mydir" (on
port 9131).  Borrowing git nomenclature, this is called a "tracking branch."
It tracks the branch from a different instance.

We don't have the changes from mydir, we have to do a merge.

Merging
-------

We can merge another branch using the "merge" command.  Let's merge the branch
we pulled from mydir:

```shell
$ mfs -i clone merge 127.0.0.1:9131:master
created merge branch: merge:fiH5x6xye4NxbVQGuhsvvki8yyUixLBXE6kUM1sxZus
merging test.txt

There were conflicts in the files above.  Fix them and use "mfs resolve" to complete the merge.
```

The merge command reported conflicts.  When a file has changed in two
different ways, mawfs merges the changes using the `diff3` program.  We can
see the result of this in the test.txt file:

```shell
$ cat clone/test.txt
this is a test
from the clone
<<<<<<< /home/mmuller/tmp/clone/test.txt
added from clone
||||||| /home/mmuller/tmp/clone/.mawfs/alt/merge:org/test.txt
=======
added from mydir
>>>>>>> /home/mmuller/tmp/clone/.mawfs/alt/127.0.0.1:9132:master/test.txt
```

The common portion of the file is the two lines at the top.  The remaining
lines illustrate the divergence: the local change shows up first under the
"<<<<<<" line, the original contents (in this case, nothing) shows up between
the "||||||" and "======" lines, and the branch from mydir appears between
the "======" and ">>>>>>" lines.

Note that the original and alternate versions of the file are actually present
in the mawfs filesystem under the special ".mawfs" directory.  We can easily
view their contents:

```shell
$ cat clone/.mawfs/alt/merge:org/test.txt
this is a test
from the clone
$ cat clone/.mawfs/alt/127.0.0.1:9132:master/test.txt
this is a test
from the clone
added from mydir
```

If we edit the file to remove the diff markups, we end up with this:

```shell
$ cat clone/test.txt
this is a test
from the clone
added from clone
added from mydir
```

We now have two options.  We can cancel the merge:

```shell
$ mfs -i clone cancel_merge
Merge cancelled
```

This reverts the changes and restores the instance to its previous state.

Alternately, we can "resolve" the merge:

```shell
$ mfs -i clone resolve
Resolve completed successfully
```

This commits the changes to the original target branch.  Now, if we were to
pull the branch into "mydir":

```shell
$ mfs -i mydir pull master                                                                                                           ~/tmp
Branch loaded into master
$ cat mydir/test.txt
this is a test
from the clone
added from clone
added from mydir
```

In this case, we successfully pulled the new branch into master, we didn't
create a tracking branch.

This worked because of our merge.  When we merged, we created a new commit (a
snaopshot of the filesystem at a given point in time) derived from the changes
to "master" from both instances.  Note that if we had made more changes to
mydir between pulling from it pulling the merged commit back to it, we would
have again created a tracking branch for clone.

This is the general MAWFS workflow: make whatever changes you want within
local instances and merge frequently.

Exercising Caution
------------------

As stated earlier, MAWFS is strictly beta code.  While we've made every
attempt to protect your data, there are likely to be bugs that could cause
data loss, or expose your data to prying eyes.  There are several mitigation
strategies that you can use to protect yourself against known and unknown
deficiencies:

-   Tunnel all peer connections over ssh.
-   Backup the backing directory, and, if possible, the plaintext directory
    too.

### SSH Tunneling


MAWFS encrypts data, but does not yet encrypt connections.  There are things
that a man-in-the-middle attacker can observe in transit which can allow them
to infer metadata that may be useful in a subsequent attack on the data, or,
in the case of connection hijacking, even destroy a repository.

To guard against this, we recommend tunneling connections over ssh.  To set up
an ssh tunnel, first pick an unused port on the machine that you will be
ssh'ing to.  We'll use port 8111 in this example.

Depending on which machine you're existing instance is on, you'll want to
either clone on your local machine or clone on the remote machine.

#### Cloning to a Remote Machine


Let's say that we have a local instance running on port 9131.  To clone to the
remote machine:

```shell
$ ssh -R 8111:localhost:9131 remote-host.example.com \
    mfs clone localhost:8111 rclone
```

The example above assumes that we want the port for our existing instance to
be 8111 on the remote machine.  Obviously, it also assumes mawfs is installed
on the remote machine.

On the remote machine, we would now start mawfs and determine the server port:

```shell
$ ssh remote-host.example.com
remote-host$ mfs mount rclone
password: <your-passwd>
$ mfs -i rclone info
Backing-dir: /home/user/.mawfs/v3eN/5Md9HHRtGdhKEZbdA
Server-Iface: 0.0.0.0
Server-Port: 9133
peer-origin: host=127.0.0.1 port=8111
```

The important piece of information here is the "Server-Port" entry which is
9133.  We can assume that our new instance is running bound to that port.

Now we can start a tunnel:

```shell
$ ssh -L 8111:localhost:9133 -R 8111:localhost:9131
```

and add the new remote instance as a peer

```shell
$ mfs -i myinst add_peer remote-host localhost:8111
```

The argument to "-L" causes ssh to tunnel connections to local port 8111 to
port 9133 (our new instance) on the remote machine.  The argument to -R causes
ssh to tunnel port 8111 on the /remote/ machine to port 9131 on our local
machine.

#### Cloning from a Remote Machine

The process for cloning from a remote machine is roughly the opposite.  Let's
say once again that the instance on our remote machine is running on port 9133.
To clone to the local machine, we would first start a tunnel:

```shell
$ ssh -L 8111:localhost:9133 remote-host.example.com
```

Then we can safely clone a new instance:

```shell
$ mfs clone localhost:8111 lclone
```

We can now obtain the port of the local instance:

```shell
$ mfs -i lclone info
Backing-dir: /home/user/.mawfs/P3VC9fhG/9pYFB0XLk21GA
Server-Iface: 0.0.0.0
Server-Port: 9135
peer-origin: host=127.0.0.1 port=8111
```

We can now set up the tunnel as follows:

```shell
$ ssh -L 8111:localhost:9133 -R 8111:localhost:9135
```

Finally, we'll want to add our new instance as a peer on the remote
machine:

```shell
remote-host$ mfs myinst add_peer newinst localhost:8111
```

### Backup

A MAWFS instance stores data as encrypted chunks and meta-data files in a
backing directory on the normal filesystem.  As such, it is possible to backup
the entire state of an instance to a remote system with something as simple as
rsync:

```shell
$ rsync -a `mfs -i myinst backing_dir`/ remote-host:myinst-backup/
```

The danger with this is that if a local chunk file becomes corrupted, it will
destroy its backup as well!

To protect against this, first verify the backing directory using "#mawfs
verify#":

```shell
if mfs verify -i myinst; then
    rsync -a `mfs -i myinst backing_dir`/ remote-host:myinst-backup/
fi
```


Server Configuration
--------------------

The server configuration is defined in config/peers in the "server" section.
Three variables are used to define a server:

`enabled`
:   Values: "true" or "false".  Enable the server.  This is mainly useful as
    an easy way to start the server on 0.0.0.0:9119.  If this is what you
    want, this is the only variable that you need.  If you need anything more
    specialized, you don't need to specify "enabled"

`iface`
:   Value: A dotted IPv4 address, e.g. "127.0.0.1".  The interface to run the
    server on.  Implies "enabled = true".  defaults to "0.0.0.0"

`port`
:   Value: an integer, e.g. "8080".  The port number to bind to.  Implies
    "enabled = true".  Defautls to "9119"

Specifying any of "enabled", "iface" or "port" starts a server.


To start a server listening on all interfaces, port 9119:

```
[server]
enabled = true
ssl = true
```

To start a server listening on 9119 on localhost:

```
[server]
iface = 127.0.0.1
ssl = true
```

To start a server listening on 1.2.3.4:1234:

```
[server]
iface = 1.2.3.4
port = 1234
ssl = true
```

The "ssl" flag shown above enables SSL (TLS, actually) for the server socket.
It should always be used for new installations unless for some reason an
insecure connection is desired, in which case appropriate steps should be
taken to secure the machine and port.

Peer Configuration
------------------

Though it's usually easier just to use the "`add_peer`" command, peer
configuration is also defined in config/peers.  Each peer has its own section
whose name must begin with "peer-". The section name after the hyphen is used
as the peer name.  The other variables are:

-   'host'.  The host name (or IP address).
-   'port'.  The port number (defaults to 9119 if not present).

Example:

```
[peer-origin]
host = example.com
port = 1234

[peer-backup]
host = backup.example.com
```
