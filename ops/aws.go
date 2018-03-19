package ops

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

func LoadAWSConfig() (aws.Config, error) {
	return external.LoadDefaultAWSConfig(external.WithMFATokenFunc(MFATokenPrompt))
}

func MFATokenPrompt() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("MFA Token: ")
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}
