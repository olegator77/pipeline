package compiler

import (
	"fmt"

	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/frontend"
	"github.com/cncd/pipeline/pipeline/frontend/yaml"
	// libcompose "github.com/docker/libcompose/yaml"
)

// TODO(bradrydzewski) compiler should handle user-defined volumes from YAML
// TODO(bradrydzewski) compiler should handle user-defined networks from YAML

// Compiler compiles the yaml
type Compiler struct {
	local     bool
	escalated []string
	prefix    string
	volumes   []string
	env       map[string]string
	base      string
	path      string
	metadata  frontend.Metadata
	aliases   []string
}

// New creates a new Compiler with options.
func New(opts ...Option) *Compiler {
	compiler := new(Compiler)
	compiler.env = map[string]string{}
	for _, opt := range opts {
		opt(compiler)
	}
	return compiler
}

// Compile compiles the YAML configuration to the pipeline intermediate
// representation configuration format.
func (c *Compiler) Compile(conf *yaml.Config) *backend.Config {
	config := new(backend.Config)

	// create a default volume
	config.Volumes = append(config.Volumes, &backend.Volume{
		Name:   fmt.Sprintf("%s_default", c.prefix),
		Driver: "local",
	})

	// create a default network
	config.Networks = append(config.Networks, &backend.Network{
		Name:   fmt.Sprintf("%s_default", c.prefix),
		Driver: "bridge",
	})

	// overrides the default workspace paths when specified
	// in the YAML file.
	if len(conf.Workspace.Base) != 0 {
		c.base = conf.Workspace.Base
	}
	if len(conf.Workspace.Path) != 0 {
		c.path = conf.Workspace.Path
	}

	// add default clone step
	if c.local == false && len(conf.Clone.Containers) == 0 {
		container := &yaml.Container{
			Name:  "clone",
			Image: "plugins/git:latest",
			Vargs: map[string]interface{}{"depth": "0"},
		}
		name := fmt.Sprintf("%s_clone", c.prefix)
		step := c.createProcess(name, container, conf.Platform)

		stage := new(backend.Stage)
		stage.Name = name
		stage.Alias = "clone"
		stage.Steps = append(stage.Steps, step)

		config.Stages = append(config.Stages, stage)
	} else if c.local == false {
		for i, container := range conf.Clone.Containers {
			if !container.Constraints.Match(c.metadata) {
				continue
			}
			stage := new(backend.Stage)
			stage.Name = fmt.Sprintf("%s_clone_%v", c.prefix, i)
			stage.Alias = container.Name

			name := fmt.Sprintf("%s_clone_%d", c.prefix, i)
			step := c.createProcess(name, container, conf.Platform)
			stage.Steps = append(stage.Steps, step)

			config.Stages = append(config.Stages, stage)
		}
	}

	// add services steps
	if len(conf.Services.Containers) != 0 {
		stage := new(backend.Stage)
		stage.Name = fmt.Sprintf("%s_services", c.prefix)
		stage.Alias = "services"

		for _, container := range conf.Services.Containers {
			c.aliases = append(c.aliases, container.Name)
		}

		for i, container := range conf.Services.Containers {
			name := fmt.Sprintf("%s_services_%d", c.prefix, i)
			step := c.createProcess(name, container, conf.Platform)
			stage.Steps = append(stage.Steps, step)

		}
		config.Stages = append(config.Stages, stage)
	}

	// add pipeline steps. 1 pipeline step per stage, at the moment
	var stage *backend.Stage
	var group string
	for i, container := range conf.Pipeline.Containers {
		if !container.Constraints.Match(c.metadata) {
			continue
		}

		if stage == nil || group != container.Group || container.Group == "" {
			group = container.Group

			stage = new(backend.Stage)
			stage.Name = fmt.Sprintf("%s_stage_%v", c.prefix, i)
			stage.Alias = container.Name
			config.Stages = append(config.Stages, stage)
		}

		name := fmt.Sprintf("%s_step_%d", c.prefix, i)
		step := c.createProcess(name, container, conf.Platform)
		stage.Steps = append(stage.Steps, step)
	}

	return config
}

// func setupNetwork(step *backend.Step, network *libcompose.Network) {
// 	step.Networks = append(step.Networks, backend.Conn{
// 		Name: network.Name,
// 		// Aliases:
// 	})
// }
//
// func setupVolume(step *backend.Step, volume *libcompose.Volume) {
// 	step.Volumes = append(step.Volumes, volume.String())
// }
//
// var (
// 	// Default plugin used to clone the repository.
// 	defaultCloneImage = "plugins/git:latest"
//
// 	// Default plugin settings used to clone the repository.
// 	defaultCloneVargs = map[string]interface{}{
// 		"depth": 0,
// 	}
// )
//
// // defaultClone returns the default step for cloning an image.
// func defaultClone() *yaml.Container {
// 	return &yaml.Container{
// 		Image: defaultCloneImage,
// 		Vargs: defaultCloneVargs,
// 	}
// }
