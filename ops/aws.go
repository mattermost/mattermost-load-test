package ops

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/aws/stscreds"
)

func init() {
	// There may be a way to do this in LoadAWSConfig, rather than by modifying this global, but
	// it's not obvious to me what it is.
	stscreds.DefaultDuration = time.Hour
}

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
