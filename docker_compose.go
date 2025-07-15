package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/discourse/launcher/v2/config"
	"github.com/discourse/launcher/v2/utils"

	"gopkg.in/yaml.v3"
)

type DockerComposeYaml struct {
	Services map[string]ComposeService
	Volumes  map[string]*interface{}
}
type ComposeService struct {
	Image       string
	Build       ComposeBuild
	Volumes     []string
	Links       []string
	Environment map[string]string
	Ports       []string
}
type ComposeBuild struct {
	Dockerfile string
	Labels     map[string]string
	Shm_Size   string
	Args       []string
	No_Cache   bool
}

func DockerComposeService(config config.Config) ComposeService {
	labels := map[string]string{}
	for k, v := range config.Labels {
		labels[k] = v
	}
	env := map[string]string{}
	env["CREATE_DB_ON_BOOT"] = "1"
	env["MIGRATE_ON_BOOT"] = "1"
	env["PRECOMPILE_ON_BOOT"] = "1"

	for k, v := range config.Env {
		env[k] = v
	}
	links := []string{}
	for _, v := range config.Links {
		links = append(links, v.Link.Name+":"+v.Link.Alias)
	}
	slices.Sort(links)
	volumes := []string{}
	for _, v := range config.Volumes {
		volumes = append(volumes, v.Volume.Host+":"+v.Volume.Guest)
	}
	slices.Sort(volumes)
	ports := []string{}
	ports = append(ports, config.Expose...)
	slices.Sort(ports)

	args := []string{}
	for k := range config.Env {
		if !slices.Contains(utils.KnownSecrets, k) {
			args = append(args, k)
		}
	}
	slices.Sort(args)

	return ComposeService{
		Image: utils.DefaultNamespace + "/" + config.Name,
		Build: ComposeBuild{
			Dockerfile: "./" + config.Name + ".dockerfile",
			Labels:     labels,
			Shm_Size:   "512m",
			Args:       args,
			No_Cache:   true,
		},
		Environment: env,
		Links:       links,
		Volumes:     volumes,
		Ports:       ports,
	}
}

func WriteDockerCompose(configs []config.Config, dir string) error {
	if err := WriteEnvConfig(configs, dir); err != nil {
		return err
	}
	pupsArgs := "--skip-tags=precompile,migrate,db"

	composeServices := map[string]ComposeService{}
	composeVolumes := map[string]*interface{}{}
	for _, config := range configs {
		if err := WriteDockerfile(config, dir, pupsArgs); err != nil {
			return err
		}
		composeServices[config.Name] = DockerComposeService(config)

		regex, err := regexp.Compile(`^[A-Za-z]`)
		if err != nil {
			return err
		}
		for _, v := range config.Volumes {
			// if this is a docker volume (vs a bind mount), add to global volume list
			matched := regex.MatchString(v.Volume.Host)
			if matched {
				composeVolumes[v.Volume.Host] = nil
			}
		}
	}

	compose := &DockerComposeYaml{
		Services: composeServices,
		Volumes:  composeVolumes,
	}

	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	err := encoder.Encode(&compose)
	yaml := b.Bytes()
	if err != nil {
		return errors.New("error marshalling compose file to write docker-compose.yaml")
	}
	if err := os.WriteFile(strings.TrimRight(dir, "/")+"/"+"docker-compose.yaml", yaml, 0660); err != nil {
		return errors.New("error writing compose file docker-compose.yaml")
	}
	return nil
}

func WriteDockerfile(config config.Config, dir string, pupsArgs string) error {
	if err := config.WriteYamlConfig(dir, config.Name+".yaml"); err != nil {
		return err
	}

	file := strings.TrimRight(dir, "/") + "/" + config.Name + ".dockerfile"
	if err := os.WriteFile(file, []byte(config.Dockerfile(pupsArgs, config.Name+".yaml")), 0660); err != nil {
		return errors.New("error writing dockerfile Dockerfile " + file)
	}
	return nil
}

func WriteEnvConfig(configs []config.Config, dir string) error {
	file := strings.TrimRight(dir, "/") + "/.envrc"
	if err := os.WriteFile(file, []byte(ExportEnv(configs)), 0660); err != nil {
		return err
	}
	return nil
}

func ExportEnv(configs []config.Config) string {
	builder := []string{}
	// prioritize the first configs for env
	slices.Reverse(configs)
	for _, config := range configs {
		// Sort env within a config
		configEnv := []string{}
		for k, v := range config.Env {
			val := strings.ReplaceAll(v, "\\", "\\\\")
			val = strings.ReplaceAll(val, "\"", "\\\"")
			configEnv = append(configEnv, "export "+k+"=\""+val+"\"")
		}
		slices.Sort(configEnv)
		builder = append(builder, strings.Join(configEnv, "\n"))
	}
	return strings.Join(builder, "\n")
}

type DockerComposeCmd struct {
	OutputDir string   `name:"output dir" default:"./compose" short:"o" help:"Output dir for docker compose files." predictor:"dir"`
	Config    []string `arg:"" name:"config" help:"config to include in the docker-compose. The first config is assumed to be the main container, and will be the parent folder of the ouput project" predictor:"config"`
}

func (r *DockerComposeCmd) Run(cli *Cli, ctx *context.Context) error {
	if len(r.Config) < 1 {
		return errors.New("empty config array")
	}

	dir := r.OutputDir + "/" + r.Config[0]
	if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	configs := []config.Config{}
	for _, configName := range r.Config {
		config, err := config.LoadConfig(cli.ConfDir, configName, true, cli.TemplatesDir)
		if err != nil {
			return err
		}
		configs = append(configs, *config)
	}
	if err := WriteDockerCompose(configs, dir); err != nil {
		return err
	}
	return nil
}
