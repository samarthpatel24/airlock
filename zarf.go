package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	zarfNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	// The key path is written into a shell command, so only plain absolute paths.
	keyPathRe = regexp.MustCompile(`^/[A-Za-z0-9._/-]+$`)
)

// zarfCmd writes a Zarf component that carries a sealed bundle into the
// air gap and runs `airlock verify` there at deploy time. Zarf deploys
// components in order and stops on the first failed action, so anything
// placed after this component (the model server) never deploys if the
// bundle was tampered with.
//
// Trust anchors never travel with the data they check: the public key is
// NOT shipped (the inside host holds it, installed out of band), and by
// default neither is the airlock binary.
func zarfCmd(args []string) error {
	fs := flag.NewFlagSet("zarf", flag.ExitOnError)
	key := fs.String("key", "", "cosign public key, to verify the bundle now (required; not shipped)")
	insideKey := fs.String("inside-key", "/etc/airlock/cosign.pub", "absolute path of the trusted key on the deploy host")
	bin := fs.String("bin", "", "optional: ship this airlock binary (only safe if the Zarf package signature is verified at deploy)")
	name := fs.String("name", "airlock-model", "Zarf package name")
	out := fs.String("out", "zarf.yaml", "where to write the Zarf package config")
	dir, err := parseDir(fs, args)
	if err != nil {
		return err
	}
	if *key == "" {
		return errors.New("--key is required")
	}
	if *bin != "" {
		fmt.Println("! shipping the verifier inside the package: deploy with Zarf signature verification (zarf package deploy --key)")
	}
	if !zarfNameRe.MatchString(*name) {
		return fmt.Errorf("--name %q: use lowercase letters, digits and '-'", *name)
	}
	if !keyPathRe.MatchString(*insideKey) {
		return fmt.Errorf("--inside-key %q: must be a plain absolute path", *insideKey)
	}

	// Never package something that would fail inside.
	r := report{Dir: dir}
	if err := doVerify(dir, *key, &r); err != nil {
		return fmt.Errorf("bundle does not verify, refusing to package: %w", err)
	}
	y, err := zarfYAML(r.M, dir, *insideKey, *bin, *name, filepath.Dir(*out))
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, []byte(y), 0o644); err != nil {
		return err
	}
	fmt.Printf("✔ bundle verified, wrote %s\n", *out)
	fmt.Printf("  inside host must already hold the trusted key at %s\n", *insideKey)
	fmt.Println("  next: zarf package create", filepath.Dir(*out), " — or import it from your own zarf.yaml")
	return nil
}

func zarfYAML(m *Manifest, dir, insideKey, bin, name, base string) (string, error) {
	// Source paths are relative to the zarf.yaml, as Zarf resolves them.
	rel := func(p string) (string, error) {
		a, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		b, err := filepath.Abs(base)
		if err != nil {
			return "", err
		}
		r, err := filepath.Rel(b, a)
		return filepath.ToSlash(r), err
	}
	q := func(s string) string { b, _ := json.Marshal(s); return string(b) } // JSON strings are valid YAML

	var b strings.Builder
	file := func(src, target, shasum string, exec bool) error {
		s, err := rel(src)
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "      - source: %s\n        destination: %s\n", q(s), q(target))
		if shasum != "" {
			fmt.Fprintf(&b, "        checksum: sha256:%s\n", shasum)
		}
		if exec {
			b.WriteString("        executable: true\n")
		}
		return nil
	}

	desc := "AI model bundle verified offline by airlock"
	if m.Model != nil {
		desc = fmt.Sprintf("%s@%s, verified offline by airlock", m.Model.Repo, m.Model.Revision[:12])
	}
	fmt.Fprintf(&b, "apiVersion: zarf.dev/v1beta1\nkind: ZarfPackageConfig\nmetadata:\n  name: %s\n  description: %s\n  version: %s\n", name, q(desc), q(version))
	b.WriteString("components:\n  - name: airlock-verify\n") // v1beta1 components are required by default
	b.WriteString("    description: \"Verify the model bundle before anything after this component deploys\"\n    files:\n")

	verifier := "airlock" // pre-installed, trusted copy on the deploy host
	if bin != "" {
		if err := file(bin, "airlock/bin/airlock", "", true); err != nil {
			return "", err
		}
		verifier = "./airlock/bin/airlock"
	}
	var extra []FileEntry
	for _, p := range []string{manifestName, signatureName} {
		sum, _, err := hashFile(filepath.Join(dir, p))
		if err != nil {
			return "", err
		}
		extra = append(extra, FileEntry{Path: p, SHA256: sum})
	}
	for _, f := range append(extra, m.Files...) {
		if err := file(filepath.Join(dir, filepath.FromSlash(f.Path)), path.Join("airlock/bundle", f.Path), f.SHA256, false); err != nil {
			return "", err
		}
	}
	b.WriteString("    actions:\n      onDeploy:\n        onSuccess:\n")
	fmt.Fprintf(&b, "          - cmd: %s verify ./airlock/bundle --key %s --report ./airlock/airlock-report.md\n", verifier, insideKey)
	b.WriteString("            description: \"airlock: verify signature, hashes and formats offline\"\n")
	return b.String(), nil
}
