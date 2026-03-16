import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ResetConfirmDialog } from "./reset-confirm-dialog";

describe("ResetConfirmDialog", () => {
  it("does not render when closed", () => {
    render(<ResetConfirmDialog open={false} onConfirm={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.queryByTestId("reset-confirm-dialog")).not.toBeInTheDocument();
  });

  it("renders dialog when open", () => {
    render(<ResetConfirmDialog open={true} onConfirm={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByTestId("reset-confirm-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("reset-dialog-title")).toHaveTextContent("Reset to defaults?");
    expect(screen.getByTestId("reset-dialog-description")).toBeInTheDocument();
  });

  it("has correct ARIA attributes", () => {
    render(<ResetConfirmDialog open={true} onConfirm={vi.fn()} onCancel={vi.fn()} />);
    const dialog = screen.getByTestId("reset-confirm-dialog");
    expect(dialog).toHaveAttribute("role", "dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(dialog).toHaveAttribute("aria-label", "Confirm reset");
  });

  it("calls onConfirm when confirm button is clicked", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(<ResetConfirmDialog open={true} onConfirm={onConfirm} onCancel={vi.fn()} />);

    await user.click(screen.getByTestId("reset-dialog-confirm"));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel when cancel button is clicked", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(<ResetConfirmDialog open={true} onConfirm={vi.fn()} onCancel={onCancel} />);

    await user.click(screen.getByTestId("reset-dialog-cancel"));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("shows cancel and confirm buttons", () => {
    render(<ResetConfirmDialog open={true} onConfirm={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByTestId("reset-dialog-cancel")).toHaveTextContent("Cancel");
    expect(screen.getByTestId("reset-dialog-confirm")).toHaveTextContent("Reset");
  });
});
