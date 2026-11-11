"""Make the site's print textures from real demo frames. Run with the airlock-demo venv (numpy) and ffmpeg on PATH.

  python halftone.py halftone IN.png OUT.png [cell=9] bright parts of IN become dots (white on transparent, used as a CSS mask)
  python halftone.py ink OUT.png                       tileable ink-wear mask: opaque with small worn-off specks

Provenance: ink.png is generated noise. Car sides (cell 9) are crops of the demo recordings:
  seal.png  ffmpeg -ss 26 -i demo/scene1.mp4 -frames:v 1 -vf crop=900:200:0:80   (seal and verify output)
  zarf.png  ffmpeg -ss 30 -i demo/scene2.mp4 -frames:v 1 -vf crop=800:560:0:60   (Zarf deploy YAML)
  chat.png  ffmpeg -ss 22 -i demo/scene2.mp4 -frames:v 1 -vf crop=800:500:800:120 (poisoned chat answer)
"""
import subprocess, sys
import numpy as np


def read_gray(path):
    w, h = map(int, subprocess.check_output(
        ["ffprobe", "-v", "error", "-show_entries", "stream=width,height", "-of", "csv=p=0", path]).decode().split(","))
    raw = subprocess.check_output(["ffmpeg", "-loglevel", "error", "-i", path, "-f", "rawvideo", "-pix_fmt", "gray", "-"])
    return np.frombuffer(raw, np.uint8).reshape(h, w).astype(float) / 255


def write_alpha(path, a):
    h, w = a.shape
    rgba = np.zeros((h, w, 4), np.uint8)
    rgba[..., :3] = 255
    rgba[..., 3] = (np.clip(a, 0, 1) * 255).astype(np.uint8)
    subprocess.run(["ffmpeg", "-loglevel", "error", "-y", "-f", "rawvideo", "-pix_fmt", "rgba", "-s", f"{w}x{h}",
                    "-i", "-", "-frames:v", "1", path], input=rgba.tobytes(), check=True)


def halftone(src, dst, cell=6):
    g = read_gray(src)
    h, w = (g.shape[0] // cell) * cell, (g.shape[1] // cell) * cell
    # mean brightness per cell, stretched so thin terminal text still prints
    m = g[:h, :w].reshape(h // cell, cell, w // cell, cell).mean(axis=(1, 3))
    m = np.clip((m - np.percentile(m, 50)) / (np.percentile(m, 99) - np.percentile(m, 50) + 1e-6), 0, 1) ** 0.7
    # 45-degree screen: offset every other row by half a cell; dot radius from tone
    yy, xx = np.mgrid[0:h, 0:w]
    cy, cx = yy // cell, xx // cell
    ox = np.where(cy % 2 == 1, cell / 2, 0)
    dx = (xx + ox) % cell - cell / 2 + 0.5
    dy = yy % cell - cell / 2 + 0.5
    r = np.sqrt(m[cy, cx] / np.pi) * cell * 0.95
    d = np.sqrt(dx * dx + dy * dy)
    write_alpha(dst, np.clip(r - d + 0.5, 0, 1))  # 1px antialiased edge


def ink(dst, size=512, seed=7):
    rng = np.random.default_rng(seed)
    f = np.fft.fftfreq(size)
    k = np.sqrt(f[:, None] ** 2 + f[None, :] ** 2)
    k[0, 0] = 1
    def field(power):  # FFT noise tiles seamlessly
        return np.real(np.fft.ifft2(np.fft.fft2(rng.standard_normal((size, size))) / k ** power))
    blot = field(1.25); blot = (blot - blot.mean()) / blot.std()
    grit = field(0.6); grit = (grit - grit.mean()) / grit.std()
    worn = (blot > 2.0) | (grit > 2.25)  # ~6% of the ink lifted in soft patches and fine specks
    write_alpha(dst, np.where(worn, 0.0, 1.0))


if __name__ == "__main__":
    if sys.argv[1] == "halftone":
        halftone(sys.argv[2], sys.argv[3], int(sys.argv[4]) if len(sys.argv) > 4 else 9)
    else:
        ink(sys.argv[2])
