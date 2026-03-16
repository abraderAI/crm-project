import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

// Mock settings-api module.
vi.mock("@/lib/settings-api", () => ({
  fetchApiKeys: vi.fn(),
  createApiKey: vi.fn(),
  revokeApiKey: vi.fn(),
}));

// Mock clipboard module.
vi.mock("@/lib/clipboard", () => ({
  copyToClipboard: vi.fn().mockResolvedValue(undefined),
}));

import { ApiKeys } from "./api-keys";
import { fetchApiKeys, createApiKey, revokeApiKey } from "@/lib/settings-api";
import { copyToClipboard } from "@/lib/clipboard";

const mockFetchApiKeys = vi.mocked(fetchApiKeys);
const mockCopyToClipboard = vi.mocked(copyToClipboard);
const mockCreateApiKey = vi.mocked(createApiKey);
const mockRevokeApiKey = vi.mocked(revokeApiKey);

const FIXTURE_KEYS = [
  {
    id: "key-1",
    name: "Production Key",
    prefix: "deft_live_abc",
    created_at: "2026-01-15T10:00:00Z",
    last_used_at: "2026-03-10T14:30:00Z",
  },
  {
    id: "key-2",
    name: "Staging Key",
    prefix: "deft_live_xyz",
    created_at: "2026-02-01T08:00:00Z",
    last_used_at: null,
  },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockFetchApiKeys.mockResolvedValue(FIXTURE_KEYS);
});

describe("ApiKeys", () => {
  it("renders the API Keys heading", async () => {
    render(<ApiKeys token="test-token" />);
    expect(screen.getByText("API Keys")).toBeInTheDocument();
  });

  it("loads and displays existing keys", async () => {
    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByText("Production Key")).toBeInTheDocument();
    });
    expect(screen.getByText("Staging Key")).toBeInTheDocument();
    expect(screen.getByText("deft_live_abc...")).toBeInTheDocument();
    expect(screen.getByText("deft_live_xyz...")).toBeInTheDocument();
  });

  it("shows empty state when no keys exist", async () => {
    mockFetchApiKeys.mockResolvedValue([]);
    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByText(/no api keys/i)).toBeInTheDocument();
    });
  });

  it("opens create key modal on button click", async () => {
    const user = userEvent.setup();
    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByTestId("create-api-key-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("create-api-key-btn"));
    expect(screen.getByTestId("create-key-modal")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Key name")).toBeInTheDocument();
  });

  it("creates a key and shows the full key once", async () => {
    const user = userEvent.setup();
    mockCreateApiKey.mockResolvedValueOnce({
      id: "key-3",
      name: "New Key",
      prefix: "deft_live_new",
      key: "deft_live_new_full_secret_123",
      created_at: "2026-03-16T00:00:00Z",
    });

    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByTestId("create-api-key-btn")).toBeInTheDocument();
    });

    // Open modal and fill form.
    await user.click(screen.getByTestId("create-api-key-btn"));
    await user.type(screen.getByPlaceholderText("Key name"), "New Key");
    await user.click(screen.getByTestId("confirm-create-key-btn"));

    // Full key should be displayed.
    await waitFor(() => {
      expect(screen.getByTestId("created-key-value")).toHaveTextContent(
        "deft_live_new_full_secret_123",
      );
    });
    expect(mockCreateApiKey).toHaveBeenCalledWith("test-token", "New Key");
  });

  it("copies full key to clipboard on copy button click", async () => {
    const user = userEvent.setup();
    mockCreateApiKey.mockResolvedValueOnce({
      id: "key-3",
      name: "New Key",
      prefix: "deft_live_new",
      key: "deft_live_new_full_secret_123",
      created_at: "2026-03-16T00:00:00Z",
    });

    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByTestId("create-api-key-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("create-api-key-btn"));
    await user.type(screen.getByPlaceholderText("Key name"), "New Key");
    await user.click(screen.getByTestId("confirm-create-key-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("copy-key-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("copy-key-btn"));

    // Verify the clipboard utility was called with the full key.
    await waitFor(() => {
      expect(mockCopyToClipboard).toHaveBeenCalledWith("deft_live_new_full_secret_123");
    });
  });

  it("closes the created key modal and refreshes the list", async () => {
    const user = userEvent.setup();
    mockCreateApiKey.mockResolvedValueOnce({
      id: "key-3",
      name: "New Key",
      prefix: "deft_live_new",
      key: "deft_live_new_full_secret_123",
      created_at: "2026-03-16T00:00:00Z",
    });

    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByTestId("create-api-key-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("create-api-key-btn"));
    await user.type(screen.getByPlaceholderText("Key name"), "New Key");
    await user.click(screen.getByTestId("confirm-create-key-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("close-created-key-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("close-created-key-btn"));

    // The created key modal should be gone.
    expect(screen.queryByTestId("created-key-value")).not.toBeInTheDocument();
    // fetchApiKeys should have been called again to refresh.
    expect(mockFetchApiKeys).toHaveBeenCalledTimes(2);
  });

  it("shows revoke confirmation dialog", async () => {
    const user = userEvent.setup();
    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByText("Production Key")).toBeInTheDocument();
    });

    const revokeButtons = screen.getAllByTestId("revoke-key-btn");
    await user.click(revokeButtons[0]!);

    expect(screen.getByTestId("revoke-confirm-dialog")).toBeInTheDocument();
    expect(screen.getByText(/revoke this api key/i)).toBeInTheDocument();
  });

  it("revokes key on confirmation and removes row", async () => {
    const user = userEvent.setup();
    mockRevokeApiKey.mockResolvedValueOnce(undefined);
    // After revoke, fetch returns only one key.
    mockFetchApiKeys.mockResolvedValueOnce(FIXTURE_KEYS).mockResolvedValueOnce([FIXTURE_KEYS[1]!]);

    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByText("Production Key")).toBeInTheDocument();
    });

    const revokeButtons = screen.getAllByTestId("revoke-key-btn");
    await user.click(revokeButtons[0]!);
    await user.click(screen.getByTestId("confirm-revoke-btn"));

    expect(mockRevokeApiKey).toHaveBeenCalledWith("test-token", "key-1");
    await waitFor(() => {
      expect(screen.queryByText("Production Key")).not.toBeInTheDocument();
    });
  });

  it("cancels revoke dialog without deleting", async () => {
    const user = userEvent.setup();
    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByText("Production Key")).toBeInTheDocument();
    });

    const revokeButtons = screen.getAllByTestId("revoke-key-btn");
    await user.click(revokeButtons[0]!);
    await user.click(screen.getByTestId("cancel-revoke-btn"));

    expect(mockRevokeApiKey).not.toHaveBeenCalled();
    expect(screen.queryByTestId("revoke-confirm-dialog")).not.toBeInTheDocument();
    expect(screen.getByText("Production Key")).toBeInTheDocument();
  });

  it("disables create button when name is empty", async () => {
    const user = userEvent.setup();
    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByTestId("create-api-key-btn")).toBeInTheDocument();
    });
    await user.click(screen.getByTestId("create-api-key-btn"));
    expect(screen.getByTestId("confirm-create-key-btn")).toBeDisabled();
  });

  it("displays last used date when available", async () => {
    render(<ApiKeys token="test-token" />);
    await waitFor(() => {
      expect(screen.getByText("Production Key")).toBeInTheDocument();
    });
    // The second key has no last_used_at, so "Never" appears in a "Last used: Never" span.
    const row = screen.getByTestId("api-key-row-key-2");
    expect(row).toHaveTextContent("Never");
  });
});
