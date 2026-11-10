# airlock

airlock checks an AI model before it goes into an air-gapped network, then checks it again on the offline side before it runs.

Moving the files is left to Zarf or a USB drive. airlock refuses file formats that can run code when loaded, confirms the weights match a pinned Hugging Face commit, and signs a manifest of every file. On the offline host it verifies all of that with no network and no other tools, and writes a report for the security team.

## What it checks

When sealing (online):

- Weights must be GGUF or safetensors, identified by file header. Pickle files (`.bin`, `.pt`, or a pickle renamed to anything else) are refused, as are unknown file types and symlinks.
- With `--hf` and `--rev`, every file must match the upstream repo at that commit. Branch names are refused because they can move.
- With `--clamav`, a ClamAV scan runs. It is off by default because the scanner and its database are one more downloaded thing to trust. If you ask for it and it can't run, sealing fails.
- A manifest of every file and its SHA-256 is signed with cosign. Nothing is uploaded to a transparency log.

When verifying (offline):

- The signature is checked before the manifest is parsed.
- Files on disk must match the manifest exactly: no missing, extra, or changed files.
- Formats are checked again.
- A Markdown report is written whether the check passes or fails.

## Usage

```sh
go build -o airlock .
cosign generate-key-pair
airlock seal ./bundle --key cosign.key --hf Qwen/Qwen2.5-0.5B-Instruct-GGUF --rev <40-char commit>
airlock verify ./bundle --key cosign.pub
```

`airlock check ./bundle` runs only the format checks.

### With Zarf

```sh
airlock zarf ./bundle --key cosign.pub
zarf package create . -o out
zarf package deploy out/zarf-package-airlock-model-*.tar.zst
```

The generated package copies the bundle and runs `airlock verify` during deploy. Put the component that serves the model after it. If verification fails, Zarf stops and the model never starts. Tested with Zarf v1.0.0.

## Installing airlock on the offline host

The offline host needs two things, installed once: the airlock binary at `/usr/local/bin/airlock` and the public key at `/etc/airlock/cosign.pub` (change the path with `--inside-key`). Every bundle after that is verified by this copy.

Neither is shipped inside the package by default. Anyone who could alter the package could also swap the key and the verifier, so both should arrive through the site's normal software intake. You can also rebuild from source on the offline side and compare hashes, since builds are reproducible with the same Go version:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o airlock . && sha256sum airlock
```

The binary has no dependencies outside the Go standard library. cosign is only needed for signing.

## Threat model

airlock detects:

- a file changed, added, or removed after sealing
- a bundle signed by a key the offline host does not trust
- pickle-based model files
- weights that differ from the pinned upstream commit

It does not cover:

- A stolen signing key. Keep `cosign.key` off shared machines. To rotate, install a new public key on the offline host through the same trusted channel.
- Harmful model behaviour. A model in a safe format from the right commit can still be a bad model, and airlock does not evaluate outputs.
- Changes after verification. Someone with write access between `verify` and model load could swap files. The Zarf integration runs `verify` during deploy to keep that window small.
- A malicious upstream commit. Pinning stops later changes to the repo, but if the pinned commit is bad, airlock accepts it.
- Transparency logs. Signatures are key-based only, because air-gapped sites usually cannot reach Rekor.
- Vulnerabilities in the serving image. Use Zarf's SBOMs or a scanner for that.

## Status

Not done yet: a smoke test that starts the model and sends it a prompt, an offline vulnerability re-scan with a bundled database, and updates that ship only changed files.
