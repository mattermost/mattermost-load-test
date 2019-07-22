package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate go-bindata -pkg terraform -nocompress -o terraformfile.go ./cluster.tf

const (
	terraformDefaultFilename = "cluster.tf"
	parametersFilename       = "generated.auto.tfvars"
	parametersFilenameJSON   = "generated.auto.tfvars.json"
)

type TerraformEnvironment struct {
	WorkingDirectory  string
	TerraformFilename string
	parameters        *terraformParameters
}

func newTerraformEnvironment(workingDirectory string, parameters *terraformParameters) (*TerraformEnvironment, error) {
	env := TerraformEnvironment{
		WorkingDirectory:  workingDirectory,
		parameters:        parameters,
		TerraformFilename: terraformDefaultFilename,
	}

	if _, err := os.Stat(env.WorkingDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(env.WorkingDirectory, 0700); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	parametersFile := parametersFilename
	if output, err := env.runCommandResult("version"); err != nil {
		return nil, errors.Wrap(err, "Unable to get terraform version")
	} else {
		versionStr := string(output)
		if strings.HasPrefix(versionStr, "Terraform v0.12") {
			// Terraform v0.12 is stricter with it's parameter file names. If the contents is JSON, the filename should end in .json
			// Delete a possibly stale params file and generate a new one with JSON extension
			_ = os.Remove(filepath.Join(env.WorkingDirectory, parametersFile))
			parametersFile = parametersFilenameJSON
		}
	}

	// Create terraform file in working directory
	file, _ := clusterTfBytes()
	err := ioutil.WriteFile(filepath.Join(env.WorkingDirectory, env.TerraformFilename), file, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to write terraform file.")
	}

	// Write parameters
	bytes, err := json.Marshal(parameters)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to marshal terraform parameters.")
	}
	err = ioutil.WriteFile(filepath.Join(env.WorkingDirectory, parametersFile), bytes, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to write parameters file.")
	}

	// Run terraform initialization
	if err := env.runCommand("init", "-input=false"); err != nil {
		return nil, errors.Wrap(err, "Unable to run init command")
	}

	return &env, nil
}

func loadTerraformEnvironment(dir string) (*TerraformEnvironment, error) {
	env := TerraformEnvironment{
		WorkingDirectory: dir,
	}

	return &env, nil
}

func (env *TerraformEnvironment) runCommand(args ...string) error {
	_, err := env.runCommandResult(args...)
	return err
}

func getCmdOutputAndLog(cmd *exec.Cmd) ([]byte, error) {
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	debugLogWriterStdOut := newLogrusWriter(logrus.StandardLogger().WithField("Terraform", "StdOut"), logrus.DebugLevel)
	defer debugLogWriterStdOut.Close()
	debugLogWriterStdErr := newLogrusWriter(logrus.StandardLogger().WithField("Terraform", "StdErr"), logrus.DebugLevel)
	defer debugLogWriterStdErr.Close()
	stdout := io.MultiWriter(debugLogWriterStdOut, &stdoutBuf)
	stderr := io.MultiWriter(debugLogWriterStdErr, &stderrBuf)
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
		wg.Done()
	}()

	_, errStderr = io.Copy(stderr, stderrIn)
	wg.Wait()

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}
	if errStdout != nil || errStderr != nil {
		return nil, fmt.Errorf("failed to capture stdout or stderr\n")
	}

	if len(stderrBuf.Bytes()) > 0 {
		err = fmt.Errorf(string(stderrBuf.Bytes()))
	}

	return stdoutBuf.Bytes(), err
}

func (env *TerraformEnvironment) runCommandResult(args ...string) ([]byte, error) {
	terraformExe := "terraform"
	if env.parameters != nil {
		terraformExe = env.parameters.TerraformPath
	}
	if _, err := exec.LookPath(terraformExe); err != nil {
		return nil, errors.Wrap(err, "Terraform not installed. Please install terraform.")
	}

	cmd := exec.Command(terraformExe, args...)
	cmd.Dir = env.WorkingDirectory
	logrus.Debugf("Running command: [%v] with args: %v", terraformExe, args)
	output, err := getCmdOutputAndLog(cmd)
	if err != nil {
		logrus.Error("Failed CMD Output: " + string(output))
		return output, errors.Wrap(err, fmt.Sprintln("Terraform command failed.", terraformExe, args))
	}

	return output, nil
}

func (env *TerraformEnvironment) apply() error {
	if err := env.runCommand("plan", "-out=tfplan", "-input=false"); err != nil {
		return err
	}
	if err := env.runCommand("apply", "-input=false", "tfplan"); err != nil {
		return err
	}
	return nil
}

func (env *TerraformEnvironment) destroy() error {
	if err := env.runCommand("destroy", "-input=false", "-force"); err != nil {
		return err
	}
	return nil
}

func (env *TerraformEnvironment) getOutputParams() (*terraformOutputParameters, error) {
	output, err := env.runCommandResult("output", "-json")
	if err != nil {
		return nil, err
	}

	var params terraformOutputParameters
	err = json.Unmarshal(output, &params)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to unmarshal terraform output")
	}

	return &params, err
}
