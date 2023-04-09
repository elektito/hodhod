package gemplex

import "net/url"

type Request struct {
	Url        *url.URL
	RemoteAddr string
}
