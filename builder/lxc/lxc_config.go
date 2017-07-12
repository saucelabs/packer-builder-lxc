package lxc

import (
	"io/ioutil"
	"regexp"
	"strings"
)

type lxcConfig struct {
	filePath string
	lines    []string
}

func NewLxcConfig(path string) (*lxcConfig, error) {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(input), "\n")
	return &lxcConfig{path, lines}, nil
}

func (c *lxcConfig) SetRootFs(path string) {
	c.setProp("lxc.rootfs", path)
}

func (c *lxcConfig) setProp(key string, value string) {
	pattern := regexp.MustCompile(`^\s*` + key + `=\s*.*$`)
	for i, line := range c.lines {
		if pattern.MatchString(line) {
			c.lines[i] = key + " = " + value
			return
		}
	}
	c.lines = append(c.lines, key+" = "+value)
}

func (c *lxcConfig) Write(filename string) error {
	output := strings.Join(c.lines, "\n")
	err := ioutil.WriteFile(filename, []byte(output), 0644)
	return err
}
