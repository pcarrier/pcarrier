package signatures

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/42wim/sshsig"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"log"
	"net"
	"os"
)

func signHello() error {
	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		return errors.New("No SSH_AUTH_SOCK environment variable")
	}
	conn, err := net.Dial("unix", sshAuthSock)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not connect to the agent: %v", err))
	}
	defer conn.Close()
	a := agent.NewClient(conn)

	keys, err := a.List()
	if err != nil {
		return errors.New("Could not list keys from the agent")
	}

	var selectedKey *agent.Key
	for _, key := range keys {
		if key.Type() == "ssh-ed25519" {
			selectedKey = key
			break
		}
	}

	if selectedKey == nil {
		return errors.New("Could not find an ed25519 key in the SSH agent")
	}

	sig, err := a.Sign(selectedKey, []byte("hello"))

	log.Printf("signed: %v %v", sig.Format, base64.RawStdEncoding.EncodeToString(sig.Blob))

	sshk, err := ssh.ParsePublicKey(selectedKey.Blob)
	if err != nil {
		log.Fatalf("Couldn't parse public key: %v", err)
	}
	sig2 := sshsig.Armor(sig, sshk, "file")
	if err != nil {
		log.Fatalf("sig2: %v", err)
	}
	log.Printf("signed 2: %s", sig2)

	return nil
}
