import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { StageTransitionModal, isBackwardMove, isCloseStage } from "./stage-transition-modal";

describe("isBackwardMove", () => {
  it("returns true for backward moves", () => {
    expect(isBackwardMove("qualified", "new_lead")).toBe(true);
    expect(isBackwardMove("negotiation", "contacted")).toBe(true);
  });

  it("returns false for forward moves", () => {
    expect(isBackwardMove("new_lead", "contacted")).toBe(false);
    expect(isBackwardMove("proposal", "negotiation")).toBe(false);
  });

  it("returns false for same stage", () => {
    expect(isBackwardMove("qualified", "qualified")).toBe(false);
  });
});

describe("isCloseStage", () => {
  it("returns true for closed_won", () => {
    expect(isCloseStage("closed_won")).toBe(true);
  });

  it("returns true for closed_lost", () => {
    expect(isCloseStage("closed_lost")).toBe(true);
  });

  it("returns false for other stages", () => {
    expect(isCloseStage("new_lead")).toBe(false);
    expect(isCloseStage("qualified")).toBe(false);
    expect(isCloseStage("negotiation")).toBe(false);
  });
});

describe("StageTransitionModal", () => {
  const defaultProps = {
    currentStage: "qualified" as const,
    targetStage: "proposal" as const,
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
  };

  it("renders modal with title", () => {
    render(<StageTransitionModal {...defaultProps} />);
    expect(screen.getByTestId("stage-transition-modal")).toBeInTheDocument();
    expect(screen.getByTestId("transition-modal-title")).toHaveTextContent("Move to Proposal");
  });

  it("renders close title for close stages", () => {
    render(<StageTransitionModal {...defaultProps} targetStage="closed_won" />);
    expect(screen.getByTestId("transition-modal-title")).toHaveTextContent("Close as Closed Won");
  });

  it("shows reason field for backward moves", () => {
    render(
      <StageTransitionModal {...defaultProps} currentStage="negotiation" targetStage="contacted" />,
    );
    expect(screen.getByTestId("transition-reason-input")).toBeInTheDocument();
  });

  it("shows close reason field for close stages", () => {
    render(<StageTransitionModal {...defaultProps} targetStage="closed_lost" />);
    expect(screen.getByTestId("transition-reason-input")).toBeInTheDocument();
  });

  it("does not show reason for forward moves", () => {
    render(<StageTransitionModal {...defaultProps} />);
    expect(screen.queryByTestId("transition-reason-input")).not.toBeInTheDocument();
  });

  it("always shows comment field", () => {
    render(<StageTransitionModal {...defaultProps} />);
    expect(screen.getByTestId("transition-comment-input")).toBeInTheDocument();
  });

  it("calls onCancel when cancel clicked", () => {
    const onCancel = vi.fn();
    render(<StageTransitionModal {...defaultProps} onCancel={onCancel} />);
    fireEvent.click(screen.getByTestId("transition-cancel-btn"));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("calls onConfirm with stage on submit for forward move", () => {
    const onConfirm = vi.fn();
    render(<StageTransitionModal {...defaultProps} onConfirm={onConfirm} />);
    fireEvent.click(screen.getByTestId("transition-confirm-btn"));
    expect(onConfirm).toHaveBeenCalledWith({ stage: "proposal" });
  });

  it("disables confirm when reason required but empty", () => {
    render(<StageTransitionModal {...defaultProps} targetStage="closed_won" />);
    expect(screen.getByTestId("transition-confirm-btn")).toBeDisabled();
  });

  it("enables confirm after reason entered for close stage", () => {
    render(<StageTransitionModal {...defaultProps} targetStage="closed_won" />);
    fireEvent.change(screen.getByTestId("transition-reason-input"), {
      target: { value: "Deal completed" },
    });
    expect(screen.getByTestId("transition-confirm-btn")).not.toBeDisabled();
  });

  it("shows loading state", () => {
    render(<StageTransitionModal {...defaultProps} isLoading />);
    expect(screen.getByTestId("transition-confirm-btn")).toHaveTextContent("Saving...");
  });
});
