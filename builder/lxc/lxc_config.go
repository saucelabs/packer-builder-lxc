package lxc

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

type LxcConfig struct {
	filePath string
	lines    []string
}

func NewLxcConfig(path string) (*LxcConfig, error) {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(input), "\n")
	return &LxcConfig{path, lines}, nil
}

func (c *LxcConfig) SetRootFs(path string) {
	pattern := regexp.MustCompile(`^lxc.rootfs\s*=\s*.*$`)
	for i, line := range c.lines {
		if pattern.MatchString(line) {
			c.lines[i] = "lxc.rootfs = " + filepath.Join(path, "rootfs")
		}
	}
}

func (c *LxcConfig) Write() error {
	output := strings.Join(c.lines, "\n")
	err := ioutil.WriteFile(c.filePath, []byte(output), 0644)
	return err
}
