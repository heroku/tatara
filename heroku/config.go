package heroku

import (
	"path/filepath"
	"os"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"strings"
	"fmt"
)

type Config struct {
	Build BuildConfig
}

type BuildConfig struct {
	Buildpacks []string
	Packages   []string
	Pre        []string
	Post       []string
	Config     map[string]string
}

func ReadConfig(appDir string) (Config, error) {
	herokuYamlFile := filepath.Join(appDir, "heroku.yml")
	_, err := os.Stat(herokuYamlFile)
	if err == nil {
		configBytes, err := ioutil.ReadFile(herokuYamlFile)
		if err == nil {
			var herokuConfig Config
			yaml.Unmarshal(configBytes, &herokuConfig)
			return herokuConfig, nil
		}
	}
	return Config{}, err
}

func (c *Config) ResolveBuildpacks() []string {
	buildpacks := make([]string, len(c.Build.Buildpacks))
	for i, buildpack := range c.Build.Buildpacks {
		if strings.HasPrefix(buildpack, "https://") || strings.HasPrefix(buildpack, "http://"){
			buildpacks[i] = buildpack
		} else {
			buildpacks[i] = fmt.Sprintf("https://buildpack-registry.s3.amazonaws.com/buildpacks/%s.tgz", buildpack)
		}
	}
	return buildpacks
}

func (c *Config) ConstructDockerfile(stack string) string {
	dockerfile := fmt.Sprintf("FROM %s", stack);
	for _, command := range c.Build.Pre {
		dockerfile += fmt.Sprintf(`
RUN %s`, command)
	}
	if len(c.Build.Packages) > 0 {
		dockerfile += fmt.Sprintf(`
RUN apt-get update`)
	}
	for _, aptPackage := range c.Build.Packages {
		dockerfile += fmt.Sprintf(`
RUN apt-get install %s -y`, aptPackage)
	}
	for _, command := range c.Build.Post {
		dockerfile += fmt.Sprintf(`
RUN %s`, command)
	}
	return dockerfile
}