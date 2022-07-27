package git_remote_dx

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
)

var (
	ErrWrongScheme = errors.New("not in dx:// schema")
)

func resolveRef(repo string) (string, string, error) {
	u, err := url.Parse(repo)
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "dx" {
		msg := fmt.Sprintf("not in dx:// schema: %s", repo)
		return "", "", errors.New(msg)
	}
	h := u.Host
	hostParts := strings.Split(h, ".")
	for i, j := 0, len(hostParts)-1; i < j; i, j = i+1, j-1 {
		hostParts[i], hostParts[j] = hostParts[j], hostParts[i]
	}
	pathSpec := "/refs/"
	pathSpec += strings.Join(hostParts, "/")
	pathSpec += u.Path
	return h, pathSpec, nil
}

type Push struct {
	From string
	To   string
}

func (p *Push) ToString() string {
	return fmt.Sprintf("%v -> %v", p.From, p.To)
}

func Run(args []string) {
	repo := args[1]

	host, prefix, err := resolveRef(repo)
	if err != nil {
		log.Fatal(err)
	}

	var gitEnv []string
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GIT_") {
			gitEnv = append(gitEnv, env)
		}
	}

	log.Printf("STARTUP\nargs:%v\nenv:%v\nhost:%v\nprefix:%v", os.Args, gitEnv, host, prefix)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("tracking https://%v/.well-known on %v", host, prefix)

	in := bufio.NewScanner(os.Stdin)
	out := bufio.NewWriter(os.Stdout)

	writeAndFlush := func(str string) error {
		if _, err := out.WriteString(str); err != nil {
			return err
		}
		if err := out.Flush(); err != nil {
			return err
		}
		return nil
	}

	pushes := []Push{}
	for in.Scan() {
		txt := in.Text()
		switch {
		case txt == "capabilities":
			writeAndFlush("refspec\npush\nfetch\n\n")
		case txt == "list":
			writeAndFlush("\n")
		case txt == "list for-push":
			writeAndFlush("\n")
		case strings.HasPrefix(txt, "push "):
			refspec := txt[5:]
			parts := strings.Split(refspec, ":")
			pushes = append(pushes, Push{
				From: parts[0],
				To:   parts[1],
			})
		case txt == "":
			if len(pushes) > 0 {
				log.Printf("pushes: %v", pushes)
			}
			return
		}
	}
}
