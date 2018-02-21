package ops

import (
	"os"
	"os/signal"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func SSH(clusterInfo *ClusterInfo, instance *ec2.Instance) error {
	client, err := sshClient(clusterInfo, instance)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	stdinFd := int(os.Stdin.Fd())
	if terminal.IsTerminal(stdinFd) {
		originalState, err := terminal.MakeRaw(stdinFd)
		if err != nil {
			return err
		}
		defer terminal.Restore(stdinFd, originalState)

		w, h, err := terminal.GetSize(stdinFd)
		if err != nil {
			return err
		}

		if err := session.RequestPty("xterm-256color", h, w, ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}); err != nil {
			return err
		}
	}

	c := make(chan os.Signal, 1000)
	signal.Notify(c, os.Interrupt)
	defer signal.Stop(c)

	if err := session.Shell(); err != nil {
		return err
	}

	stop := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case sig := <-c:
				switch sig {
				case os.Interrupt:
					session.Signal(ssh.SIGINT)
				}
			case <-stop:
				return
			}
		}
	}()
	defer func() {
		close(stop)
		<-stopped
	}()

	return session.Wait()
}

func sshClient(clusterInfo *ClusterInfo, instance *ec2.Instance) (*ssh.Client, error) {
	sshSigner, err := ssh.ParsePrivateKey(clusterInfo.SSHKey)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse ssh private key")
	}

	return ssh.Dial("tcp", aws.StringValue(instance.PublicIpAddress)+":22", &ssh.ClientConfig{
		User: "ec2-user",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshSigner),
		},
		// TODO: get and save host key from console output after instance creation
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
}
