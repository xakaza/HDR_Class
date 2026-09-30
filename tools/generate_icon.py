#!/usr/bin/env python3
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
N = 512
im = Image.new('RGBA', (N, N), (0, 0, 0, 0))
px = im.load()
mask = Image.new('L', (N, N), 0)
md = ImageDraw.Draw(mask)
md.rounded_rectangle((18, 18, N - 18, N - 18), radius=112, fill=255)

for y in range(N):
    t = y / (N - 1)
    for x in range(N):
        if mask.getpixel((x, y)):
            u = x / (N - 1)
            r = int(16 + 14 * (1 - t) + 3 * u)
            g = int(29 + 32 * (1 - t) + 14 * u)
            b = int(48 + 55 * (1 - t) + 42 * u)
            px[x, y] = (r, g, b, 255)

d = ImageDraw.Draw(im)
d.rounded_rectangle((74, 68, 438, 444), radius=72, fill=(11, 20, 34, 225), outline=(72, 176, 255, 155), width=5)
for x0, y0, x1, y1, alpha in [
    (112, 128, 400, 222, 255),
    (96, 210, 416, 304, 235),
    (112, 292, 400, 386, 215),
]:
    d.rounded_rectangle((x0, y0, x1, y1), radius=26, fill=(29, 45, 67, alpha), outline=(82, 185, 255, 210), width=4)

for y in range(148, 370, 34):
    d.rounded_rectangle((246, y, 266, y + 18), radius=5, fill=(87, 203, 255, 255))

font = None
for candidate in [
    '/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf',
    '/usr/share/fonts/truetype/liberation2/LiberationSans-Bold.ttf',
    'C:/Windows/Fonts/seguisb.ttf',
]:
    try:
        font = ImageFont.truetype(candidate, 72)
        break
    except OSError:
        pass

if font:
    label = 'HDR'
    box = d.textbbox((0, 0), label, font=font)
    tw = box[2] - box[0]
    th = box[3] - box[1]
    d.text(((N - tw) // 2, 145 - th // 2), label, font=font, fill=(238, 248, 255, 255))

cyan = (88, 205, 255, 255)
d.line((168, 338, 210, 338), fill=cyan, width=12)
d.polygon([(210, 320), (236, 338), (210, 356)], fill=cyan)
d.line((344, 360, 302, 360), fill=(157, 212, 255, 255), width=12)
d.polygon([(302, 342), (276, 360), (302, 378)], fill=(157, 212, 255, 255))

(ROOT / 'assets').mkdir(parents=True, exist_ok=True)
(ROOT / 'source').mkdir(parents=True, exist_ok=True)
im.resize((256, 256), Image.Resampling.LANCZOS).save(ROOT / 'assets' / 'hdrstudio_256.png')
im.save(ROOT / 'assets' / 'hdrstudio_512.png')
im.save(ROOT / 'source' / 'hdrstudio.ico', format='ICO', sizes=[(16,16),(24,24),(32,32),(48,48),(64,64),(128,128),(256,256)])
print('HDR Studio icon assets generated.')
