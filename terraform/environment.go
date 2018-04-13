package terraform

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//go:generate go-bindata -pkg terraform -nocompress -o terraformfile.go ./cluster.tf

const (
	terraformDefaultFilename = "cluster.tf"
	parametersFilename       = "generated.auto.tfvars"
	terraformCommand         = "terraform"
)

type TerraformEnvironment struct {
	WorkingDirectory  string
	TerraformFilename string
}

func newTerraformEnvironment(workingDirectory string, parameters *terraformParameters) (*TerraformEnvironment, error) {
	env := TerraformEnvironment{
		WorkingDirectory:  workingDirectory,
		TerraformFilename: terraformDefaultFilename,
	}

	if _, err := os.Stat(env.WorkingDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(env.WorkingDirectory, 0700); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
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
	err = ioutil.WriteFile(filepath.Join(env.WorkingDirectory, parametersFilename), bytes, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to write parameters file.")
	}

	// Run terraform initalazation
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

func (env *TerraformEnvironment) runCommandResult(args ...string) ([]byte, error) {
	if _, err := exec.LookPath(terraformCommand); err != nil {
		return nil, errors.Wrap(err, "Terraform not installed. Please install terraform.")
	}

	if err := os.Chdir(env.WorkingDirectory); err != nil {
		return nil, errors.Wrap(err, "Unable to change directory to "+env.WorkingDirectory)
	}
	cmd := exec.Command(terraformCommand, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Error("Failed CMD Ouptut: " + string(output))
		return output, errors.Wrap(err, fmt.Sprintln("Terraform command failed.", terraformCommand, args))
	}
	if err := os.Chdir(".."); err != nil {
		return nil, errors.Wrap(err, "Unable to change directory to "+env.WorkingDirectory)
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
	if err := env.runCommand("destroy", "-input=false", "-auto-approve"); err != nil {
		return err
	}
	return nil
}

func (env *TerraformEnvironment) getOuptutParams() (*terraformOutputParameters, error) {
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
