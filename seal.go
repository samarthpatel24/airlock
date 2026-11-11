package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	files, err := listFiles(dir, nil)
	if err != nil {
		return err
	}
	if err := needWeights(files); err != nil {
		return err
	}
	m := Manifest{Version: 1, Tool: "airlock " + version, Created: time.Now().UTC().Format(time.RFC3339), Files: files}
	m.Checks = append(m.Checks, "formats: weights are GGUF/safetensors, no pickle, no unknown files")
	fmt.Printf("%s %d %s, safe formats only\n", markOK, len(files), plural(len(files), "file"))

	if *repo != "" {
		if err := checkHF(dir, *repo, *rev, files); err != nil {
			return fmt.Errorf("provenance: %w", err)
		}
		m.Model = &ModelInfo{Repo: *repo, Revision: *rev}
		m.Checks = append(m.Checks, fmt.Sprintf("provenance: every file matches %s@%s", *repo, *rev))
		fmt.Printf("%s matches %s@%s\n", markOK, *repo, (*rev)[:12])
	} else {
		m.Checks = append(m.Checks, "provenance: SKIPPED (no --hf given)")
		fmt.Println(markWarn, "provenance skipped (no --hf)")
	}

	mal, err := malwareScan(dir, *clam)
	if err != nil {
		return err
	}
	m.Checks = append(m.Checks, mal)
	if strings.HasSuffix(mal, "(ClamAV)") {
		fmt.Println(markOK, mal)
	} else {
		fmt.Println(markWarn, mal)
	}

	if err := clearReserved(dir); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, manifestName), &m); err != nil {
		return err
	}
	if err := cosignSign(*key, filepath.Join(dir, manifestName), filepath.Join(dir, signatureName)); err != nil {
		return err
	}
	fmt.Printf("%s sealed %s\n", markOK, dir)
	return nil
}

// offlineSigning is a Sigstore signing config with no services at all: key
// signing, no transparency log, no timestamp authority. Nothing leaves the machine.
const offlineSigning = `{"mediaType":"application/vnd.dev.sigstore.signingconfig.v0.2+json",` +
	`"caUrls":[],"oidcUrls":[],"rekorTlogUrls":[],"tsaUrls":[],` +
	`"rekorTlogConfig":{"selector":"ANY"},"tsaConfig":{"selector":"ANY"}}`

func cosignSign(key, blob, bundle string) error {
	cfg, err := os.CreateTemp("", "airlock-signing-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(cfg.Name())
	if _, err := cfg.WriteString(offlineSigning); err != nil {
		cfg.Close()
		return err
	}
	cfg.Close()
	cmd := exec.Command("cosign", "sign-blob", "--yes", "--key", key, "--bundle", bundle, "--signing-config", cfg.Name(), blob)
	cmd.Stdin = os.Stdin
	var out bytes.Buffer
	if _, set := os.LookupEnv("COSIGN_PASSWORD"); set {
		cmd.Stdout, cmd.Stderr = &out, &out // no prompt coming: show cosign's output only if it fails
	} else {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr // it will prompt for the key password
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cosign sign-blob: %w\n%s", err, out.String())
	}
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
