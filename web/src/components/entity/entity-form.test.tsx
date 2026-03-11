import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { EntityForm } from "./entity-form";

describe("EntityForm", () => {
  it("renders form with name and description inputs", () => {
    render(<EntityForm entityLabel="Organization" onSubmit={vi.fn()} />);
    expect(screen.getByTestId("entity-form")).toBeInTheDocument();
    expect(screen.getByTestId("entity-name-input")).toBeInTheDocument();
    expect(screen.getByTestId("entity-desc-input")).toBeInTheDocument();
  });

  it("shows create button text when no initial values", () => {
    render(<EntityForm entityLabel="Organization" onSubmit={vi.fn()} />);
    expect(screen.getByTestId("entity-submit-btn")).toHaveTextContent("Create Organization");
  });

  it("shows update button text when initial values provided", () => {
    render(
      <EntityForm
        entityLabel="Organization"
        onSubmit={vi.fn()}
        initialValues={{ name: "Acme", description: "desc" }}
      />,
    );
    expect(screen.getByTestId("entity-submit-btn")).toHaveTextContent("Update Organization");
  });

  it("uses custom submitLabel", () => {
    render(<EntityForm entityLabel="Organization" onSubmit={vi.fn()} submitLabel="Save" />);
    expect(screen.getByTestId("entity-submit-btn")).toHaveTextContent("Save");
  });

  it("shows saving text when submitting", () => {
    render(<EntityForm entityLabel="Organization" onSubmit={vi.fn()} submitting={true} />);
    expect(screen.getByTestId("entity-submit-btn")).toHaveTextContent("Saving...");
  });

  it("pre-fills initial values", () => {
    render(
      <EntityForm
        entityLabel="Organization"
        onSubmit={vi.fn()}
        initialValues={{ name: "Acme", description: "A corp" }}
      />,
    );
    expect(screen.getByTestId("entity-name-input")).toHaveValue("Acme");
    expect(screen.getByTestId("entity-desc-input")).toHaveValue("A corp");
  });

  it("shows validation error when name is empty", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm entityLabel="Organization" onSubmit={onSubmit} />);

    await user.click(screen.getByTestId("entity-submit-btn"));
    expect(screen.getByTestId("entity-name-error")).toHaveTextContent("Name is required");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("calls onSubmit with form values", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm entityLabel="Organization" onSubmit={onSubmit} />);

    await user.type(screen.getByTestId("entity-name-input"), "New Org");
    await user.type(screen.getByTestId("entity-desc-input"), "Description");
    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(onSubmit).toHaveBeenCalledWith({
      name: "New Org",
      description: "Description",
      type: undefined,
      metadata: {},
    });
  });

  it("does not show type selector by default", () => {
    render(<EntityForm entityLabel="Organization" onSubmit={vi.fn()} />);
    expect(screen.queryByTestId("entity-type-select")).not.toBeInTheDocument();
  });

  it("shows type selector when showTypeSelector is true", () => {
    render(<EntityForm entityLabel="Space" onSubmit={vi.fn()} showTypeSelector={true} />);
    expect(screen.getByTestId("entity-type-select")).toBeInTheDocument();
  });

  it("includes type in submit when showTypeSelector is true", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm entityLabel="Space" onSubmit={onSubmit} showTypeSelector={true} />);

    await user.type(screen.getByTestId("entity-name-input"), "Sales");
    await user.selectOptions(screen.getByTestId("entity-type-select"), "crm");
    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ type: "crm" }));
  });

  it("renders cancel button when onCancel provided", () => {
    render(<EntityForm entityLabel="Organization" onSubmit={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByTestId("entity-cancel-btn")).toBeInTheDocument();
  });

  it("does not render cancel button when onCancel not provided", () => {
    render(<EntityForm entityLabel="Organization" onSubmit={vi.fn()} />);
    expect(screen.queryByTestId("entity-cancel-btn")).not.toBeInTheDocument();
  });

  it("calls onCancel when cancel clicked", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(<EntityForm entityLabel="Organization" onSubmit={vi.fn()} onCancel={onCancel} />);

    await user.click(screen.getByTestId("entity-cancel-btn"));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("disables fields when submitting", () => {
    render(
      <EntityForm
        entityLabel="Organization"
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
        submitting={true}
      />,
    );
    expect(screen.getByTestId("entity-name-input")).toBeDisabled();
    expect(screen.getByTestId("entity-desc-input")).toBeDisabled();
    expect(screen.getByTestId("entity-submit-btn")).toBeDisabled();
    expect(screen.getByTestId("entity-cancel-btn")).toBeDisabled();
  });

  it("renders metadata editor", () => {
    render(<EntityForm entityLabel="Organization" onSubmit={vi.fn()} />);
    expect(screen.getByTestId("metadata-editor")).toBeInTheDocument();
  });

  it("trims name and description on submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm entityLabel="Org" onSubmit={onSubmit} />);

    await user.type(screen.getByTestId("entity-name-input"), "  Acme  ");
    await user.type(screen.getByTestId("entity-desc-input"), "  desc  ");
    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ name: "Acme", description: "desc" }),
    );
  });
});
