# bspace — Universal Browser Workspace for Coding Agents

> **Status:** Idea / pre-prototype  
> **Working name:** `bspace` (not finalized)  
> **Category:** Developer tooling · Coding agents · Terminal UI · Browser automation

## 1. The idea in one paragraph

Build a **terminal-first, agent-agnostic browser workspace** that developers can launch beside their coding assistant. A single CLI provides a persistent, interactive Chromium session that both the human and an AI coding agent can inspect and operate. Through simple CLI commands and an MCP interface, the same workspace can serve as a **live frontend preview, browser automation debugger, web research/crawling environment, or visual design playground**. The goal is not to create a new browser engine: it is to make browser-based work a native, reusable part of the terminal coding workflow.

**One-liner:** *One browser workspace. Any coding agent. Directly in your terminal.*

## 2. Problem

Developers increasingly work with CLI coding agents, but browser-based tasks remain fragmented:

- A coding agent may use one browser automation tool while the developer watches an unrelated browser window.
- Visual frontend changes require repeated switching between the terminal, browser, screenshots, and developer tools.
- Browser sessions, authentication, tabs, and test context are often lost or duplicated across agent restarts and tool integrations.
- When an automation fails, it is difficult to see exactly what the agent saw, intervene, and resume without starting over.
- Tooling for browsing, scraping, visual inspection, and frontend iteration tends to be separate, even though it operates on the same browser primitives.

**Product hypothesis:** The missing abstraction is a **shared browser session with a human-friendly terminal view and an agent-friendly control API**.

## 3. Target users

1. **Agent-heavy developers:** People using Claude Code, Codex, Gemini CLI, or custom agents for day-to-day engineering.
2. **Frontend developers/design engineers:** People who want a live preview and element inspection alongside their editor or agent.
3. **Automation builders:** People developing Playwright scripts or agent workflows who need visible execution and intervention.
4. **Research / data workflows:** People doing permitted website exploration, extraction, and navigation from an agent.

The first target audience should be **developers already working primarily in the terminal**.

## 4. Product experience

### Example layout

```text
┌────────────────────────────────┬────────────────────────────┐
│ Terminal / Claude Code         │ bspace — Browser           │
│                                │                            │
│ > Fix the navbar alignment     │ [ localhost:3000      ]    │
│                                │ ┌────────────────────────┐ │
│ Agent edits components...      │ │    Live rendered app   │ │
│ Agent inspects DOM...          │ │                        │ │
│ Agent tests responsive view... │ │   Click any element    │ │
│                                │ └────────────────────────┘ │
│                                │ [Inspect] [Interact] [Log] │
└────────────────────────────────┴────────────────────────────┘
```

The right pane is an **interactive rendering of a real browser session**, not a text-only web page. Exact fidelity and responsiveness depend on the terminal and rendering backend; where inline graphics are unavailable, a normal browser window is an acceptable fallback.

### Illustrative CLI (proposed, not implemented)

```bash
# Initialize agent integrations in the current project
bspace init

# Open a shared browser session
bspace open http://localhost:3000 --split right

# Show the same session as a terminal pane / TUI
bspace attach dev

# Switch workflows without switching sessions
bspace mode design
bspace mode automation
bspace mode inspect

# Export diagnostic artifacts
bspace screenshot --out ./artifacts/home.png
bspace trace export --out ./artifacts/run.zip

# List and safely close sessions
bspace sessions
bspace close dev
```

Commands are design sketches; their spelling and scope should be validated during prototyping.

## 5. Primary workflows

### A. Live frontend preview

1. Developer starts the application locally and opens its URL in `bspace`.
2. Agent edits source code; HMR updates the browser.
3. Human views the new result in a neighboring terminal pane.
4. Agent inspects console errors, DOM, accessibility tree, and screenshots through the same session.

**Value:** Tighten the edit → view → inspect → fix loop without switching applications.

### B. Visual design playground

1. Human clicks an element in the browser pane.
2. `bspace` captures element identity, selectors, layout information, computed styles, and a screenshot crop.
3. The user sends an instruction such as “reduce this card's padding and make it align with the header.”
4. Agent changes source files, not merely temporary browser styles, and sees the live update.

**Value:** Ground design instructions in a real selected element, avoiding vague visual prompts.

### C. Watch and intervene in automation

1. Agent navigates the browser through CLI or MCP actions.
2. Every major action is visible in the browser pane and in an event log.
3. Developer can pause execution, take temporary control, correct state, and resume.
4. Failed runs can export screenshots, console output, network summaries, and action traces.

**Value:** Make agent browser automation inspectable and debuggable.

### D. Web research and controlled extraction

1. Agent opens pages in an isolated browser context.
2. It extracts permitted page content, structured DOM, and relevant links.
3. Human can inspect the source page directly and intervene if navigation fails.
4. Exports include source URLs and timestamps for provenance.

**Value:** Reuse the same browser workspace for interactive exploration and structured extraction. Respect robots directives, site terms, access controls, and rate limits as applicable; do not market the product as an anti-bot bypass tool.

## 6. What makes it different?

A terminal browser is **not** the defensible feature by itself. Browser automation, MCP connectors, and terminal rendering already have established implementations.

The distinctive combination to explore is:

- **One shared session:** Human and agent see and manipulate the same tab, cookies, navigation state, and viewport.
- **Agent-agnostic integration:** A stable CLI and MCP API with thin optional adapters for individual coding agents.
- **Native terminal ergonomics:** View, inspect, and intervene inside a split-pane workflow; external-window fallback where needed.
- **Visual-to-code handoff:** Turn a clicked element into an actionable, source-oriented context bundle for the coding agent.
- **Observability first:** Record agent actions, page state, screenshots, console output, and failures.
- **Persistence with explicit controls:** Attach to an existing browser process and recover useful project state without silently sharing sensitive sessions.

**Core positioning:** *A browser workspace and shared debugging surface for agents*, rather than another autonomous web agent.

## 7. MVP: deliberately narrow

**Version 0.1 should prove only one complete use case:** a developer runs Claude Code or another CLI agent on the left, sees a real interactive local website on the right, and both sides use the same session.

### Must-have

- Launch a Chromium browser through Playwright/CDP.
- Open and attach to a persistent **local** session.
- Render periodic screenshots into a supported terminal graphics pane; map basic mouse/keyboard input back to the browser.
- Navigate, click, type, screenshot, inspect page text/DOM, and read console errors through a CLI.
- Expose a small MCP tool surface backed by the exact same session.
- Show agent actions in an event log.
- Work with at least one terminal emulator and one coding agent end-to-end.
- Provide clear session ownership and a `close` command.

### Explicitly out of scope for v0.1

- Building an HTML/CSS/JS browser engine.
- Supporting every terminal and OS at launch.
- Full Chrome DevTools UI inside the terminal.
- Autonomous multi-step crawling orchestration.
- Multi-agent concurrent control, cloud sync, and remote team collaboration.
- Perfect 60 FPS video rendering or full browser-feature parity.
- Automatic authentication handling or unrestricted access to personal browser profiles.

### Preferred first demo

**Claude Code + local React/Next.js application + terminal browser pane.**

Demo script: ask agent to fix a responsive navbar; agent inspects DOM and console; developer watches the live preview, clicks the misaligned element, and passes the selected-element context back to the agent.

## 8. Proposed architecture

```text
┌──────────────────────────────────────────────────────┐
│ Clients                                              │
│                                                      │
│ CLI commands   MCP tools   Terminal viewer / TUI     │
└──────────┬───────────┬─────────────────┬──────────────┘
           │           │                 │
           ▼           ▼                 ▼
┌──────────────────────────────────────────────────────┐
│ bspace local daemon                                  │
│                                                      │
│ Session registry + permissions                       │
│ Action queue / pause / ownership                     │
│ Event stream + diagnostics + artifact export         │
│ Page snapshots + selected-element context            │
└─────────────────────────┬────────────────────────────┘
                          │ Playwright / CDP
                          ▼
┌──────────────────────────────────────────────────────┐
│ Isolated Chromium browser context                    │
│ Tabs | DOM | JS | CSS | cookies | storage | network   │
└──────────────────────────────────────────────────────┘
```

### Key principle: separate the browser from the viewer

The **browser runtime** owns real Chromium state. The **terminal viewer** only renders it and forwards user input. The **agent API** controls that same runtime. This prevents a terminal-rendering limitation from breaking automation and allows other viewers to be added later.

### Suggested implementation

| Layer | Initial candidate | Why |
|---|---|---|
| Runtime | TypeScript / Node.js | Strong Playwright and MCP ecosystem |
| Browser control | Playwright + Chrome DevTools Protocol | Established automation and inspection primitives |
| Daemon transport | Local Unix socket or loopback WebSocket | Fast session/control channel; no public listener by default |
| Agent API | CLI + MCP server | Works with many coding agents without custom plugins |
| Terminal interface | Ink / OpenTUI | TUI controls and session panels |
| Rendering | Terminal graphics protocol + screenshot frames | Pragmatic MVP; true native browser rendering is unnecessary |
| Fallback | External window bound to same session | Universal usability when terminal graphics are limited |
| Diagnostics | Action log, console log, screenshots; tracing later | Makes failures understandable |

**Rendering caveat:** Screenshot-stream rendering is a proof-of-concept strategy, not guaranteed to match a native browser's latency, text sharpness, video quality, accessibility, or interaction smoothness. Benchmark it before committing to it as the primary UX.

## 9. Agent integration contract

The integration should use the same primitives regardless of agent:

```text
browser_open(url, session_id?)
browser_snapshot(session_id)
browser_click(session_id, locator)
browser_type(session_id, locator, text)
browser_screenshot(session_id)
browser_console(session_id)
browser_select_element(session_id)
browser_pause(session_id)
browser_resume(session_id)
browser_close(session_id)
```

Treat these as **conceptual operations**. The implementation can provide CLI counterparts plus MCP tools with stable schemas.

Prefer semantic locators and accessibility snapshots over brittle pixel coordinates. Keep selected-element context compact and source-relevant (DOM path, attributes, computed styles, bounding box, screenshot crop, and optionally local source mapping).

### Human/agent input coordination

For MVP, serialize actions. Give the human a visible **Pause agent / Take control / Resume** flow. Reject or queue agent actions while the human has control rather than allowing unpredictable simultaneous clicks.

## 10. Security and privacy defaults

- Bind daemon interfaces to localhost or Unix sockets, with appropriate access permissions.
- Use isolated browser profiles/contexts by default; never automatically attach to a user's everyday Chrome profile.
- Make session sharing and reuse visible and opt-in.
- Keep cookies, tokens, passwords, and sensitive request bodies out of default logs/exports.
- Confirm or gate risky actions such as form submission, purchases, destructive UI actions, and credential entry where appropriate.
- Treat website content as untrusted; pages must not become authority over agent instructions.
- Provide session cleanup, retention controls, and explicit export of diagnostic artifacts.

## 11. Existing work to examine

These are **reference projects, not an assertion of feature parity or a complete competitive audit**:

- [zenbu-labs/terminal-browser](https://github.com/zenbu-labs/terminal-browser) — browser rendering and workflow inside terminals.
- [vercel-labs/agent-browser](https://github.com/vercel-labs/agent-browser) — browser CLI and agent-oriented interaction model.
- [ChromeDevTools/chrome-devtools-mcp](https://github.com/ChromeDevTools/chrome-devtools-mcp) — debugging and inspection via MCP.
- [browser-use/browser-use](https://github.com/browser-use/browser-use) — agent/browser workflow patterns.
- [Playwright](https://playwright.dev/) — established browser automation API.

**Before investing heavily:** inspect these projects' current feature sets, licenses, rendering approach, integration boundaries, and open issues. In particular, determine whether `bspace` is better as a plugin/extension to an existing terminal browser than as a standalone implementation.

## 12. Implementation milestones

### Milestone 0 — Feasibility spike

- Verify terminal graphics capability on macOS (e.g., Kitty or Ghostty).
- Launch a headless/headed Chromium session and display successive frames in a terminal pane.
- Send click, scroll, keyboard, and resize events back to the browser.
- Measure screenshot latency, CPU usage, and quality at realistic pane sizes.

**Exit criterion:** A developer can navigate a local web app in the terminal pane with tolerable interaction latency.

### Milestone 1 — Shared session

- Implement a local daemon, session IDs, lifecycle commands, and browser action queue.
- Connect the TUI and CLI to exactly one session.
- Add DOM inspection, screenshots, console events, and human override.

**Exit criterion:** Actions via CLI immediately appear in the same session shown in the terminal pane.

### Milestone 2 — Agent integration

- Add MCP server with minimal browsing/inspection tools.
- Supply a small Claude Code integration example; test another agent without agent-specific runtime changes.
- Export selected-element context and add a simple inspection mode.

**Exit criterion:** Two different coding agents can use the same documented interface.

### Milestone 3 — Showcase

- Publish a polished frontend iteration demo: inspect → select → agent edits → browser updates.
- Package a one-command setup, terminal capability detection, and external-viewer fallback.
- Document security boundaries and limitations; collect early developer feedback.

**Exit criterion:** A new developer can replicate the demo without patching the code.

## 13. Main technical risks

| Risk | Mitigation |
|---|---|
| Browser graphics lag in terminal | Measure early; reduce frame rate / region-update where possible; use external viewer fallback |
| Terminal inconsistencies | Start with a small support matrix; detect capabilities instead of guessing |
| Mouse coordinate and scaling errors | Single viewport model; track pane dimensions, device pixel ratio, and scroll state |
| Agent and human action collisions | Explicit control arbitration and ordered action queue |
| Existing tools already cover most of the product | Build a differentiating design workflow; integrate or extend rather than duplicate |
| Authentication or leaked debug artifacts | Isolated contexts, redaction, consent, local-only defaults |
| Design element cannot map to source | Start with DOM/style/selector context; treat source mapping as optional/best-effort |

## 14. How to judge whether the idea is worthwhile

Do not evaluate it solely by GitHub stars or the novelty of a terminal browser screenshot. Test whether the shared workflow solves a measurable problem.

**Early metrics:**

- Time from `bspace open` to first usable live preview.
- Input-to-visible-update latency and CPU/RAM use at typical pane size.
- Percentage of actions reflected consistently across agent API and terminal viewer.
- Time to debug a failed automation compared with screenshots/logs alone.
- Time to complete a simple frontend correction with and without visual element selection.
- Number of setup steps to connect a second coding agent.

**Kill/pivot criteria:** If interactive terminal viewing is consistently too slow or cumbersome, retain the **shared browser daemon + MCP/CLI + design inspection** and allow an ordinary browser window as the primary viewer. The shared environment is the durable value; the embedded pixels are not mandatory.

## 15. Longer-term opportunities (not MVP promises)

- Responsive viewport presets and side-by-side visual comparisons.
- Visual regression snapshots and annotated diffs.
- Record/replay of agent browser sessions.
- Cross-agent session handoff and reproducible workspace manifests.
- Network/console/DOM timelines for debugging.
- Scoped crawling jobs with provenance and policy-aware controls.
- Optional desktop viewer, IDE extension, and remote browser backend.

## 16. Open decisions

1. **Extend or build?** Can an existing terminal browser or agent-browser runtime be reused cleanly?
2. **Viewer priority:** Is a true terminal graphics pane essential to adoption, or is a terminal-controlled companion window sufficient?
3. **Primary differentiator:** Is visual-to-source design iteration more compelling than automation observability?
4. **Initial platform:** macOS + one supported terminal first, or Linux support from day one?
5. **Agent model:** Expose MCP tools alone, or ship an installation assistant that registers tools for each supported agent?

## 17. First concrete experiment

**Build the smallest possible vertical slice:**

1. Start Chromium with Playwright.
2. Open `http://localhost:3000`.
3. Render a screenshot periodically in a terminal side pane.
4. Forward a user click and a keyboard input to the browser.
5. Implement `bspace click` and `bspace screenshot` against the same active tab.
6. Add one MCP action that clicks an element, and observe it appear in the pane.

If this is smooth and stable, proceed to visual element selection. If it is not, test the same architecture with an external viewer before abandoning the product idea.

---

**Summary:** `bspace` should be pitched as **a shared, observable browser workspace for humans and AI coding agents**. Its unique value is the combination of universal agent integration, persistent shared state, human intervention, and optional terminal-native visual interaction—not the existence of browser automation or a browser rendered in a terminal.
