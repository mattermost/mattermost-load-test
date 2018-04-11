package terraform

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

func generateSSHKey() (privateKeyPEM, authorizedKey []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	authorizedKey = ssh.MarshalAuthorizedKey(publicKey)
	return
}

func generatePassword() (string, error) {
	passwordBytes := make([]byte, 15)
	if _, err := rand.Read(passwordBytes); err != nil {
		return "", errors.Wrap(err, "unable to generate password")
	}
	return base64.RawURLEncoding.EncodeToString(passwordBytes), nil
}
