package lxc

import (
	"io/ioutil"
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
	c.SetProp("lxc.rootfs", path)
}

func (c *LxcConfig) SetProp(key string, value string) {
	pattern := regexp.MustCompile(`^\s*` + key + `=\s*.*$`)
	for i, line := range c.lines {
		if pattern.MatchString(line) {
			c.lines[i] = key + " = " + value
			return
		}
	}
	c.lines = append(c.lines, key+" = "+value)
}

func (c *LxcConfig) Write(filename string) error {
	output := strings.Join(c.lines, "\n")
	err := ioutil.WriteFile(filename, []byte(output), 0644)
	return err
}
