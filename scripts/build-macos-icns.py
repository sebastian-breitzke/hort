#!/usr/bin/env python3
"""Build a macOS .iconset from a square master with a Big-Sur squircle mask.

Squircle approximation: superellipse |x|^n + |y|^n = 1 with n=5.
Mask is rendered at 4x resolution and downsampled with LANCZOS for smooth AA.
No extra inset — we assume the source already fills its canvas; we only clip
the corners to the squircle.
"""
from PIL import Image, ImageChops
import sys, os

MASTER = sys.argv[1]
OUT_DIR = sys.argv[2]

SIZES = [
    (16,  "icon_16x16.png"),
    (32,  "icon_16x16@2x.png"),
    (32,  "icon_32x32.png"),
    (64,  "icon_32x32@2x.png"),
    (128, "icon_128x128.png"),
    (256, "icon_128x128@2x.png"),
    (256, "icon_256x256.png"),
    (512, "icon_256x256@2x.png"),
    (512, "icon_512x512.png"),
    (1024,"icon_512x512@2x.png"),
]

N = 5.0
SUPERSAMPLE = 4

def squircle_mask(size: int) -> Image.Image:
    s = size * SUPERSAMPLE
    mask = Image.new("L", (s, s), 0)
    px = mask.load()
    r = s / 2.0
    for y in range(s):
        dy = (y + 0.5 - r) / r
        ay = abs(dy) ** N
        for x in range(s):
            dx = (x + 0.5 - r) / r
            if ay + abs(dx) ** N <= 1.0:
                px[x, y] = 255
    return mask.resize((size, size), Image.LANCZOS)

def build_icon(src: Image.Image, size: int) -> Image.Image:
    art = src.resize((size, size), Image.LANCZOS).convert("RGBA")
    mask = squircle_mask(size)
    r, g, b, a = art.split()
    combined = ImageChops.darker(a, mask)
    return Image.merge("RGBA", (r, g, b, combined))

def main():
    os.makedirs(OUT_DIR, exist_ok=True)
    src = Image.open(MASTER).convert("RGBA")
    w, h = src.size
    s = min(w, h)
    src = src.crop(((w - s) // 2, (h - s) // 2, (w - s) // 2 + s, (h - s) // 2 + s))

    for size, name in SIZES:
        out = build_icon(src, size)
        out.save(os.path.join(OUT_DIR, name), "PNG")
        print(f"  {name} ({size}x{size})")

if __name__ == "__main__":
    main()
