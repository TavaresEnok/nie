"""Generate an image-only synthetic PDF for real OCR integration tests.

Development dependency: Pillow and a local TrueType font. No external content.
"""
import argparse
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

parser = argparse.ArgumentParser()
parser.add_argument("--font", default="/usr/share/fonts/truetype/lato/Lato-Bold.ttf")
parser.add_argument("--output", default="internal/testfixture/scanned.pdf")
args = parser.parse_args()
image = Image.new("RGB", (1800, 450), "white")
draw = ImageDraw.Draw(image)
font = ImageFont.truetype(args.font, 92)
draw.text((90, 100), "BGP NETWORK ENGINE", fill="black", font=font)
draw.text((90, 235), "SYNTHETIC OCR TEST PAGE", fill="black", font=ImageFont.truetype(args.font, 58))
target = Path(args.output)
target.parent.mkdir(parents=True, exist_ok=True)
image.save(target, "PDF", resolution=150.0, title="Synthetic image-only OCR fixture")
