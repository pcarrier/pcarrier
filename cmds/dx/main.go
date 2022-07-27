package main

import (
	"log"
	"os"
	"path/filepath"
	git_remote_dx "pcarrier.com/git-remote-dx"
	"pcarrier.com/ssh/signatures"
	"syscall"
)

// Looks for not-self in PATH
func findUnderDx() string {
	envPath := os.Getenv("PATH")
	paths := filepath.SplitList(envPath)

	self, err := os.Executable()
	if err != nil {
		log.Fatalf("Could not find our own executable: %v", err)
	}
	under := false
	for _, path := range paths {
		candidate := filepath.Join(path, "dx")
		if candidate == self {
			under = true
			continue
		} else if under {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	log.Fatal("Could not find a dx under PATH")
	return ""
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Subcommand needed")
	}

	cmd := os.Args[1]
	switch cmd {
	case "git-remote-dx":
		git_remote_dx.Run(os.Args[2:])
	case "sig":
		signatures.Run()
	default:
		syscall.Exec(findUnderDx(), os.Args, os.Environ())
	}
}
