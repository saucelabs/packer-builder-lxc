package lxc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/packer/packer"
	"github.com/mitchellh/multistep"
)

type stepLxcCreate struct{}

func (s *stepLxcCreate) createFromTemplate(containerName string, config LxcTemplateConfig) (string, error) {
	lxcDir := "/var/lib/lxc"
	rootfs := filepath.Join(lxcDir, containerName, "rootfs")

	commands := make([][]string, 2)
	commands[0] = append(config.EnvVars, []string{"lxc-create", "-n", containerName, "-t", config.Name, "--"}...)
	commands[0] = append(commands[0], config.Parameters...)
	// prevent tmp from being cleaned on boot, we put provisioning scripts there
	// TODO: wait for init to finish before moving on to provisioning instead of this
	commands[1] = []string{"touch", filepath.Join(rootfs, "tmp", ".tmpfs")}

	err := s.SudoCommands(commands...)
	return rootfs, err
}

func (s *stepLxcCreate) createFromRootFs(containerName string, config RootFsConfig) (string, error) {
	lxcDir := "/var/lib/lxc"
	containerPath := filepath.Join(lxcDir, containerName)
	rootfs := filepath.Join(containerPath, "rootfs")
	containerConfig, err := NewLxcConfig(config.ConfigFile)
	if err != nil {
		err = fmt.Errorf("Could not read lxc config (%s): %s", config.ConfigFile, err)
		return "", err
	}
	containerConfig.SetRootFs(rootfs)
	tmpDir, err := ioutil.TempDir("", "lxcconfig")
	if err != nil {
		err = fmt.Errorf("Could not create temp directory for lxc config (%s): %s", tmpDir, err)
		return rootfs, err
	}
	defer os.RemoveAll(tmpDir)

	err = containerConfig.Write(filepath.Join(tmpDir, "lxc.config"))
	if err != nil {
		err = fmt.Errorf("Could not write lxc config to %s: %s", filepath.Join(tmpDir, "lxc.config"), err)
		return rootfs, err
	}

	commands := make([][]string, 3)
	commands[0] = []string{"mkdir", containerPath}
	commands[1] = []string{"tar", "-C", containerPath, "-xf", config.Archive}
	commands[2] = []string{"cp", filepath.Join(tmpDir, "lxc.config"), filepath.Join(containerPath, "config")}

	err = s.SudoCommands(commands...)
	return rootfs, err
}

func (s *stepLxcCreate) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	errorHandler := func(err error) {
		state.Put("error", err)
		ui.Error(err.Error())
	}

	if config.PackerForce {
		s.destroy(config.ContainerName, ui)
	}

	var rootfs string
	var err error
	if config.LxcTemplate.Name != "" {
		ui.Say("Creating container from template...")
		rootfs, err = s.createFromTemplate(config.ContainerName, config.LxcTemplate)
	} else {
		ui.Say(fmt.Sprintf("Creating container from archive: %s", config.RootFs.Archive))
		rootfs, err = s.createFromRootFs(config.ContainerName, config.RootFs)
	}
	if err != nil {
		errorHandler(err)
		return multistep.ActionHalt
	}
	ui.Say("Starting container...")
	if err = s.SudoCommand("lxc-start", "-d", "-n", config.ContainerName); err != nil {
		errorHandler(fmt.Errorf("Error starting container: %s", err))
		return multistep.ActionHalt
	}

	state.Put("mount_path", rootfs)
	return multistep.ActionContinue
}

func (s *stepLxcCreate) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	s.destroy(config.ContainerName, state.Get("ui").(packer.Ui))
}

func (s *stepLxcCreate) destroy(name string, ui packer.Ui) {
	command := []string{
		"lxc-destroy", "-f", "-n", name,
	}

	ui.Say("Unregistering and deleting virtual machine...")
	if err := s.SudoCommand(command...); err != nil {
		ui.Error(fmt.Sprintf("Error deleting virtual machine: %s", err))
	}
}

func (s *stepLxcCreate) SudoCommand(args ...string) error {
	var stdout, stderr bytes.Buffer

	log.Printf("Executing sudo command: %#v", args)
	cmd := exec.Command("sudo", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	stdoutString := strings.TrimSpace(stdout.String())
	stderrString := strings.TrimSpace(stderr.String())

	if _, ok := err.(*exec.ExitError); ok {
		err = fmt.Errorf("Sudo command (%s) failed with error: %s", args, stderrString)
	}

	log.Printf("stdout: %s", stdoutString)
	log.Printf("stderr: %s", stderrString)

	return err
}

func (s *stepLxcCreate) SudoCommands(commands ...[]string) error {
	for _, command := range commands {
		log.Printf("Executing sudo command: %#v", command)
		if err := s.SudoCommand(command...); err != nil {
			return err
		}
	}
	return nil
}
