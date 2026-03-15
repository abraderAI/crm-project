/**
 * Browser fingerprinting module.
 * Generates a stable hash from browser properties for visitor identification.
 * Uses canvas, screen, and navigator properties — no third-party dependencies.
 */

/** Components collected for fingerprinting. */
export interface FingerprintComponents {
  userAgent: string;
  language: string;
  screenResolution: string;
  timezoneOffset: number;
  colorDepth: number;
  hardwareConcurrency: number;
  platform: string;
  canvasHash: string;
}

/** Collect browser fingerprint components. */
export function collectComponents(): FingerprintComponents {
  /* v8 ignore next 2 -- typeof checks always true in browser/jsdom; false only in Node SSR */
  const nav = typeof navigator !== "undefined" ? navigator : undefined;
  const scr = typeof screen !== "undefined" ? screen : undefined;

  return {
    userAgent: nav?.userAgent ?? "unknown",
    language: nav?.language ?? "unknown",
    screenResolution: scr ? `${scr.width}x${scr.height}` : "0x0",
    timezoneOffset: new Date().getTimezoneOffset(),
    colorDepth: scr?.colorDepth ?? 0,
    hardwareConcurrency: nav?.hardwareConcurrency ?? 0,
    platform: nav?.platform ?? "unknown",
    canvasHash: getCanvasHash(),
  };
}

/** Generate a hash string from a canvas element. */
function getCanvasHash(): string {
  /* v8 ignore next -- typeof check always true in browser/jsdom */
  if (typeof document === "undefined") return "no-canvas";
  try {
    const canvas = document.createElement("canvas");
    const ctx = canvas.getContext("2d");
    if (!ctx) return "no-ctx";
    /* v8 ignore start -- canvas drawing requires native canvas, unavailable in jsdom */
    canvas.width = 200;
    canvas.height = 50;
    ctx.textBaseline = "top";
    ctx.font = "14px Arial";
    ctx.fillStyle = "#f60";
    ctx.fillRect(0, 0, 200, 50);
    ctx.fillStyle = "#069";
    ctx.fillText("CRM Widget FP", 2, 15);
    return canvas.toDataURL().slice(-32);
    /* v8 ignore stop */
  } catch /* v8 ignore next */ {
    return "canvas-error";
  }
}

/** Simple FNV-1a hash for strings, returns hex. */
export function fnv1aHash(input: string): string {
  let hash = 0x811c9dc5;
  for (let i = 0; i < input.length; i++) {
    hash ^= input.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  return (hash >>> 0).toString(16).padStart(8, "0");
}

/** Generate the full fingerprint hash. */
export function generateFingerprint(): string {
  const components = collectComponents();
  const raw = JSON.stringify(components);
  return fnv1aHash(raw);
}
