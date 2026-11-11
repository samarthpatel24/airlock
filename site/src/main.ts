import "@fontsource-variable/big-shoulders-stencil";
import "@fontsource-variable/archivo/wdth.css";
import "@fontsource-variable/jetbrains-mono";
import "./style.css";

// Copy buttons: copy the code block named by data-copy, then report the result on the button.
for (const button of document.querySelectorAll<HTMLButtonElement>("button[data-copy]")) {
  const label = button.querySelector("span");
  const code = document.getElementById(button.dataset.copy ?? "");
  if (!label || !code) continue;
  let timer = 0;
  button.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(code.textContent ?? "");
      button.dataset.state = "done";
      label.textContent = "Copied";
    } catch {
      // Clipboard blocked (insecure origin or denied): select the text so Ctrl+C works.
      getSelection()?.selectAllChildren(code);
      button.dataset.state = "error";
      label.textContent = "Press Ctrl+C";
    }
    clearTimeout(timer);
    timer = window.setTimeout(() => {
      delete button.dataset.state;
      label.textContent = "Copy";
    }, 2200);
  });
}

const calm = matchMedia("(prefers-reduced-motion: reduce)");

// Split-flap hash board: each character flips through a few random hex glyphs and settles on
// the real value. The real hash stays in the DOM as text; the flipping cells are decoration.
const flaps = document.querySelector<HTMLElement>(".flaps");
if (flaps && !calm.matches && "IntersectionObserver" in window) {
  for (const flap of flaps.querySelectorAll<HTMLElement>(".flap")) {
    const final = flap.dataset.final ?? "";
    const cells = [...final].map((ch, i) => {
      const n = 5 + ((i * 7) % 5);
      const glyphs = Array.from({ length: n }, () => "0123456789abcdef"[Math.floor(Math.random() * 16)]);
      return `<span class="cell" style="--n:${n};--i:${i}"><span class="strip">${[...glyphs, ch].map((g) => `<i>${g}</i>`).join("")}</span></span>`;
    });
    flap.innerHTML = `<span class="sr-only">${final}</span><span class="cells" aria-hidden="true">${cells.join("")}</span>`;
  }
  const watch = new IntersectionObserver(
    ([entry]) => {
      if (!entry.isIntersecting) return;
      flaps.classList.add("flipping");
      watch.disconnect();
    },
    { threshold: 0.9 },
  );
  watch.observe(flaps);
}

// Demo videos play only while on screen, and never on their own under reduced motion.
const videos = document.querySelectorAll<HTMLVideoElement>(".frame video");
// Without autoplay (reduced motion or no observer) the visitor needs controls to start them.
if (calm.matches || !("IntersectionObserver" in window)) {
  for (const video of videos) video.controls = true;
} else {
  const seen = new IntersectionObserver(
    (entries) => {
      for (const { target, isIntersecting } of entries) {
        const video = target as HTMLVideoElement;
        if (isIntersecting) video.play().catch(() => {});
        else video.pause();
      }
    },
    { threshold: 0.5 },
  );
  for (const video of videos) {
    video.preload = "metadata";
    seen.observe(video);
  }
}
