package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/42wim/sshsig"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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

type ReadEvent struct {
	text string
	sig  *sshsig.Signature
}

func main() {
	if len(os.Args) < 2 {
		log.Print("Expected subcommand.")
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "sign":
		err := signHello()
		if err != nil {
			log.Fatalf("Could not sign 'hello' (%v).", err)
		}
	case "read":
		events := make(chan *ReadEvent)
		s := bufio.NewScanner(os.Stdin)
		go func() {
			var accumulatedText []string
			for s.Scan() {
				line := s.Text()
				if line != "-----BEGIN SSH SIGNATURE-----" {
					accumulatedText = append(accumulatedText, line)
					continue
				}
				re := ReadEvent{text: strings.Join(accumulatedText, "\n")}
				var accumulatedSig = []string{line}
				for s.Scan() {
					sigLine := s.Text()
					accumulatedSig = append(accumulatedSig, sigLine)
					if sigLine == "-----END SSH SIGNATURE-----" {
						stringTxt := strings.Join(accumulatedSig, "\n")
						sig, err := sshsig.Decode([]byte(stringTxt))
						if err != nil {
							log.Panicf("Invalid sig (%v).", err)
						}
						re.sig = sig
						break
					}
				}
				events <- &re
			}
			finalTxt := strings.Join(accumulatedText, "\n")
			if len(finalTxt) > 0 {
				events <- &ReadEvent{text: finalTxt}
			}
			close(events)
		}()
		for evt := range events {
			log.Printf("%s %v", strconv.Quote(evt.text), evt.sig)
		}
	default:
		log.Fatalf("unknown command %v", cmd)
	}
}
