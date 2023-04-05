# Gemplex

Gemplex is a Gemini server, capable of multiplexing connections to multiple
upstream servers, whether based on hostname or path, and also serving static
content or running CGI scripts on its own.

## Completion Status

This is a speculative readme at this point. Not everything I write here about is
actually implemented. Here's a task list to give an idea of what is implemented
now, and what is not.

 - [ ] Serve static content
 - [ ] Run CGI scripts
 - [ ] Multiplex requests to upstreams based on SNI (supports client certificates)
 - [x] Multiplex requests to upstreams based on URL path (does not support client certificates)
 - [ ] Spartan (plain text) upstream
 - [ ] Prefix routes
 - [ ] Regex rules
 - [ ] URL routes
 - [ ] Catch-all routes
 - [ ] Upstreams using UNIX sockets
 
## Config File

Gemplex uses a json formatted configuration file. Here's an example:

``` json
{
    "listen": "0.0.0.0:1965",
    "routes": [
        {
            "hostname": "gardening.example.org",
            "upstream": "gardener"
        },
        {
            "hostname": "culture.example.org",
            "upstream": "culture"
        },
        {
            "prefix": "gemini://example.com/foo/",
            "upstream": "foo"
        },
        {
            "url": "gemini://example.com/bar/",
            "upstream": "bar"
        },
        {
            "catch_all": true,
            "upstream": "default"
        }
    ],
    "upstreams": [
        {
            "name": "gardner",
            "addr": "localhost:19650"
        },
        {
            "name": "foo",
            "addr": "/var/run/foo/gemini.sock",
            "tls": false
        },
        {
            "name": "bar",
            "location": "/srv/gem/"
        },
        {
            "name": "default",
            "cgi": "/srv/cgi-bin/gemini.cgi"
        }
    ],
    "certs": [
        {
            "host": "*.example.com",
            "cert": "/etc/certs/example.com.cer",
            "key": "/etc/certs/example.com.key"
        },
        {
            "host": "gardening.example.org",
            "cert": "/etc/certs/gardening.example.org.cer",
            "key": "/etc/certs/gardening.example.org.key"
        }
    ]
}
```

### Routes

Routes with a "hostname" use sni if the upstream supports it. You won't need to
provide a certificate for these. If the upstream does not support tls (that is,
it uses the spartan protocol), then Gemplex will terminate tls, and it would
then need to have a certificate provided for the hostname.

Route objects can have the following fields:

 - `hostname`
 - `prefix`
 - `regex`
 - `url`
 - `catch_all`
 - `upstream`

"upstream" is mandatory for all routes. One, and only one, of the other fields
must be used for each routes.

### Upstreams

An upstream object can have the following fields:

 - `name`
 - `addr`: Either a "hostname", a "hostname:port" pair, or a path to a UNIX
   socket (which must start with a `/`). If no port is specified, it defaults to
   `1965`.
 - `location`: A directory from which files are served.
 - `cgi`: Path to a GCI script.
 - `tls`: A boolean value which defaults to `true`. If false, Gemplex talks to
   upstreams in plain text.

### Certificates

The `certs` key contains a list of certificates to be used by Gemplex when
terminating tls. This is only needed if you use path-based routes. The following
values can be used in certificate objects:

 - `hostname`: The hostname to use the certificate for. This can be a single
   hostname like `example.com`, or it can be a wildcard like `*.example.com`.
   Notice that only a single wildcard at the beginning is supported, and it must
   be followed by a `.`.
 - `cert`: The certificate file.
 - `key`: The certificate key file.
