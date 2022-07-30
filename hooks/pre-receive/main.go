package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"os"
	"os/exec"
	"pcarrier.com/ssh/signatures"
	"strconv"
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

	args := []string{"show-ref"}
	cmd := exec.Command("git", args...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Couldn't show-ref (%v).", err)
	}
	log.Printf("%v %v", strconv.Quote(outb.String()), strconv.Quote(errb.String()))

	for _, update := range refUpdates {
		log.Printf("%v: %v -> %v", update.Ref, update.From, update.To)
		if !strings.HasPrefix(update.Ref, "refs/") {
			log.Fatalf("ref does not start with refs/: %s", update.Ref)
		}
		parts := strings.Split(update.Ref, "/")
		parts = parts[0 : len(parts)-1]
		for cutAt := len(parts); cutAt > 0; cutAt-- {
			log.Printf("Looking for %s/@meta", strings.Join(parts[:cutAt], "/"))
		}

		sigStatus, err := update.checkSig()
		if err != nil {
			log.Fatalf("Could not check signature: %v", err)
		}
		if sigStatus != signatures.SigValid {
			log.Fatalf("Signature for %v is %v", update.Ref, sigStatus.ToString())
		}
	}
}

type ObjectType int8

const (
	InvalidObject ObjectType = 0
	CommitObject  ObjectType = 1
	TreeObject    ObjectType = 2
	BlobObject    ObjectType = 3
	TagObject     ObjectType = 4
	// 5 reserved for future expansion
	OFSDeltaObject ObjectType = 6
	REFDeltaObject ObjectType = 7

	AnyObject ObjectType = -127
)

func getType(id ID) (ObjectType, error) {
	cmd := exec.Command("git", "cat-file", "-t", id.String())
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return InvalidObject, err
	}
	if errb.Len() != 0 {
		return InvalidObject, err
	}

	typ := outb.String()
	switch typ {
	case "tree\n":
		return TreeObject, nil
	case "commit\n":
		return CommitObject, nil
	case "tag\n":
		return TagObject, nil
	case "blob\n":
		return BlobObject, nil
	default:
		return InvalidObject, errors.New(fmt.Sprintf("unknown type %s", typ))
	}
}

func (ru RefUpdate) checkSig() (signatures.SigStatus, error) {
	if ru.To == ZeroID {
		return signatures.SigAbsent, nil
	}

	typ, err := getType(ru.To)
	if err != nil {
		return signatures.SigAbsent, err
	}
	switch typ {
	case TagObject:
		cmd := exec.Command("git", "cat-file", "tag", ru.To.String())
		var outb bytes.Buffer
		cmd.Stdout = &outb
		err := cmd.Run()
		if err != nil {
			log.Fatalf("Couldn't show tag %v (%v).", ru.Ref, err)
		}
		sigStatus := signatures.CheckTag(func(pk ssh.PublicKey) bool {
			return true
		}, bytes.NewReader(outb.Bytes()))
		return sigStatus, nil
	case CommitObject:
		cmd := exec.Command("git", "cat-file", "commit", ru.To.String())
		var outb bytes.Buffer
		cmd.Stdout = &outb
		err := cmd.Run()
		if err != nil {
			log.Fatalf("Couldn't show commit %v (%v).", ru.Ref, err)
		}
		status, err := signatures.CheckCommit(func(pk ssh.PublicKey) bool {
			return true
		}, bytes.NewReader(outb.Bytes()))
		if err != nil {
			log.Fatalf("Couldn't check status (%v).", err)
		}
		return status, nil
	default:
		return signatures.SigAbsent, errors.New(fmt.Sprintf("unsupported object type %v", typ))
	}

	return signatures.SigAbsent, nil
}
