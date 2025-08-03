package main

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/discourse/launcher/v2/config"
	"github.com/discourse/launcher/v2/utils"

	"gopkg.in/yaml.v3"
)

type PrintCmd struct {
	Type   string `arg:"" enum:"dockerfile,config,env" help:"print parts of a config"`
	Config string `arg:"" name:"config" help:"config" predictor:"config"`
}

func (r *PrintCmd) Run(cli *Cli) error {
	loadedConfig, err := config.LoadConfig(cli.ConfDir, r.Config, true, cli.TemplatesDir)
	if err != nil {
		return err
	}
	switch r.Type {
	case "dockerfile":
		pupsArgs := "--skip-tags=precompile,migrate,db"
		fmt.Println(loadedConfig.Dockerfile(pupsArgs, "config.yaml"))
	case "env":
		env := map[string]string{}
		for k, v := range loadedConfig.Env {
			if slices.Contains(utils.KnownSecrets, k) {
				continue
			}
			env[k] = v
		}
		var b bytes.Buffer
		encoder := yaml.NewEncoder(&b)
		encoder.SetIndent(2)
		if err := encoder.Encode(&env); err != nil {
			return err
		}
		yaml := b.Bytes()
		fmt.Println(string(yaml))
	default:
		fmt.Println(loadedConfig.Yaml())
	}
	return nil
}
