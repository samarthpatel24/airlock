---
version: 1
slug: "site-index-html"
primary_target: "index.html"
related_targets: []
---

# airlock.sammatrix.in landing page

Scope: one landing page, `index.html` in `site/`, TypeScript + Vite static build, deployed on Vercel at airlock.sammatrix.in.
Visitor mode: Persuade.

Audience and job: platform, MLOps and security engineers deciding whether airlock fits their model intake; hiring managers judging depth. Primary action: open the GitHub repo. Secondary: copy the quick start.
Proof on hand: the two demo recordings (demo/scene1.mp4, demo/scene2.mp4), real CLI output, the tested model (Qwen2.5 0.5B Q4_K_M @ 9217f5db, 491 400 032 bytes, SHA-256 74a4da8c…), Zarf v1.0.0 test.
Constraints: no hype, no generic hacker look (no phosphor glow, no matrix rain, no fake terminal everywhere), limits shown openly, brand palette and logo kept, no invented users, stars, licence or metrics, no loaded political emblems (no stars, flags, crowds).
Memorable moment: the train crosses the gap and the verify car is stamped through the gate; the poisoned car is stopped in red.

## Direction contract

THESIS: The model crosses the air gap as a train of broadside cars; every section is a station on one line (a rail runs the page and the bundle rides it), the journey's own content travels as cars (hero train, the three line cars, the refused cars on the siding), and the airlock is the gate it must pass. (Amended 2026-11-11 to match the user-approved scroll-journey plan.) Refuses the category default of a centred headline, two buttons and a row of feature cards.

OWN-WORLD: Soot black ground (#0a0e0a); cars have dark painted sides (#101810) under a pale-green (#c8f7c5) name band (user decision 2026-11-11: dark cars win over pale sides); mint (#5af78e) is the single action colour and drives every wedge; red (#ff5f5f) is rationed to refusal only; grey (#8a9a8a) for pending. Stencil condensed caps for headlines and car numbers, set on the line of travel; spec mono for real file names, bytes, hashes. Wedge-cut buttons, slogan bands, halftone photomontage made from real demo frames (site/tools/halftone.py), printed ink wear from a generated noise mask (user decision: made in code, no outside asset), rail and platform-board furniture. No gradients, no glow, no rounded cards.

STORY: The visitor sees a model ride from online to offline, understands seal, move, verify in one glance, believes it because each car carries real values and real recordings, sees what it catches and what it does not, and opens GitHub.

FIRST VIEWPORT: Platform header strip with logo, nav and a mint GitHub wedge. Left: lowercase airlock in the stencil face at about 22vh (PRODUCT.md: name is lowercase), slogan beneath, primary mint wedge "View on GitHub" and an underlined "Quick start" link. Right and across: a diagonal train rising at about 10 degrees, engine plus cars SEAL, MOVE, VERIFY with spec labels, a dashed air-gap gate between MOVE and VERIFY. A platform board bottom-left reads the next stop and checks passed.

FORM: Constructivist agit-train broadside, dealt lead of the bolder register (deal position 1). seed:5adaaf0c82422ff4

SIGNATURE INTERACTION: scroll-driven travel; as the visitor scrolls, the cars advance along the line and the verify car is stamped PASSED at the gate; later the poisoned car reaches the gate and stops, stamped REFUSED in red. Reduced motion shows the final states statically.

RAISES: proof as registration (a tampered car prints off register); spec mono ids on every car; annotations pinned on leader lines; system states print themselves as stencilled lines, never as chrome.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
