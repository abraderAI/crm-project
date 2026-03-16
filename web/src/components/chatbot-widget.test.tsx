import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach, type Mock } from "vitest";

// Mock Clerk auth — provide token for authenticated users, null for anonymous.
const mockGetToken = vi.fn<() => Promise<string | null>>();
vi.mock("@clerk/nextjs", () => ({
  useAuth: () => ({ getToken: mockGetToken }),
}));

// Mock the api-client to intercept fetch calls.
vi.mock("@/lib/api-client", () => ({
  buildUrl: (path: string) => `http://localhost:8080/v1${path}`,
  buildHeaders: (token?: string | null) => {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      Accept: "application/json",
    };
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
    return headers;
  },
}));

import { ChatbotWidget } from "./chatbot-widget";

describe("ChatbotWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetToken.mockResolvedValue("test-token");
    // Mock global fetch.
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ reply: "Hello from bot!" }),
    });
  });

  it("renders the floating chat bubble", () => {
    render(<ChatbotWidget />);
    expect(screen.getByTestId("chatbot-bubble")).toBeInTheDocument();
  });

  it("does not render the chat panel initially", () => {
    render(<ChatbotWidget />);
    expect(screen.queryByTestId("chatbot-panel")).not.toBeInTheDocument();
  });

  it("opens the chat panel on bubble click", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    expect(screen.getByTestId("chatbot-panel")).toBeInTheDocument();
  });

  it("closes the chat panel when close button is clicked", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));
    expect(screen.getByTestId("chatbot-panel")).toBeInTheDocument();

    await user.click(screen.getByTestId("chatbot-close"));
    expect(screen.queryByTestId("chatbot-panel")).not.toBeInTheDocument();
  });

  it("renders message input and send button when panel is open", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    expect(screen.getByTestId("chatbot-input")).toBeInTheDocument();
    expect(screen.getByTestId("chatbot-send")).toBeInTheDocument();
  });

  it("sends a message to POST /v1/chat/message", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "Hello bot");
    await user.click(screen.getByTestId("chatbot-send"));

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "http://localhost:8080/v1/chat/message",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ message: "Hello bot" }),
        }),
      );
    });
  });

  it("sends message on Enter key press", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "Hello Enter{Enter}");

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "http://localhost:8080/v1/chat/message",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ message: "Hello Enter" }),
        }),
      );
    });
  });

  it("displays user message in conversation history", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "My message");
    await user.click(screen.getByTestId("chatbot-send"));

    expect(screen.getByText("My message")).toBeInTheDocument();
  });

  it("displays bot reply in conversation history", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "Hi");
    await user.click(screen.getByTestId("chatbot-send"));

    await waitFor(() => {
      expect(screen.getByText("Hello from bot!")).toBeInTheDocument();
    });
  });

  it("clears the input field after sending", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input") as HTMLInputElement;
    await user.type(input, "Test clear");
    await user.click(screen.getByTestId("chatbot-send"));

    expect(input.value).toBe("");
  });

  it("does not send empty messages", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));
    await user.click(screen.getByTestId("chatbot-send"));

    expect(global.fetch).not.toHaveBeenCalled();
  });

  it("does not send whitespace-only messages", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "   ");
    await user.click(screen.getByTestId("chatbot-send"));

    expect(global.fetch).not.toHaveBeenCalled();
  });

  it("includes auth token in request headers when authenticated", async () => {
    mockGetToken.mockResolvedValue("my-auth-token");
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "Auth test");
    await user.click(screen.getByTestId("chatbot-send"));

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        "http://localhost:8080/v1/chat/message",
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: "Bearer my-auth-token",
          }),
        }),
      );
    });
  });

  it("sends request without auth header for anonymous users", async () => {
    mockGetToken.mockResolvedValue(null);
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "Anon test");
    await user.click(screen.getByTestId("chatbot-send"));

    await waitFor(() => {
      const fetchMock = global.fetch as Mock;
      const [, opts] = fetchMock.mock.calls[0] as [string, RequestInit];
      const headers = opts.headers as Record<string, string>;
      expect(headers["Authorization"]).toBeUndefined();
    });
  });

  it("shows error message when API call fails", async () => {
    (global.fetch as Mock).mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: async () => ({ title: "Internal Server Error" }),
    });

    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "Error test");
    await user.click(screen.getByTestId("chatbot-send"));

    await waitFor(() => {
      expect(screen.getByTestId("chatbot-error")).toBeInTheDocument();
    });
  });

  it("disables send button while loading", async () => {
    // Make fetch hang indefinitely.
    (global.fetch as Mock).mockImplementationOnce(() => new Promise(() => {}));

    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "Loading test");
    await user.click(screen.getByTestId("chatbot-send"));

    expect(screen.getByTestId("chatbot-send")).toBeDisabled();
  });

  it("renders greeting message in panel header", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    await user.click(screen.getByTestId("chatbot-bubble"));

    expect(screen.getByTestId("chatbot-header")).toHaveTextContent("DEFT Assistant");
  });

  it("maintains conversation history across toggle", async () => {
    const user = userEvent.setup();
    render(<ChatbotWidget />);

    // Open, send message, close, reopen.
    await user.click(screen.getByTestId("chatbot-bubble"));
    const input = screen.getByTestId("chatbot-input");
    await user.type(input, "Persist test");
    await user.click(screen.getByTestId("chatbot-send"));

    await waitFor(() => {
      expect(screen.getByText("Hello from bot!")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("chatbot-close"));
    await user.click(screen.getByTestId("chatbot-bubble"));

    // History should still be there.
    expect(screen.getByText("Persist test")).toBeInTheDocument();
    expect(screen.getByText("Hello from bot!")).toBeInTheDocument();
  });
});
