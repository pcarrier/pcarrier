package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type ID [20]byte

var ZeroID = ID{}

func ParseID(str string) (ID, error) {
	slice, err := hex.DecodeString(str)
	if err != nil {
		return ID{}, err
	}
	var id ID
	copy(id[:], slice)
	return id, err
}

func (id ID) String() string {
	return hex.EncodeToString(id[:])
}

type RefUpdate struct {
	From ID
	To   ID
	Ref  string
}

func parseRefUpdate(line string) (*RefUpdate, error) {
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		log.Printf("Parts: %v", parts)
		return nil, errors.New("wrong number of columns")
	}
	from, err := ParseID(parts[0])
	if err != nil {
		return nil, errors.New("Wrong ID format")
	}
	to, err := ParseID(parts[1])
	if err != nil {
		return nil, errors.New("Wrong ID format")
	}
	return &RefUpdate{
		From: from,
		To:   to,
		Ref:  parts[2],
	}, nil
}

func main() {
	log.Printf("env:%v args:%v", os.Environ(), os.Args)

	i := bufio.NewScanner(os.Stdin)

	var refUpdates []*RefUpdate
	for i.Scan() {
		refUpdate, err := parseRefUpdate(i.Text())
		if err != nil {
			log.Fatalf("Could not parse Ref update: %v", err)
		}
		refUpdates = append(refUpdates, refUpdate)
	}

	for _, update := range refUpdates {
		sigStatus, err := update.To.checkSig()
		if err != nil {
			log.Fatalf("Could not check signature: %v", err)
		}
		if sigStatus != SigValid {
			log.Fatalf("Signature for %v is %v", update.Ref, sigStatus.ToString())
		}
	}
}

type SigStatus int8

const (
	SigUnknown SigStatus = iota
	SigAbsent
	SigValid
)

func (s SigStatus) ToString() string {
	switch s {
	case SigValid:
		return "valid"
	case SigAbsent:
		return "absent"
	}
	return "unknown"
}

func (id *ID) checkSig() (SigStatus, error) {
	if *id == ZeroID {
		return SigAbsent, nil
	}

	command := []string{"show", "--show-signature", "--pretty=format:", "--no-patch", id.String()}
	cmd := exec.Command("git", command...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return SigUnknown, err
	}

	go io.Copy(os.Stdout, stderr)

	out, err := cmd.StdoutPipe()
	if err != nil {
		return SigUnknown, err
	}
	if err := cmd.Start(); err != nil {
		return SigUnknown, err
	}
	scanner := bufio.NewScanner(out)
	sigStatus := SigAbsent
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), `Good "git" signature for `) {
			sigStatus = SigValid
		}
	}
	if err := cmd.Wait(); err != nil {
		return SigUnknown, err
	}
	return sigStatus, nil
}
