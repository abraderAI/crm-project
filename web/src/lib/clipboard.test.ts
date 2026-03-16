import { describe, expect, it, vi, beforeEach } from "vitest";

import { copyToClipboard } from "./clipboard";

describe("copyToClipboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("calls navigator.clipboard.writeText with the given text", async () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      get: () => ({ writeText: writeTextMock }),
      configurable: true,
    });

    await copyToClipboard("test-text");
    expect(writeTextMock).toHaveBeenCalledWith("test-text");
  });

  it("propagates errors from clipboard API", async () => {
    const writeTextMock = vi.fn().mockRejectedValue(new Error("Permission denied"));
    Object.defineProperty(navigator, "clipboard", {
      get: () => ({ writeText: writeTextMock }),
      configurable: true,
    });

    await expect(copyToClipboard("test-text")).rejects.toThrow("Permission denied");
  });
});
