// airlock is a safety gate for AI models entering air-gapped networks.
// Zarf (or a USB drive) moves the boxes; airlock proves the model inside is
// a safe format, genuine, untampered, and leaves a report for sign-off.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// parseDir accepts the bundle dir before or after the flags.
func parseDir(fs *flag.FlagSet, args []string) (string, error) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		fs.Parse(args[1:])
		if fs.NArg() == 0 {
			return args[0], nil
		}
	} else {
		fs.Parse(args)
		if fs.NArg() == 1 {
			return fs.Arg(0), nil
		}
	}
	return "", errors.New("expected exactly one bundle directory")
}

const version = "0.1.0"

const usage = `airlock ` + version + ` — safety gate for AI models entering air-gapped networks

  airlock check  <dir>                      refuse unsafe or unknown model formats
  airlock seal   <dir> --key cosign.key     check, write manifest, sign it   (online side)
  airlock verify <dir> --key cosign.pub     check signature, hashes, formats (offline side)
  airlock zarf   <dir> --key cosign.pub --bin <airlock-for-target>
                                            write a Zarf component that re-verifies at deploy
  airlock smoke  [--url http://127.0.0.1:8080]
                                            ask the running model a known question, add to report

Run "airlock <command> -h" for flags.`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "check":
		err = checkCmd(os.Args[2:])
	case "seal":
		err = seal(os.Args[2:])
	case "verify":
		err = verify(os.Args[2:])
	case "zarf":
		err = zarfCmd(os.Args[2:])
	case "smoke":
		err = smoke(os.Args[2:])
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "✘", err)
		os.Exit(1)
	}
}
