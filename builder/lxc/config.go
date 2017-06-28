package lxc

import (
	"fmt"
	"time"

	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/template/interpolate"
	"github.com/mitchellh/mapstructure"
)

type LxcTemplateConfig struct {
	Name       string
	Parameters []string
	EnvVars    []string
}

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	ConfigFile          string            `mapstructure:"config_file"`
	OutputDir           string            `mapstructure:"output_directory"`
	ExportConfig        ExportConfig      `mapstructure:"export_config"`
	ContainerName       string            `mapstructure:"container_name"`
	CommandWrapper      string            `mapstructure:"command_wrapper"`
	RawInitTimeout      string            `mapstructure:"init_timeout"`
	CloneSource         string            `mapstructure:"clone_container"`
	CleanupFirst        bool              `mapstructure:"cleanup_first"`
	LxcTemplate         LxcTemplateConfig `mapstructure:"lxc_template"`
	TargetRunlevel      int               `mapstructure:"target_runlevel"`
	InitTimeout         time.Duration

	ctx interpolate.Context
}

type ExportConfig struct {
	Filename string
	Folders  []ExportFolder `mapstructure:"folders"`
}

type ExportFolder struct {
	Src  string
	Dest string
}

func NewConfig(raws ...interface{}) (*Config, error) {
	var c Config

	var md mapstructure.Metadata
	err := config.Decode(&c, &config.DecodeOpts{
		Metadata:    &md,
		Interpolate: true,
	}, raws...)
	if err != nil {
		return nil, err
	}

	// Accumulate any errors
	var errs *packer.MultiError

	if c.OutputDir == "" {
		c.OutputDir = fmt.Sprintf("output-%s", c.PackerBuildName)
	}

	if c.ContainerName == "" {
		c.ContainerName = fmt.Sprintf("packer-%s", c.PackerBuildName)
	}

	if c.CommandWrapper == "" {
		c.CommandWrapper = "{{.Command}}"
	}

	if c.RawInitTimeout == "" {
		c.RawInitTimeout = "20s"
	}

	c.InitTimeout, err = time.ParseDuration(c.RawInitTimeout)
	if err != nil {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("Failed parsing init_timeout: %s", err))
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs
	}

	return &c, nil
}
