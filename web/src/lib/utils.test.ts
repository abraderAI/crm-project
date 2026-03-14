import { describe, expect, it } from "vitest";
import { cn, parseMetadata } from "./utils";

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("handles conditional classes", () => {
    expect(cn("base", false && "hidden", "extra")).toBe("base extra");
  });

  it("deduplicates conflicting Tailwind classes", () => {
    expect(cn("p-4", "p-2")).toBe("p-2");
  });

  it("handles undefined and null", () => {
    expect(cn("base", undefined, null)).toBe("base");
  });

  it("handles empty inputs", () => {
    expect(cn()).toBe("");
  });

  it("handles arrays", () => {
    expect(cn(["foo", "bar"])).toBe("foo bar");
  });

  it("handles objects", () => {
    expect(cn({ foo: true, bar: false, baz: true })).toBe("foo baz");
  });
});

describe("parseMetadata", () => {
  it("returns empty object for undefined", () => {
    expect(parseMetadata(undefined)).toEqual({});
  });

  it("returns empty object for empty string", () => {
    expect(parseMetadata("")).toEqual({});
  });

  it("returns the object as-is when given a Record", () => {
    const input = { tier: "pro", count: 5 };
    expect(parseMetadata(input)).toBe(input);
  });

  it("parses valid JSON string to object", () => {
    expect(parseMetadata('{"status":"open"}')).toEqual({ status: "open" });
  });

  it("returns empty object for invalid JSON", () => {
    expect(parseMetadata("not-json")).toEqual({});
  });

  it("returns empty object for JSON array", () => {
    expect(parseMetadata("[1,2,3]")).toEqual({});
  });

  it("returns empty object for JSON primitive", () => {
    expect(parseMetadata("42")).toEqual({});
  });
});
