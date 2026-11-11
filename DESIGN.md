---
name: airlock
description: A constructivist broadside page where a model rides one rail from online seal to offline verify, and refused models stop at the gate.
colors:
  soot-black: "#0a0e0a"
  car-paint: "#101810"
  rail-green: "#2e5a37"
  board-green: "#c8f7c5"
  board-green-dim: "color-mix(in srgb, #c8f7c5 78%, #0a0e0a)"
  signal-mint: "#5af78e"
  ash-grey: "#8a9a8a"
  refusal-red: "#ff5f5f"
typography:
  display:
    fontFamily: "Big Shoulders Stencil Variable, Big Shoulders Stencil, sans-serif"
    fontSize: "clamp(5.5rem, 15.5vw, 14rem)"
    fontWeight: 800
    lineHeight: 0.8
    letterSpacing: "-0.005em"
  headline:
    fontFamily: "Big Shoulders Stencil Variable, Big Shoulders Stencil, sans-serif"
    fontSize: "clamp(3rem, 8.4vw, 7.5rem)"
    fontWeight: 800
    lineHeight: 0.86
    letterSpacing: "-0.005em"
  title:
    fontFamily: "Big Shoulders Stencil Variable, Big Shoulders Stencil, sans-serif"
    fontSize: "clamp(3.4rem, 6vw, 5.6rem)"
    fontWeight: 800
    lineHeight: 0.86
  body:
    fontFamily: "Archivo Variable, Archivo, system-ui, sans-serif"
    fontSize: "1.125rem"
    fontWeight: 400
    lineHeight: 1.55
  label:
    fontFamily: "Archivo Variable, Archivo, system-ui, sans-serif"
    fontVariation: '"wdth" 75'
    fontSize: "0.85rem"
    fontWeight: 800
    letterSpacing: "0.1em"
  code:
    fontFamily: "JetBrains Mono Variable, JetBrains Mono, ui-monospace, monospace"
    fontSize: "0.8rem"
    fontWeight: 400
    lineHeight: 1.4
spacing:
  gutter: "clamp(1rem, 4vw, 4rem)"
  section: "clamp(5rem, 11vw, 9.5rem)"
components:
  button-primary:
    backgroundColor: "{colors.signal-mint}"
    textColor: "{colors.soot-black}"
    padding: "1rem 2.2rem 1rem 1.4rem"
  button-primary-hover:
    backgroundColor: "{colors.board-green}"
    textColor: "{colors.soot-black}"
  button-inverse:
    backgroundColor: "{colors.soot-black}"
    textColor: "{colors.signal-mint}"
    padding: "1rem 2.2rem 1rem 1.4rem"
  button-inverse-hover:
    backgroundColor: "{colors.car-paint}"
    textColor: "{colors.signal-mint}"
  car:
    backgroundColor: "{colors.car-paint}"
    textColor: "{colors.board-green}"
    padding: "0 1.5rem 1.9rem"
  car-name-band:
    backgroundColor: "{colors.board-green}"
    textColor: "{colors.soot-black}"
    padding: "0.9rem 1.5rem 0.45rem"
  platform-board:
    backgroundColor: "{colors.soot-black}"
    textColor: "{colors.board-green}"
    width: "min(23rem, 40%)"
  stamp-pass:
    backgroundColor: "{colors.soot-black}"
    textColor: "{colors.signal-mint}"
    padding: "0.35rem 0.75rem 0.3rem"
  stamp-refuse:
    backgroundColor: "{colors.refusal-red}"
    textColor: "{colors.soot-black}"
    padding: "0.35rem 0.75rem 0.3rem"
  refused-car:
    backgroundColor: "{colors.soot-black}"
    textColor: "{colors.board-green-dim}"
    padding: "0 1.2rem 1.7rem"
  code-block:
    backgroundColor: "{colors.car-paint}"
    textColor: "{colors.board-green}"
    padding: "1.1rem 1.3rem 1.4rem"
  copy-button:
    backgroundColor: "{colors.soot-black}"
    textColor: "{colors.board-green}"
    padding: "0.35rem 0.7rem"
---

# Design System: airlock

## Overview

**Creative North Star: "The Line of Travel"**

A model rides one rail from online to offline. The page is that rail: every section is a station on it, the content travels as painted cars, and the air gap is the gate the model must pass. The surface is a constructivist broadside on a soot-black ground: stencil caps set on a -2.5deg tilt, square-cut wedges for actions, and dark cars printed with halftones of real demo frames.

Each section makes one claim, shows its proof as real values and recordings, and states its limit. Limits sit in a timetable on the page, not in a footnote. Refusals are cars turned back at the gate with the exact message airlock prints, not warnings or alerts. The system rejects the category default of a centred headline, two buttons and a row of feature cards, and it rejects phosphor glow, matrix rain and a fake terminal used as the look.

**Key Characteristics:**
- Dark only. One ground (#0a0e0a); the build has no light theme.
- Stencil caps on a tilt. The lowercase wordmark is the only lowercase stencil.
- Cars are the unit of content: name band, halftone side, spec ids, bogies, plate.
- Mint is the action colour; red appears only on refusals.
- Flat. No radii, gradients, glow or shadows. Depth comes from tilt, rails and bogies.
- Scroll drives the journey. Reduced motion shows the final states.

## Colors

The palette is near-black with pale green type. Mint marks actions and passes, red marks refusals, grey marks pending checks, and the cars sit a shade off the ground.

### Primary
- **Signal Mint** (#5af78e): The action colour. Drives every wedge button, the focus ring, text selection and the closing band. It also marks the PASSED stamp and the check ticks.

### Secondary
- **Board Green** (#c8f7c5): Body text on the ground, the name bands on cars, the platform board frame, and the hover state of the mint wedge.

### Tertiary
- **Refusal Red** (#ff5f5f): Refusal only. The REFUSED stamp, the rejected sample text, the mismatched hash cell, and the off-register print on the poisoned car.

### Neutral
- **Soot Black** (#0a0e0a): Page ground, header strip, platform board and stamp fill.
- **Car Paint** (#101810): Painted car sides and code blocks.
- **Rail Green** (#2e5a37): Rules and borders: the header underline, card borders, the route rail, board row rules and the sleepers under each line.
- **Ash Grey** (#8a9a8a): Checks that have not ticked yet, spec keys, table heads and the siding barrier.
- **Board Green Dim** (`color-mix(in srgb, #c8f7c5 78%, #0a0e0a)`, about #9ec49d): Secondary running text and intro copy.

Station labels, inline code and the route's progress line also take mint. They are recorded as built.

### Named Rules
**The One Action Rule.** Mint marks what the visitor can act on and what has passed. It never marks a refusal or a pending check.

**The Refusal Rule.** Red appears only on refused items.

**The Pending Grey Rule.** A check that has not ticked yet is grey. Spec keys are grey.

## Typography

**Display Font:** Big Shoulders Stencil Variable (fallback: Big Shoulders Stencil, sans-serif)
**Body Font:** Archivo Variable (fallback: Archivo, system-ui, sans-serif), set at wdth 100; labels drop to wdth 75 to 82
**Label/Mono Font:** JetBrains Mono Variable (fallback: JetBrains Mono, ui-monospace, monospace)

**Character:** Stencil caps carry the headlines like cut-out lettering on a train. Archivo is the plain working voice. Mono is the record of the run: ids, hashes and commands.

### Hierarchy
- **Display** (800, clamp(5.5rem, 15.5vw, 14rem), line-height 0.8; 13vw capped at 12rem between 961px and 1359px): The hero wordmark, `airlock`, in lowercase. Set in the hero only.
- **Headline** (800, clamp(3rem, 8.4vw, 7.5rem), line-height 0.86, max 14ch, set on the tilt): Section headings, uppercase stencil.
- **Title** (800, clamp(3.4rem, 6vw, 5.6rem) on the line, clamp(2.3rem, 3.3vw, 3.2rem) on the hero train, line-height 0.86): Car names, uppercase stencil on a Board Green band.
- **Body** (400, 1.125rem, line-height 1.55; 1.0625rem under 640px): Running text. Intros run to about 58ch. `text-wrap: pretty`.
- **Label** (800, 0.85rem, letter-spacing 0.1em, uppercase, wdth 75): Station lines, board labels, table heads, the gap label.
- **Code** (JetBrains Mono, 0.8rem, line-height 1.4 for spec blocks; 0.92rem, line-height 1.7 in code blocks; 0.74rem to 0.78rem bold for plates and notes): Real ids, hashes, byte counts, commands and their annotations.

### Named Rules
**The Stencil Caps Rule.** Headlines and car names are uppercase stencil with ink wear. The lowercase wordmark is the only lowercase stencil on the page.

**The Record Rule.** Mono sets the record: spec blocks, car plates, leader-line notes, commands and code.

## Layout

- **Page:** one column of sections with side padding `clamp(1rem, 4vw, 4rem)` and section padding `clamp(5rem, 11vw, 9.5rem)`.
- **Route:** a 4px Rail Green rail runs from the first section to the buffer stop at the end of the line. Each section carries a station mark on it. The riding bundle is sticky at 50vh.
- **Hero:** minimum height clamp(44rem, 100svh minus 3.9rem, 58rem). Copy column 40rem max. The train sits at left 35%, rotated -9deg, on a mint band.
- **Line:** three cars in equal columns, gap clamp(1.2rem, 2.4vw, 2.2rem), set on the tilt.
- **Start and reel:** the quick-start code blocks sit in a 1.55fr / 1fr grid. The two demo videos sit side by side, the second offset 3.5rem down.
- **Zarf:** a 1fr / 1.1fr split. **Limits:** a two-column timetable of rows.
- **Mid widths, 961px to 1359px:** the whole hero train scales to 0.8, and to 0.66 from 961px to 1180px.
- **Narrow, 960px and under:** the hero train moves below the copy. The route rail moves to the gutter. The line and the siding stack in one column, and their rails are hidden. The gate barrier runs horizontally under the siding.
- **Phone, 640px and under:** the hero train turns vertical: cars stack down the screen, the gap becomes a horizontal dashed bar, and the train tilts -3deg. The header becomes two rows, with the nav in a horizontal scroll row. Manifest rows and timetable rows become single-column lists.

## Elevation & Depth

Flat. The build uses no box-shadow, no backdrop filter, no blur and no gradient. Depth comes from the tilt, painted car sides against the soot ground, a rail under each consist, and bogies under each car. The one hard offset is a text-shadow on the poisoned car's heading, recorded under Components as a refusal-only effect.

### Named Rules
**The No-Lift Rule.** Surfaces never lift on hover or scroll. A state change moves the wedge, its clip cut or a stamp, never a shadow.

## Shapes

Nothing is rounded. The build has no border-radius. Corners are square, and the angle comes from clip-path wedges: the action wedge's cut, and the punched ticket on the copy button after a successful copy. Borders are 1px rules, 2px Rail Green or ink frames on cars, 3px board or ink frames on the hero engine and code blocks, and 3px dashed frames for the gap and the barrier. Car sides carry a halftone of a demo frame, masked to the car. Each car is broadside, with two bogies underneath and a coupler to the next car.

### Named Rules
**The Square Cut Rule.** Nothing is rounded. Angles come from clip-path cuts and from the tilt.

## Components

Each component states its character first.

### Buttons
A wedge: a square button whose right edge is cut at an angle, so the cut deepens when the visitor reaches for it.
- **Shape:** square, clip-path cut 1.1rem (1.9rem on hover).
- **Primary:** Signal Mint wedge, Soot Black text, 1.15rem, weight 800, uppercase, letter-spacing 0.05em, padding 1rem 2.2rem 1rem 1.4rem, with a 1.15em arrow stroke icon.
- **Hover:** fill moves to Board Green, the cut deepens, the wedge shifts 0.45rem right, and the arrow moves 0.4rem. Timing 0.25s to 0.3s on `cubic-bezier(0.16, 1, 0.3, 1)`.
- **Active:** translateY(1px).
- **Small** (header): 0.95rem, padding .55rem 1.6rem .55rem .95rem. **Large** (closing band): 1.4rem, padding 1.3rem 3rem 1.3rem 1.8rem.
- **Inverse** (on the mint closing band): Soot Black wedge with Signal Mint text, Car Paint on hover.
- **Text link:** the Quick start link and the header nav are underlined text. The underline is transparent until hover or focus, then mint.

### Car
The car is a painted broadside. It carries its name, its station line, its specs and its plate, and it moves along the rail.
- **Sides:** Car Paint, with a halftone of a demo frame printed in Rail Green over the side. The halftone is a mask on the car's own pseudo-element, so the side keeps its painted ground.
- **Name band:** Board Green, full bleed across the car, with the stencil name in Soot Black.
- **Station line:** label style in Signal Mint, under the band.
- **Spec block:** mono definition list. Keys in Ash Grey, values in Board Green, tabular numerals.
- **Plate:** a mono tag, "Car 1" to "Car 3", on the car's top edge. Mint with Soot Black text, or Board Green on phones.
- **Note:** a mono leader-line annotation with a 2px ink leader, printed on the band below the rails.
- **Bogies:** two wheel pairs under each car, Soot Black, with a Rail Green coupler to the next car.
- **Raster provenance:** `site/public/print/seal.png`, `zarf.png` and `chat.png` are halftones (`site/tools/halftone.py`, halftone mode, cell 9px as recorded in the brief; the script's own default is 6px) of crops from `demo/scene1.mp4` at 26 s and `demo/scene2.mp4` at 30 s and 22 s. The per-file mapping is not recorded in the repo. `site/public/print/ink.png` is generated noise from the script's `ink` mode and is the wear mask for stencil, plates, the board frame and stamps. The posters `site/public/media/scene1-poster.jpg` (38 s) and `scene2-poster.jpg` (20 s) are frames. `site/public/social.png` is a copy of `docs/social.png`.

### Platform Board
A black board with a worn Board Green frame and Rail Green row rules. Each row has a grey label on the left and a bold value on the right. Checks read "Checking" in grey, then print a mint tick with the result.

### Stamps
Stencil caps at weight 900, rotated, with the ink-wear mask.
- **Passed:** Signal Mint text and 4px mint border on a Soot Black fill, rotated -12deg.
- **Refused:** Soot Black text on a Refusal Red fill, rotated -6deg, placed over the car's top edge.

### Refused Car
A car on the siding with a transparent body and a Rail Green frame. The message is a red mono sample on a Soot Black plate with a red 55% border, the exact text airlock prints. The cut-off poisoned car is the only one that prints off register.
- **Off-register print:** a red 2px ring offset by 6px, and a red text-shadow offset by 6px and 3.5px with no blur. It applies only to this refused car and is refusal-only. It is not a shadow token.

### Split-flap Row
A hash row on the poisoned car: each character sits in a 1.15em by 1.5em cell with a hinge line, and mismatched characters are red. On view it flips through random hex glyphs and settles on the real value. The real hash stays in the DOM for screen readers.

### Code Block with Copy
A Car Paint block with a 6px Board Green top border and mono code at 0.92rem. The copy button sits at the top right with a Rail Green border, turns mint on hover, and after a copy shows "Copied" with a punched-ticket edge. If clipboard access fails, the button reads "Press Ctrl+C" and selects the code.

### The Journey (scroll-driven motion)
Motion runs only under `prefers-reduced-motion: no-preference` and where `animation-timeline: scroll()` is supported.
- The hero train departs over 34vh to 110vh, translating 7vw and -1.1vw.
- The verify car's scan bar crosses the car over 3vh to 24vh, then the three checks tick at 9vh, 15vh and 21vh, and the PASSED stamp lands over 25vh to 31vh.
- The rail draws down to the bundle. Each station mark lights when the bundle passes it.
- Headings slide in along the tilt, 3.5rem over their entry range.
- The refused train rolls in over the siding, overshoots 0.9rem, and brakes at the gate. The poisoned car drifts off register. The REFUSED stamp lands from contain 15% to 22%.
- On first load the train pulls in over 1.5s. The split-flap cells flip for 0.5s plus 0.07s per cell, stepped.
- Videos play only while on screen, at 50% visibility.

**Reduced motion** shows the final states without animation: checks ticked, PASSED and REFUSED stamped, the rail drawn, the poisoned car offset, the train at rest, the split-flap showing its final text, and the videos showing controls.

### Named Rules
**The Off-Register Rule.** Only refused items print off register.

## Do's and Don'ts

### Do:
- **Do** set each section as a station on the rail, and let the content travel as cars.
- **Do** give every car its spec ids in mono and its plate, and pin annotations on leader lines.
- **Do** state the limits on the page in the timetable, where the visitor reads them.
- **Do** write the name as lowercase `airlock`.
- **Do** give every motion a static final state under reduced motion.
- **Do** keep mint for actions and passes only.

### Don't:
- **Don't** use gradients, glow, rounded cards or lift shadows.
- **Don't** use phosphor glow, matrix rain, or a fake terminal as the look.
- **Don't** use red for anything but a refusal.
- **Don't** state a licence, stars, users, testimonials, benchmarks or adoption figures. None exist yet (PRODUCT.md).
- **Don't** use stars, flags or crowds as imagery.
- **Don't** change the wordmark's case or set it in uppercase stencil.
