package main

import (
	"encoding/json"
	"fmt"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Servers []ServerConfig `toml:"servers"`
}

type ServerConfig struct {
	Listen				int							`toml:"listen"`
	ErrorPages			map[string]string			`toml:"error_pages"`
	ServerName			string						`toml:"server_name"`
	ClientMaxBodySize	string						`toml:"client_max_body_size"`
	ClientTimeout		int							`toml:"client_timeout"`
	SessionTimeout		int							`toml:"session_timeout"`
	Locations			map[string]LocationConfig	`toml:"locations"`
}

type LocationConfig struct {
	Root				string		`toml:"root"`
	UploadDir			string		`toml:"upload_dir"`
	Methods				[]string	`toml:"methods"`
	Autoindex			bool		`toml:"autoindex"`
	CGIPath				string		`toml:"cgi_path"`
	CGIExtension		string		`toml:"cgi_extension"`
}

func	LoadConfig(path string) (Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return config, err
	}
	return config, nil
}

func (c Config) PrettyPrint() {
	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		fmt.Println("Error printing config: ", err)
		return
	}
	fmt.Println(string(b))
}
