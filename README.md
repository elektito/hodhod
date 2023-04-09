# About Gemplex

Gemplex is a Gemini server, capable of serving multiple capsules. Apart from
static content, it also supports serving dynamic content through CGI scripts.

Gemplex is still far from being battle-tested and feature complete. Check the
"Completion Status" section to find out more.

# Completion Status

The following features are planned for Gemplex.

 - [x] Serve static content
 - [x] Serve dynamic content through CGI
 - [x] Serving multiple capsules using SNI
 - [x] Prefix routes
 - [x] URL routes
 - [ ] Write a more complete documentation available on Gemini
 - [ ] Client certificates
 - [ ] Redirects
 
And maybe later:

 - [ ] Regex routes
 - [ ] Longest match pattern matching
 
# Installation

You can install gemplex by running:

``` sh
go install github.com/elektito/gemplex@latest
```

# Config File

Gemplex uses a json formatted configuration file. Here's an example:

``` json
{
    "listen": "0.0.0.0:1965",

    "match_options": {
        "query_params": "remove",
        "trailing_slash": "esnure"
    },

    "routes": [
        {
            "prefix": "gemini://example.org/blog/",
            "backend": "gemlog"
        },
        {
            "url": "gemini://example.com/search",
            "backend": "search"
        },
        {
            "prefix": "gemini://example.org",
            "backend": "home"
        }
    ],
    
    "backends": [
        {
            "name": "gemlog",
            "type": "static",
            "location": "/srv/gemini/gemlog/",
            "file_ext": "strip"
        },
        {
            "name": "search",
            "type": "cgi",
            "script": "/var/cgi/search.cgi"
        },
        {
            "name": "home",
            "type": "static",
            "location": "/srv/gemini/example.org/",
            "file_ext": "strip"
        }
    ],

    "certs": [
        {
            "cert": "/etc/certs/example.com.cer",
            "key": "/etc/certs/example.com.key"
        },
        {
            "cert": "/etc/certs/star.example.org.cer",
            "key": "/etc/certs/star.example.org.key"
        }
    ]
}
```

## Routes

Routes are patterns that match urls to backends. Each route must have a
`backend` key specifying the name of the backend config, as well as a pattern
that can be specified by one of the following keys:

 - `prefix`: The url prefix to match. Including the `gemini://` scheme is not
   mandatory.
 - `url`: The full url to match. Including the `gemini://` scheme is not
   mandatory.
 - `hostname`: The hostname to match.
   
Query parameters are normally ignored when matching. If you want to change this
behavior, you can set the global `match_options.query_params` field to one of
these values:

 - `remove`: The default behavior. The query part of the URL is removed before
   pattern matching.
 - `include`: The query part of the URL is included when pattern matching.
 
When matching urls against patterns, a trailing slash is by default added if not
present, so that `/page/` and `/page` can be treated the same. If you don't want
this behavior, you can use the global `match_options.trailing_slash` field. The
following values are allowed for this field:

 - `ensure`: The default behavior. The trailing slash is added to all request
   URLs that don't have one, before pattern matching.
 - `remove`: The trailing slash, if present, is always removed from the request
   URL before pattern matching.
 - `ifpresent`: Gemplex will not add or remove trailing slashes. The trailing
   slash, if present, will be part of the URL when matching for patterns.

## Backends

Each backend specifies a source of gemini pages. The following fields are
mandatory for all backends:

 - `name`: The name by which we refer to this backend in the routes.
 - `type`: The type of the backend. Can be either `static` or `cgi`.
 
Each backend type has its own set of other fields that can specify its behavior.

For `static` backends, the following fields are available:

 - `location`: Mandatory. The location to serve static content from. Must point
   to a valid directory.
 - `file_ext`: Optional. Can be set to `strip` or `include`. If set to `strip`
   (the default behavior), `/page.gmi` can be accessed as `/page`. If set to
   `include`, the filename in the request path must be the same as the filename
   on the file system.

For `cgi` backends, the following fields are available:

 - `script`: The path to the CGI script.

## Certificates

The `certs` key contains a list of certificates to be used by Gemplex. The
appropriate certificate will be chosen and served based on the request SNI
value.

 - `cert`: The certificate file.
 - `key`: The certificate key file.
