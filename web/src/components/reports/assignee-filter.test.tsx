import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { AssigneeFilter } from "./assignee-filter";

const mockMembers = {
  data: [
    {
      id: "m1",
      user_id: "user-alice",
      role: "admin",
      org_id: "org1",
      created_at: "",
      updated_at: "",
    },
    {
      id: "m2",
      user_id: "user-bob",
      role: "viewer",
      org_id: "org1",
      created_at: "",
      updated_at: "",
    },
  ],
  page_info: { has_more: false },
};

let fetchSpy: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(mockMembers), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
});

afterEach(() => {
  fetchSpy.mockRestore();
});

describe("AssigneeFilter", () => {
  it("renders 'All assignees' option", async () => {
    render(<AssigneeFilter orgId="org1" value={null} onChange={vi.fn()} />);
    await waitFor(() => {
      expect(screen.getByTestId("assignee-option-all")).toBeInTheDocument();
    });
    expect(screen.getByTestId("assignee-option-all")).toHaveTextContent("All assignees");
  });

  it("fetches and renders member list", async () => {
    render(<AssigneeFilter orgId="org1" value={null} onChange={vi.fn()} />);
    await waitFor(() => {
      expect(screen.getByTestId("assignee-option-user-alice")).toBeInTheDocument();
    });
    expect(screen.getByTestId("assignee-option-user-bob")).toBeInTheDocument();
  });

  it("calls onChange with user ID on selection", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<AssigneeFilter orgId="org1" value={null} onChange={onChange} />);

    await waitFor(() => {
      expect(screen.getByTestId("assignee-option-user-alice")).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByTestId("assignee-select"), "user-alice");
    expect(onChange).toHaveBeenCalledWith("user-alice");
  });

  it("calls onChange with null when 'All assignees' selected", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<AssigneeFilter orgId="org1" value="user-alice" onChange={onChange} />);

    await waitFor(() => {
      expect(screen.getByTestId("assignee-option-user-alice")).toBeInTheDocument();
    });

    await user.selectOptions(screen.getByTestId("assignee-select"), "");
    expect(onChange).toHaveBeenCalledWith(null);
  });

  it("renders the filter container", async () => {
    render(<AssigneeFilter orgId="org1" value={null} onChange={vi.fn()} />);
    expect(screen.getByTestId("assignee-filter")).toBeInTheDocument();
  });

  it("select is disabled while loading", () => {
    // Before any fetch resolves.
    fetchSpy.mockReturnValue(new Promise(() => {}));
    render(<AssigneeFilter orgId="org1" value={null} onChange={vi.fn()} />);
    expect(screen.getByTestId("assignee-select")).toBeDisabled();
  });

  it("select is enabled after loading", async () => {
    render(<AssigneeFilter orgId="org1" value={null} onChange={vi.fn()} />);
    await waitFor(() => {
      expect(screen.getByTestId("assignee-select")).not.toBeDisabled();
    });
  });

  it("handles fetch error gracefully", async () => {
    fetchSpy.mockRejectedValueOnce(new Error("Network error"));
    render(<AssigneeFilter orgId="org1" value={null} onChange={vi.fn()} />);
    await waitFor(() => {
      expect(screen.getByTestId("assignee-select")).not.toBeDisabled();
    });
    // Should still show "All assignees" even on error.
    expect(screen.getByTestId("assignee-option-all")).toBeInTheDocument();
  });

  it("reflects the selected value", async () => {
    render(<AssigneeFilter orgId="org1" value="user-bob" onChange={vi.fn()} />);
    await waitFor(() => {
      expect(screen.getByTestId("assignee-option-user-bob")).toBeInTheDocument();
    });
    const select = screen.getByTestId("assignee-select") as HTMLSelectElement;
    expect(select.value).toBe("user-bob");
  });
});
