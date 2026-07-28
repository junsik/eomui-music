#!/usr/bin/env python3
"""아이콘 생성 — 빨간 음표.

만드는 것:
  extension/icons/icon{16,48,128}.png   크롬 확장 아이콘
  app/icon.ico                          트레이 아이콘 (exe에 embed)

    pip install pillow
    python scripts/generate_icons.py
"""

from pathlib import Path

from PIL import Image, ImageDraw

# 이 스크립트 위치를 기준으로 잡는다. 어디서 실행하든 동작한다.
ROOT = Path(__file__).resolve().parent.parent
EXT_ICONS = ROOT / "extension" / "icons"
APP_ICO = ROOT / "app" / "icon.ico"

PNG_SIZES = [16, 48, 128]
ICO_SIZES = [16, 24, 32, 48, 256]

ROSE = (225, 29, 72, 255)  # rose-600
WHITE = (255, 255, 255, 255)


def create_music_icon(size: int) -> Image.Image:
    """주어진 크기의 빨간 음표 아이콘을 만든다."""
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # 둥근 사각형 배경
    padding = int(size * 0.05)
    draw.rounded_rectangle(
        [padding, padding, size - padding, size - padding],
        radius=int(size * 0.22),
        fill=ROSE,
    )

    center = size / 2
    scale = size / 128.0

    # 음표 머리
    note_x = center - 18 * scale
    note_y = center + 16 * scale
    note_w = 22 * scale
    note_h = 16 * scale
    draw.ellipse(
        [
            note_x - note_w / 2,
            note_y - note_h / 2,
            note_x + note_w / 2,
            note_y + note_h / 2,
        ],
        fill=WHITE,
    )

    # 기둥
    stem_x = note_x + note_w / 2 - 2 * scale
    stem_top = center - 28 * scale
    draw.line(
        [(stem_x, note_y - note_h / 2), (stem_x, stem_top)],
        fill=WHITE,
        width=max(2, int(3 * scale)),
    )

    # 깃발
    draw.polygon(
        [
            (stem_x, stem_top),
            (stem_x + 20 * scale, stem_top + 8 * scale),
            (stem_x + 4 * scale, stem_top + 20 * scale),
        ],
        fill=WHITE,
    )

    return img


def main() -> None:
    EXT_ICONS.mkdir(parents=True, exist_ok=True)
    for size in PNG_SIZES:
        path = EXT_ICONS / f"icon{size}.png"
        create_music_icon(size).save(path, "PNG")
        print(f"생성: {path.relative_to(ROOT)}")

    # .ico 는 여러 크기를 한 파일에 담는다. 가장 큰 것에서 줄여 넣는다.
    APP_ICO.parent.mkdir(parents=True, exist_ok=True)
    create_music_icon(max(ICO_SIZES)).save(
        APP_ICO, format="ICO", sizes=[(s, s) for s in ICO_SIZES]
    )
    print(f"생성: {APP_ICO.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
