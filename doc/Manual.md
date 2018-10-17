

Configuring a Server
--------------------

The server configuration is defined in config/peers in the "server" section.
Three variables are used to define a server:

enabled::
    Values: "true" or "false".  Enable the server.  This is mainly useful as
    an easy way to start the server on 0.0.0.0:9119.  If this is what you
    want, this is the only variable that you need.  If you need anything more
    specialized, you don't need to specify "enabled"
iface::
    Value: A dotted IPv4 address, e.g. "127.0.0.1".  The interface to run the
    server on.  Implies "enabled = true".  defaults to "0.0.0.0"
port::
    Value: an integer, e.g. "8080".  The port number to bind to.  Implies
    "enabled = true".  Defautls to "9119"

Specifying any of "enabled", "iface" or "port" starts a server.


To start a server listening on all interfaces, port 9119:

```
    [server]
    enabled = true
```

To start a server listening on 9119 on localhost:

```
    [server]
    iface = 127.0.0.1
```

To start a server listening on 1.2.3.4:1234:

```
    [server]
    iface = 1.2.3.4
    port = 1234
```

Configuring Peers
-----------------

Peer configuration is also defined in config/peers.  Each peer has its own
section whose name must begin with "peer-". The section name after the hyphen
is used as the peer name.  The other variables are:

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

