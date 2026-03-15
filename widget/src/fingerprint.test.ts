import { describe, it, expect } from "vitest";
import { collectComponents, fnv1aHash, generateFingerprint } from "./fingerprint";

describe("fnv1aHash", () => {
  it("returns an 8-character hex string", () => {
    const hash = fnv1aHash("test");
    expect(hash).toMatch(/^[0-9a-f]{8}$/);
  });

  it("is deterministic", () => {
    expect(fnv1aHash("hello")).toBe(fnv1aHash("hello"));
  });

  it("produces different hashes for different inputs", () => {
    expect(fnv1aHash("a")).not.toBe(fnv1aHash("b"));
  });

  it("handles empty string", () => {
    const hash = fnv1aHash("");
    expect(hash).toMatch(/^[0-9a-f]{8}$/);
  });

  it("handles unicode", () => {
    const hash = fnv1aHash("日本語");
    expect(hash).toMatch(/^[0-9a-f]{8}$/);
  });

  it("handles very long strings", () => {
    const hash = fnv1aHash("x".repeat(100000));
    expect(hash).toMatch(/^[0-9a-f]{8}$/);
  });

  it("handles special characters", () => {
    expect(fnv1aHash("!@#$%^&*()")).toMatch(/^[0-9a-f]{8}$/);
    expect(fnv1aHash("\n\t\r")).toMatch(/^[0-9a-f]{8}$/);
    expect(fnv1aHash("\0")).toMatch(/^[0-9a-f]{8}$/);
  });
});

describe("collectComponents", () => {
  it("returns all required fields", () => {
    const components = collectComponents();
    expect(components).toHaveProperty("userAgent");
    expect(components).toHaveProperty("language");
    expect(components).toHaveProperty("screenResolution");
    expect(components).toHaveProperty("timezoneOffset");
    expect(components).toHaveProperty("colorDepth");
    expect(components).toHaveProperty("hardwareConcurrency");
    expect(components).toHaveProperty("platform");
    expect(components).toHaveProperty("canvasHash");
  });

  it("returns string types for string fields", () => {
    const components = collectComponents();
    expect(typeof components.userAgent).toBe("string");
    expect(typeof components.language).toBe("string");
    expect(typeof components.screenResolution).toBe("string");
    expect(typeof components.platform).toBe("string");
    expect(typeof components.canvasHash).toBe("string");
  });

  it("returns number types for numeric fields", () => {
    const components = collectComponents();
    expect(typeof components.timezoneOffset).toBe("number");
    expect(typeof components.colorDepth).toBe("number");
    expect(typeof components.hardwareConcurrency).toBe("number");
  });
});

describe("generateFingerprint", () => {
  it("returns an 8-character hex string", () => {
    const fp = generateFingerprint();
    expect(fp).toMatch(/^[0-9a-f]{8}$/);
  });

  it("is deterministic within same environment", () => {
    expect(generateFingerprint()).toBe(generateFingerprint());
  });
});

describe("canvas fingerprint edge cases", () => {
  it("handles missing canvas context gracefully", () => {
    // jsdom doesn't implement canvas, so getContext returns null.
    // The fingerprint module handles this with the "no-ctx" or "canvas-error" fallback.
    const components = collectComponents();
    // Should be one of the fallback values.
    expect(["no-canvas", "no-ctx", "canvas-error"]).toContain(components.canvasHash);
  });

  it("still generates a valid fingerprint without canvas", () => {
    const fp = generateFingerprint();
    expect(fp).toMatch(/^[0-9a-f]{8}$/);
    expect(fp.length).toBe(8);
  });
});
