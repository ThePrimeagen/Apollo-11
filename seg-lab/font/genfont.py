#!/usr/bin/env python3
"""Build SegmentedAlpha.ttf — one-cell 14-segment A–Z in the PUA.

Unicode only encodes segmented digits (U+1FBF0–U+1FBF9). This font puts a
matching 14-segment alphabet at U+E000–U+E019 (A–Z) so a terminal can draw
HELLO WORLD as one cell per letter. Digits are also drawn at U+1FBF0–U+1FBF9
so they match when Noto is missing.
"""

from __future__ import annotations

import math
from pathlib import Path

from fontTools.fontBuilder import FontBuilder
from fontTools.pens.ttGlyphPen import TTGlyphPen

UPM = 1000
# Cascadia Mono is 1200/2048 em ≈ 586/1000. Stay on that grid or
# one-cell letters overflow into their neighbors.
ADV = 586
LSB = 48

# Frame in the same neighborhood as Noto's segmented zero, scaled to ADV.
L, R = 70, 516
T, B = 770, -50
MY = 360
CX = (L + R) / 2
TH = 52
GAP = 12

# Segment bits, matching seg-lab/seg/seg.go
A, B, C, D, E, F, G1, G2, H, I, J, K, Lseg, M = (
    1 << i for i in range(14)
)

FONT = {
    "0": A | B | C | D | E | F,
    "1": B | C,
    "2": A | B | G1 | G2 | E | D,
    "3": A | B | C | D | G1 | G2,
    "4": F | G1 | G2 | B | C,
    "5": A | F | G1 | G2 | C | D,
    "6": A | F | E | D | C | G1 | G2,
    "7": A | B | C,
    "8": A | B | C | D | E | F | G1 | G2,
    "9": A | F | G1 | G2 | B | C | D,
    "A": A | F | B | G1 | G2 | E | C,
    "B": A | B | C | D | I | Lseg | G1 | G2,
    "C": A | F | E | D,
    "D": A | B | C | D | I | Lseg,
    "E": A | F | E | D | G1,
    "F": A | F | E | G1,
    "G": A | F | E | D | C | G2,
    "H": F | E | B | C | G1 | G2,
    "I": A | D | I | Lseg,
    "J": B | C | D | E,
    "K": F | E | G1 | J | M,
    "L": F | E | D,
    "M": F | E | B | C | H | J,
    "N": F | E | B | C | H | M,
    "O": A | F | E | D | C | B,
    "P": A | F | B | G1 | G2 | E,
    "Q": A | F | E | D | C | B | M,
    "R": A | F | B | G1 | G2 | E | M,
    "S": A | F | G1 | G2 | C | D,
    "T": A | I | Lseg,
    "U": F | E | D | C | B,
    "V": F | B | K | M,
    "W": F | E | B | C | K | M,
    "X": H | J | K | M,
    "Y": H | J | Lseg,
    "Z": A | J | K | D,
    "-": G1 | G2,
}


def thick_line(x1: float, y1: float, x2: float, y2: float, w: float) -> list[tuple[float, float]]:
    dx, dy = x2 - x1, y2 - y1
    length = math.hypot(dx, dy)
    if length == 0:
        return []
    nx, ny = -dy / length * w / 2, dx / length * w / 2
    return [
        (x1 + nx, y1 + ny),
        (x2 + nx, y2 + ny),
        (x2 - nx, y2 - ny),
        (x1 - nx, y1 - ny),
    ]


def segments(bits: int) -> list[list[tuple[float, float]]]:
    g = GAP
    out: list[list[tuple[float, float]]] = []

    def add(x1, y1, x2, y2):
        poly = thick_line(x1, y1, x2, y2, TH)
        if poly:
            out.append(poly)

    if bits & A:
        add(L + g, T, R - g, T)
    if bits & D:
        add(L + g, B, R - g, B)
    if bits & G1:
        add(L + g, MY, CX - g, MY)
    if bits & G2:
        add(CX + g, MY, R - g, MY)
    if bits & F:
        add(L, T - g, L, MY + g)
    if bits & E:
        add(L, MY - g, L, B + g)
    if bits & B:
        add(R, T - g, R, MY + g)
    if bits & C:
        add(R, MY - g, R, B + g)
    if bits & I:
        add(CX, T - g, CX, MY + g)
    if bits & Lseg:
        add(CX, MY - g, CX, B + g)
    if bits & H:
        add(L + g * 2, T - g * 2, CX - g, MY + g)
    if bits & J:
        add(R - g * 2, T - g * 2, CX + g, MY + g)
    if bits & K:
        add(L + g * 2, B + g * 2, CX - g, MY - g)
    if bits & M:
        add(R - g * 2, B + g * 2, CX + g, MY - g)
    return out


def draw_glyph(bits: int):
    pen = TTGlyphPen(None)
    for poly in segments(bits):
        pen.moveTo(poly[0])
        for pt in poly[1:]:
            pen.lineTo(pt)
        pen.closePath()
    return pen.glyph()


def empty_glyph():
    return TTGlyphPen(None).glyph()


def build(path: Path) -> None:
    order = [".notdef", "space"]
    cmap = {0x20: "space"}
    glyphs = {".notdef": empty_glyph(), "space": empty_glyph()}
    metrics = {".notdef": (ADV, LSB), "space": (ADV, LSB)}

    for i, ch in enumerate("ABCDEFGHIJKLMNOPQRSTUVWXYZ"):
        name = f"uniE{i:03X}"
        order.append(name)
        cmap[0xE000 + i] = name
        glyphs[name] = draw_glyph(FONT[ch])
        metrics[name] = (ADV, LSB)

    for i, ch in enumerate("0123456789"):
        name = f"u1FBF{i}"
        order.append(name)
        cmap[0x1FBF0 + i] = name
        glyphs[name] = draw_glyph(FONT[ch])
        metrics[name] = (ADV, LSB)

    name = "hyphen.seg"
    order.append(name)
    cmap[0x2D] = name
    glyphs[name] = draw_glyph(FONT["-"])
    metrics[name] = (ADV, LSB)

    fb = FontBuilder(UPM, isTTF=True)
    fb.setupGlyphOrder(order)
    fb.setupCharacterMap(cmap)
    fb.setupGlyf(glyphs)
    fb.setupHorizontalMetrics(metrics)
    fb.setupHorizontalHeader(ascent=800, descent=-200)
    fb.setupOS2(sTypoAscender=800, sTypoDescender=-200, usWinAscent=800, usWinDescent=200)
    fb.setupPost()
    fb.setupNameTable(
        {
            "familyName": "Segmented Alpha",
            "styleName": "Regular",
            "uniqueFontIdentifier": "seg-lab:Segmented Alpha:Regular",
            "fullName": "Segmented Alpha Regular",
            "psName": "SegmentedAlpha-Regular",
            "version": "Version 1.000",
            "description": "14-segment A–Z at U+E000–U+E019; digits at U+1FBF0–U+1FBF9.",
        }
    )
    path.parent.mkdir(parents=True, exist_ok=True)
    fb.save(path)


if __name__ == "__main__":
    out = Path(__file__).with_name("SegmentedAlpha.ttf")
    build(out)
    print(f"wrote {out} ({out.stat().st_size} bytes)")
