import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { ExportButton } from "./export-button";

let fetchSpy: ReturnType<typeof vi.spyOn>;
let createObjectURLSpy: ReturnType<typeof vi.spyOn>;
let revokeObjectURLSpy: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response("col1,col2\na,b\n", {
      status: 200,
      headers: { "Content-Type": "text/csv" },
    }),
  );
  createObjectURLSpy = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:mock-url");
  revokeObjectURLSpy = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
});

afterEach(() => {
  fetchSpy.mockRestore();
  createObjectURLSpy.mockRestore();
  revokeObjectURLSpy.mockRestore();
});

describe("ExportButton", () => {
  it("renders button", () => {
    render(<ExportButton url="http://example.com/export" filename="test.csv" />);
    expect(screen.getByTestId("export-button")).toBeInTheDocument();
    expect(screen.getByTestId("export-button")).toHaveTextContent("Export CSV");
  });

  it("renders download icon by default", () => {
    render(<ExportButton url="http://example.com/export" filename="test.csv" />);
    expect(screen.getByTestId("export-icon")).toBeInTheDocument();
    expect(screen.queryByTestId("export-spinner")).not.toBeInTheDocument();
  });

  it("triggers download on click", async () => {
    const user = userEvent.setup();

    render(<ExportButton url="http://example.com/export" filename="report.csv" />);
    await user.click(screen.getByTestId("export-button"));

    await waitFor(() => {
      expect(fetchSpy).toHaveBeenCalledWith("http://example.com/export");
    });
    expect(createObjectURLSpy).toHaveBeenCalled();
    expect(revokeObjectURLSpy).toHaveBeenCalledWith("blob:mock-url");
  });

  it("shows error on fetch failure", async () => {
    const user = userEvent.setup();
    fetchSpy.mockRejectedValueOnce(new Error("Network failure"));

    render(<ExportButton url="http://example.com/export" filename="test.csv" />);
    await user.click(screen.getByTestId("export-button"));

    await waitFor(() => {
      expect(screen.getByTestId("export-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("export-error")).toHaveTextContent("Network failure");
  });

  it("shows error on non-ok response", async () => {
    const user = userEvent.setup();
    fetchSpy.mockResolvedValueOnce(
      new Response("", { status: 500, statusText: "Internal Server Error" }),
    );

    render(<ExportButton url="http://example.com/export" filename="test.csv" />);
    await user.click(screen.getByTestId("export-button"));

    await waitFor(() => {
      expect(screen.getByTestId("export-error")).toBeInTheDocument();
    });
    expect(screen.getByTestId("export-error")).toHaveTextContent("Export failed: 500");
  });

  it("renders the wrapper container", () => {
    render(<ExportButton url="http://example.com/export" filename="test.csv" />);
    expect(screen.getByTestId("export-button-wrapper")).toBeInTheDocument();
  });

  it("does not show error initially", () => {
    render(<ExportButton url="http://example.com/export" filename="test.csv" />);
    expect(screen.queryByTestId("export-error")).not.toBeInTheDocument();
  });
});
