package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	ConfigFilePath = "config.json"
)

type Route struct {
	Hostname string `json:"hostname"`
	Prefix   string `json:"prefix"`
	Regex    string `json:"regex"`
	Url      string `json:"url"`
	CatchAll bool   `json:"catch_all"`
	Upstream string `json:"upstream"`
}

type Upstream struct {
	Name     string `json:"name"`
	Addr     string `json:"addr"`
	Location string `json:"location"`
	Cgi      string `json:"cgi"`
	Tls      bool   `json:"tls"`
}

type Cert struct {
	Hostname string `json:"hostname"`
	CertFile string `json:"cert"`
	KeyFile  string `json:"key"`
}

type GemplexConfig struct {
	ListenAddr string     `json:"listen"`
	Routes     []Route    `json:"routes"`
	Upstreams  []Upstream `json:"upstreams"`
	Certs      []Cert     `json:"certs"`
}

func (u *Upstream) UnmarshalJson(data []byte) (err error) {
	// this custom unmarshaller is intended to fill in the default values.
	//
	// the alias here is needed in order to prevent infinite recursive calls to
	// this method.
	type UpstreamAlias Upstream
	upstream := &UpstreamAlias{
		Tls: true,
	}

	err = json.Unmarshal(data, upstream)
	if err != nil {
		return
	}

	*u = Upstream(*upstream)
	return
}

func Load() (config GemplexConfig, err error) {
	f, err := os.Open(ConfigFilePath)
	if err != nil {
		return
	}
	defer f.Close()

	// set top-level default values
	config.ListenAddr = "127.0.0.1:1965"

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)

	if err == nil {
		err = validateConfig(&config)
	}

	return
}

func (cfg *GemplexConfig) GetUpstreamByName(name string) *Upstream {
	for _, upstream := range cfg.Upstreams {
		if upstream.Name == name {
			return &upstream
		}
	}

	return nil
}

func (cfg *GemplexConfig) GetUpstreamByHostname(hostname string) *Upstream {
	for _, route := range cfg.Routes {
		if route.Hostname == hostname {
			for _, upstream := range cfg.Upstreams {
				if upstream.Name == route.Upstream {
					return &upstream
				}
			}

			// this should not happen, because we validate the upstream names
			// when loading config.
			panic(fmt.Sprintf("No such upstream found: %s", route.Upstream))
		}
	}

	return nil
}

func validateConfig(cfg *GemplexConfig) (err error) {
	for _, route := range cfg.Routes {
		upstream := cfg.GetUpstreamByName(route.Upstream)
		if upstream == nil {
			return fmt.Errorf("Invalid upstream in route: %s", route.Upstream)
		}
	}

	return nil
}
