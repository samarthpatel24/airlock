package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// seal runs on the online side: check everything, write the manifest, sign it.
func seal(args []string) error {
	fs := flag.NewFlagSet("seal", flag.ExitOnError)
	key := fs.String("key", "", "cosign private key (required)")
	repo := fs.String("hf", "", "Hugging Face repo to check provenance against, e.g. Qwen/Qwen2.5-0.5B-Instruct-GGUF")
	rev := fs.String("rev", "", "pinned 40-char commit for --hf")
	clam := fs.Bool("clamav", false, "also run a ClamAV malware scan (opt-in: needs clamscan and its downloaded database)")
	dir, err := parseDir(fs, args)
	if err != nil {
		return err
	}
	if *key == "" {
		return errors.New("--key is required")
	}

	files, err := listFiles(dir)
	if err != nil {
		return err
	}
	if err := needWeights(files); err != nil {
		return err
	}
	m := Manifest{Version: 1, Tool: "airlock " + version, Created: time.Now().UTC().Format(time.RFC3339), Files: files}
	m.Checks = append(m.Checks, "formats: weights are GGUF/safetensors, no pickle, no unknown files")
	fmt.Printf("✔ %d files, safe formats only\n", len(files))

	if *repo != "" {
		if err := checkHF(dir, *repo, *rev, files); err != nil {
			return fmt.Errorf("provenance: %w", err)
		}
		m.Model = &ModelInfo{Repo: *repo, Revision: *rev}
		m.Checks = append(m.Checks, fmt.Sprintf("provenance: every file matches %s@%s", *repo, *rev))
		fmt.Printf("✔ matches %s@%s\n", *repo, (*rev)[:12])
	} else {
		m.Checks = append(m.Checks, "provenance: SKIPPED (no --hf given)")
		fmt.Println("! provenance skipped (no --hf)")
	}

	mal, err := malwareScan(dir, *clam)
	if err != nil {
		return err
	}
	m.Checks = append(m.Checks, mal)
	fmt.Println(mal)

	if err := clearReserved(dir); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, manifestName), &m); err != nil {
		return err
	}
	// Key-based signing with no transparency-log upload: nothing leaves the machine.
	// ponytail: cosign 3.x marks these two flags deprecated; switch to a
	// --signing-config with no tlog (or cosign's promised offline flag) when they go.
	cmd := exec.Command("cosign", "sign-blob", "--yes", "--key", *key,
		"--bundle", filepath.Join(dir, signatureName),
		"--tlog-upload=false", "--use-signing-config=false",
		filepath.Join(dir, manifestName))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cosign sign-blob: %w", err)
	}
	fmt.Printf("✔ sealed %s\n", dir)
	return nil
}

// malwareScan is opt-in: the scanner and its database come from the internet
// and are one more thing to trust. Off, the report says so. On, it must
// actually run; an explicit request never degrades to a silent skip.
func malwareScan(dir string, enabled bool) (string, error) {
	if !enabled {
		return "malware: not requested (enable with --clamav)", nil
	}
	if _, err := exec.LookPath("clamscan"); err != nil {
		return "", errors.New("--clamav: clamscan not installed")
	}
	out, err := exec.Command("clamscan", "--no-summary", "--infected", "-r", dir).CombinedOutput()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return "malware: clean (ClamAV)", nil
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return "", fmt.Errorf("malware: ClamAV found infected files:\n%s", out)
	default:
		return "", fmt.Errorf("--clamav: ClamAV could not run (database missing? run freshclam): %s", out)
	}
}
