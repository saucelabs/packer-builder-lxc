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

func (s *stepLxcCreate) Run(state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	errorHandler := func(err error) {
		state.Put("error", err)
		ui.Error(err.Error())
	}

	// TODO: read from env
	lxc_dir := "/var/lib/lxc"
	name := config.ContainerName
	rootfs := filepath.Join(lxc_dir, name, "rootfs")

	if config.PackerForce {
		s.destroy(name, ui)
	}

	if config.LxcTemplate.Name != "" {
		commands := make([][]string, 2)
		commands[0] = append(config.LxcTemplate.EnvVars, []string{"lxc-create", "-n", name, "-t", config.LxcTemplate.Name, "--"}...)
		commands[0] = append(commands[0], config.LxcTemplate.Parameters...)
		// prevent tmp from being cleaned on boot, we put provisioning scripts there
		// TODO: wait for init to finish before moving on to provisioning instead of this
		commands[1] = []string{"touch", filepath.Join(rootfs, "tmp", ".tmpfs")}

		ui.Say("Creating container...")
		for _, command := range commands {
			log.Printf("Executing sudo command: %#v", command)
			if err := s.SudoCommand(command...); err != nil {
				errorHandler(fmt.Errorf("Error creating container: %s", err))
				return multistep.ActionHalt
			}
		}
	} else {
		containerPath := filepath.Join(lxc_dir, name)
		containerConfig, err := NewLxcConfig(config.RootFs.LxcConfig)
		if err != nil {
			errorHandler(fmt.Errorf("Error creating container: %s", err))
			return multistep.ActionHalt
		}
		containerConfig.SetRootFs(rootfs)
		tmpDir, err := ioutil.TempDir("", "lxcconfig")
		if err != nil {
			errorHandler(fmt.Errorf("Error creating container: %s", err))
			return multistep.ActionHalt
		}
		err = containerConfig.Write(filepath.Join(tmpDir, "lxc.config"))
		if err != nil {
			os.RemoveAll(tmpDir)
			errorHandler(fmt.Errorf("Error creating container: %s", err))
			return multistep.ActionHalt
		}

		commands := make([][]string, 3)
		commands[0] = []string{"mkdir", containerPath}
		commands[1] = []string{"tar", "-C", containerPath, "-xf", config.RootFs.Archive}
		commands[2] = []string{"cp", filepath.Join(tmpDir, "lxc.config"), filepath.Join(containerPath, "config")}

		ui.Say("Creating container...")
		for _, command := range commands {
			log.Printf("Executing sudo command: %#v", command)
			if err := s.SudoCommand(command...); err != nil {
				errorHandler(fmt.Errorf("Error creating container: %s", err))
				return multistep.ActionHalt
			}
		}
	}
	ui.Say("Creating container...")
	if err := s.SudoCommand("lxc-start", "-d", "-n", name); err != nil {
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
		err = fmt.Errorf("Sudo command error: %s", stderrString)
	}

	log.Printf("stdout: %s", stdoutString)
	log.Printf("stderr: %s", stderrString)

	return err
}
