/* --------------------------------------------------------------------------
   Artwork colour
   --------------------------------------------------------------------------
   The peek's wash used to be one fixed green, which said "media" and nothing
   else. Taking it from the album art instead makes the bar belong to the thing
   playing, and costs one small canvas read per track.

   The sampling is deliberately crude — a coarse histogram over a downscaled
   copy — because the goal is a wash behind text, not a palette. What it does
   care about is not returning a colour that ruins the text on top of it, which
   is why near-black, near-white and near-grey buckets are discarded before the
   winner is picked.
   -------------------------------------------------------------------------- */

/** Edge length the artwork is sampled at. Small enough to be free, big enough to be stable. */
export const SAMPLE_SIZE = 32;

/** Bits dropped per channel when bucketing, so near-identical pixels group. */
const BUCKET_SHIFT = 4;

/** Below this the pixel is shadow and tells us nothing about the art's colour. */
const MIN_LUMINANCE = 0.12;
/** Above this it is a highlight, and a wash of it would be a white bar. */
const MAX_LUMINANCE = 0.92;
/** Below this a pixel is grey, and a grey accent is the same as no accent. */
const MIN_SATURATION = 0.18;

export interface Rgb {
  r: number;
  g: number;
  b: number;
}

export function relativeLuminance({ r, g, b }: Rgb) {
  return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
}

export function saturation({ r, g, b }: Rgb) {
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  return max === 0 ? 0 : (max - min) / max;
}

/** Whether a pixel is worth counting towards the accent. */
export function isUsableSample(pixel: Rgb) {
  const luminance = relativeLuminance(pixel);
  return (
    luminance >= MIN_LUMINANCE &&
    luminance <= MAX_LUMINANCE &&
    saturation(pixel) >= MIN_SATURATION
  );
}

/**
 * The most common usable colour in an RGBA buffer, or null when there is none.
 *
 * Null is a real answer: a black-and-white cover has no accent to take, and a
 * wash invented from its greys would be worse than the default.
 */
export function dominantColor(pixels: Uint8ClampedArray): Rgb | null {
  const counts = new Map<number, { count: number; r: number; g: number; b: number }>();

  for (let index = 0; index < pixels.length; index += 4) {
    // A transparent pixel is padding around a non-square cover, not colour.
    if (pixels[index + 3] < 128) continue;

    const pixel = { r: pixels[index], g: pixels[index + 1], b: pixels[index + 2] };
    if (!isUsableSample(pixel)) continue;

    const key =
      ((pixel.r >> BUCKET_SHIFT) << 8) |
      ((pixel.g >> BUCKET_SHIFT) << 4) |
      (pixel.b >> BUCKET_SHIFT);

    const bucket = counts.get(key);
    if (bucket) {
      bucket.count += 1;
      bucket.r += pixel.r;
      bucket.g += pixel.g;
      bucket.b += pixel.b;
    } else {
      counts.set(key, { count: 1, ...pixel });
    }
  }

  let winner: { count: number; r: number; g: number; b: number } | null = null;
  for (const bucket of counts.values()) {
    if (!winner || bucket.count > winner.count) winner = bucket;
  }
  if (!winner) return null;

  // The bucket's average rather than its key, so the colour is one that was
  // actually in the image instead of the corner of the range it fell in.
  return {
    r: Math.round(winner.r / winner.count),
    g: Math.round(winner.g / winner.count),
    b: Math.round(winner.b / winner.count),
  };
}

export function toCssColor({ r, g, b }: Rgb) {
  return `rgb(${r} ${g} ${b})`;
}
