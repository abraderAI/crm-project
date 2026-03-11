import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { EntitySettings } from "./entity-settings";

const defaultValues = {
  id: "org-1",
  slug: "acme-corp",
  name: "Acme Corp",
  description: "A test org",
  metadata: '{"tier":"pro"}',
};

describe("EntitySettings", () => {
  it("renders settings container", () => {
    render(<EntitySettings entityType="org" currentValues={defaultValues} onSave={vi.fn()} />);
    expect(screen.getByTestId("entity-settings")).toBeInTheDocument();
  });

  it("renders header with slug", () => {
    render(<EntitySettings entityType="org" currentValues={defaultValues} onSave={vi.fn()} />);
    expect(screen.getByText(/acme-corp — Settings/)).toBeInTheDocument();
  });

  it("renders the edit form", () => {
    render(<EntitySettings entityType="org" currentValues={defaultValues} onSave={vi.fn()} />);
    expect(screen.getByTestId("entity-form")).toBeInTheDocument();
    expect(screen.getByTestId("entity-name-input")).toHaveValue("Acme Corp");
  });

  it("renders danger zone when onDelete provided", () => {
    render(
      <EntitySettings
        entityType="org"
        currentValues={defaultValues}
        onSave={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByTestId("danger-zone")).toBeInTheDocument();
    expect(screen.getByTestId("entity-delete-btn")).toBeInTheDocument();
  });

  it("hides danger zone when onDelete not provided", () => {
    render(<EntitySettings entityType="org" currentValues={defaultValues} onSave={vi.fn()} />);
    expect(screen.queryByTestId("danger-zone")).not.toBeInTheDocument();
  });

  it("calls onDelete when delete button clicked", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    render(
      <EntitySettings
        entityType="org"
        currentValues={defaultValues}
        onSave={vi.fn()}
        onDelete={onDelete}
      />,
    );

    await user.click(screen.getByTestId("entity-delete-btn"));
    expect(onDelete).toHaveBeenCalledOnce();
  });

  it("disables delete button when loading", () => {
    render(
      <EntitySettings
        entityType="org"
        currentValues={defaultValues}
        onSave={vi.fn()}
        onDelete={vi.fn()}
        loading={true}
      />,
    );
    expect(screen.getByTestId("entity-delete-btn")).toBeDisabled();
  });

  it("renders with space entity type", () => {
    render(
      <EntitySettings
        entityType="space"
        currentValues={{ ...defaultValues, type: "crm" }}
        onSave={vi.fn()}
      />,
    );
    expect(screen.getByTestId("entity-type-select")).toBeInTheDocument();
  });

  it("calls onSave when form submitted", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(<EntitySettings entityType="org" currentValues={defaultValues} onSave={onSave} />);

    await user.click(screen.getByTestId("entity-submit-btn"));
    expect(onSave).toHaveBeenCalled();
  });

  it("shows delete text with entity type", () => {
    render(
      <EntitySettings
        entityType="board"
        currentValues={defaultValues}
        onSave={vi.fn()}
        onDelete={vi.fn()}
      />,
    );
    expect(screen.getByText(/Delete board/)).toBeInTheDocument();
    expect(screen.getByText(/Permanently delete this board/)).toBeInTheDocument();
  });
});
