// airlock is a safety gate for AI models entering air-gapped networks.
// Zarf (or a USB drive) moves the boxes; airlock proves the model inside is
// a safe format, genuine, untampered, and leaves a report for sign-off.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// parseDir accepts the bundle dir anywhere among the flags, and resolves
// symlinks in its path (external drives are often reached through one).
func parseDir(fs *flag.FlagSet, args []string) (string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return "", err
		}
		if fs.NArg() == 0 {
			break
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(pos) != 1 {
		return "", errors.New("expected exactly one bundle directory")
	}
	return filepath.EvalSymlinks(pos[0])
}

// outside refuses an output path inside the bundle: the file would show up
// as unexpected and fail every later verify.
func outside(dir, path, flagName string) error {
	p, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if parent, err := filepath.EvalSymlinks(filepath.Dir(p)); err == nil {
		p = filepath.Join(parent, filepath.Base(p))
	}
	d, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if rel, err := filepath.Rel(d, p); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s %q is inside the bundle; write it somewhere else", flagName, path)
	}
	return nil
}

// Status marks are coloured only on a terminal, and never when NO_COLOR is set.
var (
	markOK   = mark(os.Stdout, "✔", "32")
	markWarn = mark(os.Stdout, "!", "33")
	markBad  = mark(os.Stderr, "✘", "31")
)

func mark(f *os.File, s, color string) string {
	if _, off := os.LookupEnv("NO_COLOR"); off {
		return s
	}
	if st, err := f.Stat(); err != nil || st.Mode()&os.ModeCharDevice == 0 {
		return s
	}
	return "\x1b[" + color + "m" + s + "\x1b[0m"
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
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
		fmt.Fprintln(os.Stderr, markBad, err)
		os.Exit(1)
	}
}
