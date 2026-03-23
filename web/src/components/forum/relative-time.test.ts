import { describe, it, expect, vi, afterEach } from "vitest";
import { relativeTime } from "./relative-time";

describe("relativeTime", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("returns 'just now' for timestamps within 60 seconds", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-23T12:00:30Z"));
    expect(relativeTime("2026-03-23T12:00:00Z")).toBe("just now");
  });

  it("returns minutes for timestamps within an hour", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-23T12:15:00Z"));
    expect(relativeTime("2026-03-23T12:00:00Z")).toBe("15m ago");
  });

  it("returns hours for timestamps within a day", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-23T15:00:00Z"));
    expect(relativeTime("2026-03-23T12:00:00Z")).toBe("3h ago");
  });

  it("returns 'yesterday' for 1 day ago", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-24T12:00:00Z"));
    expect(relativeTime("2026-03-23T12:00:00Z")).toBe("yesterday");
  });

  it("returns days for timestamps within a month", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-30T12:00:00Z"));
    expect(relativeTime("2026-03-23T12:00:00Z")).toBe("7d ago");
  });

  it("returns months for timestamps within a year", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-06-23T12:00:00Z"));
    expect(relativeTime("2026-03-23T12:00:00Z")).toBe("3mo ago");
  });

  it("returns years for old timestamps", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2028-03-23T12:00:00Z"));
    expect(relativeTime("2026-03-23T12:00:00Z")).toBe("2y ago");
  });
});
