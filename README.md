# Gemplex

Gemplex is a Gemini server, capable of serving multiple capsules. Apart from
static content, it also supports serving dynamic content through CGI scripts.

Gemplex is in pre-pre-alpha stage, meaning it hasn't been written at the time of
this writing. Development is going to start real soon now, hopefully! Check the
"Completion Status" section to find out more.

## Completion Status

The following features are planned for Gemplex.

 - [ ] Serve static content
 - [ ] Serve dynamic content through CGI
 - [ ] Serving multiple capsules using SNI
 - [ ] Prefix routes
 - [ ] URL routes
 - [ ] Write a more complete documentation available on Gemini
 - [ ] Client certificates
 
And maybe later:

 - [ ] Regex routes
 - [ ] Longest match pattern matching
 
## Config File

Gemplex uses a json formatted configuration file. Here's an example:

``` json
{
    "listen": "0.0.0.0:1965",

    "routes": [
        {
            "prefix": "gemini://example.org/blog/",
            "type": "static",
            "location": "/srv/gemini/gemlog/",
            "strip_file_ext": true
        },
        {
            "url": "gemini://example.com/search",
            "type": "cgi",
            "script": "/var/cgi/search.cgi"
        },**
        {
            "prefix": "gemini://example.org",
            "type": "static",
            "location": "/srv/gemini/example.org/",
            "strip_file_ext": true
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

### Routes

Each item in the routes list defines either a separate capsule or a part of one.
Gemplex goes through this list one by one and look for a match. It is important
then, that longer/more specific rules are put higher up in the list.

The following keys can be used in each item:

 - `prefix`: The url prefix to match. The gemini:// scheme can optionally be
   dropped.
 - `regex`: A regular expression. The supported syntax is Go's. The gemini://
   scheme should not be included.
 - `url`: The full url to match, excluding query parameters. The gemini://
   scheme can optionally be dropped.
 - `hostname`: The hostname to match.
 - `type`: The type of content to serve. Can be either `static` or `cgi`.
 - `location`: The location to serve static content from.
 - `script`: The path to the CGI script.
 - `strip_file_ext`: If set to true, the `.gmi` extension is stripped from
   static filenames, when serving, so `/page.gmi` is accessed at `/page`. (This
   option can also be set globally.)
 
 When matching urls against patterns, a trailing slash is always added if not
 present, so that `/page/` and `/page` can be treated the same.

### Certificates

The `certs` key contains a list of certificates to be used by Gemplex when
terminating tls.

 - `cert`: The certificate file.
 - `key`: The certificate key file.
