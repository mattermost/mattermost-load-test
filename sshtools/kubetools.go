package sshtools

import (
	"os"
	"os/exec"
)

// SSHInteractiveKubesPod will create an interactive shell for the given pod.
func SSHInteractiveKubesPod(podName string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pa := os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Dir:   cwd,
	}

	kubectl, err := exec.LookPath("kubectl")
	if err != nil {
		return err
	}

	proc, err := os.StartProcess(kubectl, []string{"kubectl", "exec", "-it", podName, "--", "/bin/bash"}, &pa)
	if err != nil {
		return err
	}

	_, err = proc.Wait()
	if err != nil {
		return err
	}

	return nil
}
