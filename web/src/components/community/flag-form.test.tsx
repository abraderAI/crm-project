import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { FlagForm } from "./flag-form";

describe("FlagForm", () => {
  it("renders the report heading and icon", () => {
    render(<FlagForm onSubmit={vi.fn()} />);
    expect(screen.getByText("Report Content")).toBeInTheDocument();
    expect(screen.getByTestId("flag-icon")).toBeInTheDocument();
  });

  it("renders all predefined reasons", () => {
    render(<FlagForm onSubmit={vi.fn()} />);
    expect(screen.getByTestId("flag-reason-spam-or-misleading")).toBeInTheDocument();
    expect(screen.getByTestId("flag-reason-harassment-or-abuse")).toBeInTheDocument();
    expect(screen.getByTestId("flag-reason-off-topic-content")).toBeInTheDocument();
    expect(screen.getByTestId("flag-reason-inappropriate-language")).toBeInTheDocument();
    expect(screen.getByTestId("flag-reason-other")).toBeInTheDocument();
  });

  it("submits with a predefined reason", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<FlagForm onSubmit={onSubmit} />);

    await user.click(screen.getByTestId("flag-reason-spam-or-misleading"));
    await user.click(screen.getByTestId("flag-submit-btn"));
    expect(onSubmit).toHaveBeenCalledWith("Spam or misleading");
  });

  it("shows error when submitting without a reason", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<FlagForm onSubmit={onSubmit} />);

    await user.click(screen.getByTestId("flag-submit-btn"));
    expect(screen.getByTestId("flag-error")).toHaveTextContent("Please select or enter a reason.");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("shows custom reason textarea when Other is selected", async () => {
    const user = userEvent.setup();
    render(<FlagForm onSubmit={vi.fn()} />);

    expect(screen.queryByTestId("flag-custom-reason")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("flag-reason-other"));
    expect(screen.getByTestId("flag-custom-reason")).toBeInTheDocument();
  });

  it("submits with custom reason", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<FlagForm onSubmit={onSubmit} />);

    await user.click(screen.getByTestId("flag-reason-other"));
    await user.type(screen.getByTestId("flag-custom-reason"), "Custom issue description");
    await user.click(screen.getByTestId("flag-submit-btn"));
    expect(onSubmit).toHaveBeenCalledWith("Custom issue description");
  });

  it("shows error when Other selected but custom reason is empty", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<FlagForm onSubmit={onSubmit} />);

    await user.click(screen.getByTestId("flag-reason-other"));
    await user.click(screen.getByTestId("flag-submit-btn"));
    expect(screen.getByTestId("flag-error")).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("renders cancel button when onCancel provided", () => {
    render(<FlagForm onSubmit={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByTestId("flag-cancel-btn")).toBeInTheDocument();
  });

  it("does not render cancel button when onCancel not provided", () => {
    render(<FlagForm onSubmit={vi.fn()} />);
    expect(screen.queryByTestId("flag-cancel-btn")).not.toBeInTheDocument();
  });

  it("calls onCancel when cancel clicked", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(<FlagForm onSubmit={vi.fn()} onCancel={onCancel} />);

    await user.click(screen.getByTestId("flag-cancel-btn"));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("shows loading state", () => {
    render(<FlagForm onSubmit={vi.fn()} loading={true} />);
    expect(screen.getByTestId("flag-submit-btn")).toHaveTextContent("Submitting...");
    expect(screen.getByTestId("flag-submit-btn")).toBeDisabled();
  });

  it("clears error on successful submission", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<FlagForm onSubmit={onSubmit} />);

    // Submit without reason to trigger error
    await user.click(screen.getByTestId("flag-submit-btn"));
    expect(screen.getByTestId("flag-error")).toBeInTheDocument();

    // Select reason and submit again
    await user.click(screen.getByTestId("flag-reason-spam-or-misleading"));
    await user.click(screen.getByTestId("flag-submit-btn"));
    expect(screen.queryByTestId("flag-error")).not.toBeInTheDocument();
  });
});
