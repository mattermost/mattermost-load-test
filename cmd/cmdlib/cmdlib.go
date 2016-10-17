// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package cmdlib

import (
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
)

type GlobalCommandConfig struct {
	PrettyPrint bool
}

type CommandContext struct {
	LoadTestConfig      *loadtestconfig.LoadTestConfig
	GlobalCommandConfig *GlobalCommandConfig
}

func MakeCommandContext() *CommandContext {
	var config struct {
		GlobalCommandConfig GlobalCommandConfig
	}
	err := loadtestconfig.UnmarshalConfigStruct(&config)
	if err != nil {
		panic(err)
	}

	loadTestConfig := loadtestconfig.GetConfig()

	return &CommandContext{
		LoadTestConfig:      loadTestConfig,
		GlobalCommandConfig: &config.GlobalCommandConfig,
	}
}

func (c *CommandContext) PrettyPrintln(a ...interface{}) (int, error) {
	if c.GlobalCommandConfig.PrettyPrint {
		return fmt.Println(a...)
	}
	return 0, nil
}

func (c *CommandContext) Println(a ...interface{}) (int, error) {
	return fmt.Println(a...)
}

func (c *CommandContext) Print(a ...interface{}) (int, error) {
	return fmt.Print(a...)
}

func (c *CommandContext) PrintErrorln(a ...interface{}) (int, error) {
	return fmt.Fprintln(os.Stderr, a...)
}

func (c *CommandContext) PrintError(a ...interface{}) (int, error) {
	return fmt.Fprint(os.Stderr, a...)
}

func (c *CommandContext) PrintErrors(errors []error) {
	if len(errors) > 0 {
		c.PrintErrorln("Errors where encountered: ")
		for _, err := range errors {
			if err != nil {
				c.PrintErrorln(err.Error())
			}
		}
		c.PrintErrorln("------------ End Errors")
	}
}

func (c *CommandContext) PrintResultsHeader() {
	c.PrettyPrintln("Results")
	c.PrettyPrintln("=========")
}

func GetClient(config *loadtestconfig.ConnectionConfiguration) (*model.Client, error) {
	client := model.NewClient(config.ServerURL)
	_, err := client.Login(config.AdminEmail, config.AdminPassword)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func GetUserClient(config *loadtestconfig.ConnectionConfiguration, user *loadtestconfig.ServerStateUser) *model.Client {
	client := model.NewClient(config.ServerURL)
	client.AuthToken = user.SessionToken
	client.AuthType = model.HEADER_BEARER
	return client
}
