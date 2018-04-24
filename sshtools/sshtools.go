package sshtools

import (
	"bytes"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func SSHInteractiveTerminal(sshKey []byte, addr string) error {
	client, err := SSHClient(sshKey, addr)
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

func SSHClient(sshKey []byte, addr string) (*ssh.Client, error) {
	sshSigner, err := ssh.ParsePrivateKey(sshKey)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse ssh private key")
	}

	return ssh.Dial("tcp", addr+":22", &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(sshSigner),
		},
		// TODO: get and save host key from console output after instance creation
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
}

func RemoteCommand(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	defer session.Close()

	if err := session.Run(cmd); err != nil {
		return err
	}

	return nil
}

func UploadFile(client *ssh.Client, source, destination string) error {
	f, err := os.Open(source)
	if err != nil {
		return errors.Wrap(err, "unable to open source file")
	}
	defer f.Close()

	return UploadReader(client, f, destination)
}

func UploadBytes(client *ssh.Client, source []byte, destination string) error {
	return UploadReader(client, bytes.NewReader(source), destination)
}

func UploadReader(client *ssh.Client, source io.Reader, destination string) error {
	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "unable to create ssh session")
	}
	defer session.Close()

	session.Stdin = source
	if err := session.Run("cat > " + shellQuote(destination)); err != nil {
		return err
	}

	return nil
}

func shellQuote(s string) string {
	if strings.ContainsAny(s, `'\`) {
		// TODO
		panic("shell quoting not actually implemented. don't use weird paths")
	}
	return "'" + s + "'"
}
