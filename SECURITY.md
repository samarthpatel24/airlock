# Security policy

airlock is a security gate, so a way around it is a vulnerability. Please report privately.

## Reporting

Use **Report a vulnerability** on the repository's [Security tab](https://github.com/samarthpatel24/airlock/security/advisories/new). Do not open a public issue.

Include the airlock version (`airlock version`), the command you ran, and the smallest bundle or steps that show the problem. You should get a reply within a week.

## In scope

- A bundle that `airlock verify` passes although a file was changed, added, removed or signed by a key the host does not trust.
- A file format that can run code when loaded getting past `seal` or `check`.
- A crafted GGUF or safetensors file that crashes or hangs the parsers, or makes them read outside the file.
- A model whose files do not match the pinned Hugging Face commit getting sealed.
- The Zarf deploy step being skipped or bypassed in a signed package deployed with `--verify=always`.
- Anything in the report that renders as Markdown or HTML from model output.

## Out of scope

The limits listed under [Threat model → Not covered](README.md#06--threat-model), such as a stolen signing key, harmful model behaviour, or a malicious upstream commit. Ideas for closing those gaps are welcome as normal issues.

## Supported versions

Only the latest release gets fixes.
