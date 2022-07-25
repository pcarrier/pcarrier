package main

import (
	"bufio"
	"errors"
	"github.com/libgit2/git2go/v33"
	"log"
	"os"
	"strings"
)

type Hash = []byte

type RefUpdate struct {
	From git.Oid
	To   git.Oid
	Ref  string
}

func parseRefUpdate(line string) (*RefUpdate, error) {
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		log.Printf("Parts: %v", parts)
		return nil, errors.New("wrong number of columns")
	}
	return &RefUpdate{
		From: git.NewOid(parts[0]),
		To:   git.NewOid(parts[1]),
		Ref:  parts[2],
	}, nil
}

func main() {
	log.Printf("env:%v args:%v", os.Environ(), os.Args)

	root := os.Getenv("GIT_PROJECT_ROOT")

	i := bufio.NewScanner(os.Stdin)

	var refUpdates []*RefUpdate
	for i.Scan() {
		refUpdate, err := parseRefUpdate(i.Text())
		if err != nil {
			log.Fatalf("Could not parse Ref update: %v", err)
		}
		refUpdates = append(refUpdates, refUpdate)
	}

	repo, err := git.OpenRepository(root)
	if err != nil {ssh pc
		log.Fatal("Opening repo: %v", err)
	}

	for _, update := range refUpdates {
		obj, err := repo.LookupCommit(&update.To)
		if err != nil {
			log.Fatalf("Looking up commit %v: %v", update.Ref, err)
		}
		a, b, err := obj.ExtractSignature()
		if err != nil {
			log.Fatalf("Looking up signature for %v: %v", update.Ref, err)
		}
		log.Printf("Signature: %v %v", a, b)
	}
}
