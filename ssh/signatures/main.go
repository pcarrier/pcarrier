package signatures

import (
	"bufio"
	"bytes"
	"errors"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"strings"
)

type ReadEvent struct {
	text string
	sig  string
}

type Allowed func(pk ssh.PublicKey) bool

type SigStatus int8

const (
	SigUnknown SigStatus = iota
	SigAbsent
	SigValid
	SigInvalid
)

func (s SigStatus) ToString() string {
	switch s {
	case SigValid:
		return "valid"
	case SigAbsent:
		return "absent"
	case SigInvalid:
		return "invalid"
	}
	return "unknown"
}

func Run() {
	if len(os.Args) < 3 {
		log.Print("Expected subcommand.")
		os.Exit(1)
	}
	cmd := os.Args[2]
	switch cmd {
	case "sign":
		err := signHello()
		if err != nil {
			log.Fatalf("Could not sign 'hello' (%v).", err)
		}
	case "check-commit":
		status, err := CheckCommit(func(_ ssh.PublicKey) bool { return true }, os.Stdin)
		if err != nil {
			log.Fatalf("Verification failed (%v).", err)
		}
		if status != SigValid {
			log.Fatalf("Signature is not valid (%v).", status)
		}
	case "check-tag":
		status := CheckTag(func(_ ssh.PublicKey) bool { return true }, os.Stdin)
		if status != SigValid {
			log.Fatalf("Signature is not valid (%v).", status)
		}
	default:
		log.Fatalf("Unknown command %v.", cmd)
	}
}

func CheckCommit(allowed Allowed, in io.Reader) (SigStatus, error) {
	s := bufio.NewScanner(in)

	headers := []string{}
	sigbuf := new(bytes.Buffer)
	sigw := bufio.NewWriter(sigbuf)
	bodbuf := new(bytes.Buffer)
	bodw := bufio.NewWriter(bodbuf)
	inHeaders := true
	inSig := false
	for s.Scan() {
		line := s.Text()
		switch {
		case line == "":
			inHeaders = false
		case strings.HasPrefix(line, "gpgsig "):
			inSig = true
			_, _ = sigw.Write([]byte(line[7:] + "\n"))
		case inSig && strings.HasPrefix(line, " "):
			_, _ = sigw.Write([]byte(line[1:] + "\n"))
		case inHeaders:
			headers = append(headers, line)
		default:
			_, _ = bodw.Write([]byte(s.Text() + "\n"))
		}
	}
	if err := sigw.Flush(); err != nil {
		return SigAbsent, err
	}
	if err := bodw.Flush(); err != nil {
		return SigAbsent, err
	}

	sig, err := Decode(sigbuf.Bytes())
	if err != nil {
		return SigInvalid, errors.New("could not decode signature")
	}
	if sig.Namespace != "git" {
		return SigInvalid, errors.New("invalid namespace")
	}
	if !allowed(sig.PK) {
		return SigInvalid, errors.New("key is not allowed")
	}

	rawMsg := strings.Join(headers, "\n") + "\n\n" + bodbuf.String()
	if err := Verify(strings.NewReader(rawMsg), sig); err != nil {
		return SigInvalid, err
	}
	return SigValid, nil
}

func CheckTag(allowed Allowed, in io.Reader) SigStatus {
	events := make(chan *ReadEvent)
	s := bufio.NewScanner(in)
	go func() {
		var accumulatedText []string
		for s.Scan() {
			line := s.Text()
			if line != "-----BEGIN SSH SIGNATURE-----" {
				accumulatedText = append(accumulatedText, line)
				continue
			}
			re := ReadEvent{text: strings.Join(accumulatedText, "\n")}
			accumulatedText = []string{line}
			for s.Scan() {
				line := s.Text()
				accumulatedText = append(accumulatedText, line)
				if line == "-----END SSH SIGNATURE-----" {
					stringTxt := strings.Join(accumulatedText, "\n")
					accumulatedText = []string{}
					re.sig = stringTxt
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
		sig, err := Decode([]byte(evt.sig))
		if err != nil {
			return SigInvalid
		}
		if sig.Namespace != "git" {
			return SigInvalid
		}
		if !allowed(sig.PK) {
			return SigInvalid
		}
		err = Verify(strings.NewReader(evt.text+"\n"), sig)
		if err != nil {
			return SigInvalid
		}
	}
	return SigValid
}
