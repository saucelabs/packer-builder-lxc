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

const LxcDir string = "/var/lib/lxc"

type LxcTemplateConfig struct {
	Name       string
	Parameters []string
	EnvVars    []string `mapstructure:"environment_vars"`
}

type RootFsConfig struct {
	ConfigFile string `mapstructure:"config"`
	Archive    string
}

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	ConfigFile          string            `mapstructure:"config_file"`
	OutputDir           string            `mapstructure:"output_directory"`
	ExportConfig        ExportConfig      `mapstructure:"export_config"`
	SidediskFolders     []SidediskFolder  `mapstructure:"sidedisks"`
	ContainerName       string            `mapstructure:"container_name"`
	CommandWrapper      string            `mapstructure:"command_wrapper"`
	RawInitTimeout      string            `mapstructure:"init_timeout"`
	LxcTemplate         LxcTemplateConfig `mapstructure:"lxc_template"`
	RootFs              RootFsConfig      `mapstructure:"rootfs"`
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

type SidediskFolder struct {
	Archive string
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

	if c.LxcTemplate.Name != "" && c.RootFs != (RootFsConfig{}) {
		errs = packer.MultiErrorAppend(errs, fmt.Errorf("Cannot build with both lxc_template and rootfs configuration options"))
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs
	}

	return &c, nil
}
