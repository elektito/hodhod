package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	ConfigFilePath = "config.json"
)

type Route struct {
	Prefix   string `json:"prefix"`
	Url      string `json:"url"`
	Hostname string `json:"hostname"`
	Backend  string `json:"backend"`
}

type Backend struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Location string `json:"location"`
	FileExt  string `json:"file_ext"`
	Script   string `json:"cgi"`
}

type Cert struct {
	CertFile string `json:"cert"`
	KeyFile  string `json:"key"`
}

type MatchOptionsConfig struct {
	QueryParams   string `json:"query_params"`
	TrailingSlash string `json:"trailing_slash"`
}

type GemplexConfig struct {
	ListenAddr   string             `json:"listen"`
	MatchOptions MatchOptionsConfig `json:"match_options"`
	Routes       []Route            `json:"routes"`
	Backends     []Backend          `json:"backends"`
	Certs        []Cert             `json:"certs"`
}

func Load() (config GemplexConfig, err error) {
	f, err := os.Open(ConfigFilePath)
	if err != nil {
		return
	}
	defer f.Close()

	// set top-level default values
	config.ListenAddr = "127.0.0.1:1965"
	config.MatchOptions.QueryParams = "remove"
	config.MatchOptions.TrailingSlash = "ensure"

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)

	if err == nil {
		setDefaultsAndNormalize(&config)
		err = validateConfig(&config)
	}

	return
}

func (cfg *GemplexConfig) GetBackendByName(name string) *Backend {
	for _, backend := range cfg.Backends {
		if backend.Name == name {
			return &backend
		}
	}

	return nil
}

func (cfg *GemplexConfig) GetBackendByUrl(urlStr string) (backend *Backend, unmatched string) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return
	}

	if cfg.MatchOptions.QueryParams != "include" {
		u.RawQuery = ""
	}

	if cfg.MatchOptions.TrailingSlash == "ensure" {
		if !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}
	} else if cfg.MatchOptions.TrailingSlash == "remove" {
		if strings.HasSuffix(u.Path, "/") {
			u.Path = u.Path[:len(u.Path)-1]
		}
	}

	for _, route := range cfg.Routes {
		switch {
		case route.Hostname != "" && route.Hostname == u.Hostname():
			unmatched = u.Path
			if len(unmatched) > 0 {
				// remove leading slash, so we can join the path to "location"
				unmatched = unmatched[1:]
			}
			backend = cfg.GetBackendByName(route.Backend)
			return
		case route.Prefix != "" && strings.HasPrefix(u.Path, route.Prefix):
			if len(u.Path) > len(route.Prefix) {
				unmatched = unmatched[len(route.Prefix):]
				if unmatched[0] == '/' {
					unmatched = unmatched[1:]
				}
			}
			backend = cfg.GetBackendByName(route.Backend)
			return
		case route.Url != "" && route.Url == u.String():
			backend = cfg.GetBackendByName(route.Backend)
			return
		}
	}

	return
}

func setDefaultsAndNormalize(cfg *GemplexConfig) {
	for i, route := range cfg.Routes {
		if route.Prefix != "" && !strings.HasPrefix(route.Prefix, "gemini://") {
			cfg.Routes[i].Prefix = "gemini://" + route.Prefix
		}

		if route.Url != "" && !strings.HasPrefix(route.Url, "gemini://") {
			cfg.Routes[i].Url = "gemini://" + route.Url
		}
	}

	for i, backend := range cfg.Backends {
		if backend.Type == "static" && backend.FileExt == "" {
			cfg.Backends[i].FileExt = "strip"
		}
	}
}

func validateConfig(cfg *GemplexConfig) (err error) {
	switch cfg.MatchOptions.QueryParams {
	case "include":
	case "remove":
	default:
		return fmt.Errorf("Invalid value for 'query_params' option; valid values are 'remove' and 'include'.")
	}

	switch cfg.MatchOptions.TrailingSlash {
	case "ensure":
	case "remove":
	case "ifpresent":
	default:
		return fmt.Errorf("Invalid value for 'trailing_slash' option.")
	}

	for _, route := range cfg.Routes {
		if route.Backend == "" {
			return fmt.Errorf("Empty backend name in routes.")
		}

		backend := cfg.GetBackendByName(route.Backend)
		if backend == nil {
			return fmt.Errorf("Invalid backend in route: %s", route.Backend)
		}

		if route.Prefix == "" && route.Hostname == "" && route.Url == "" {
			return fmt.Errorf("Route has no pattern.")
		}

		switch {
		case route.Prefix != "" && route.Hostname != "":
			fallthrough
		case route.Prefix != "" && route.Url != "":
			fallthrough
		case route.Hostname != "" && route.Url != "":
			return fmt.Errorf("Multiple patterns in one route.")
		}
	}

	for _, backend := range cfg.Backends {
		if backend.Name == "" {
			return fmt.Errorf("Backend has no name.")
		}

		switch backend.Type {
		case "static":
			if backend.Location == "" {
				return fmt.Errorf("Location missing for static backend.")
			}
			if backend.FileExt != "strip" && backend.FileExt != "include" {
				return fmt.Errorf("Invalid value '%s' for file_ext option; valid values are 'strip' and 'include'.", backend.FileExt)
			}
		case "cgi":
			if backend.Script == "" {
				return fmt.Errorf("Script missing for cgi backend.")
			}
		default:
			return fmt.Errorf("Invalid backend type '%s'; valid values are 'static' and 'cgi'.", backend.Type)
		}
	}

	if len(cfg.Certs) == 0 {
		return fmt.Errorf("No certificates")
	}

	for _, cert := range cfg.Certs {
		if cert.CertFile == "" {
			return fmt.Errorf("Cert file missing.")
		}

		if cert.CertFile == "" {
			return fmt.Errorf("Key file missing.")
		}
	}

	return nil
}
