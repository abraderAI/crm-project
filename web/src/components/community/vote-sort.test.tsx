import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { VoteSort } from "./vote-sort";

describe("VoteSort", () => {
  it("renders all sort options", () => {
    render(<VoteSort value="votes" onChange={vi.fn()} />);
    expect(screen.getByTestId("sort-option-votes")).toBeInTheDocument();
    expect(screen.getByTestId("sort-option-newest")).toBeInTheDocument();
    expect(screen.getByTestId("sort-option-oldest")).toBeInTheDocument();
  });

  it("highlights the active sort option", () => {
    render(<VoteSort value="newest" onChange={vi.fn()} />);
    expect(screen.getByTestId("sort-option-newest")).toHaveClass("bg-primary");
    expect(screen.getByTestId("sort-option-votes")).not.toHaveClass("bg-primary");
  });

  it("calls onChange with the selected option", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<VoteSort value="votes" onChange={onChange} />);

    await user.click(screen.getByTestId("sort-option-newest"));
    expect(onChange).toHaveBeenCalledWith("newest");
  });

  it("calls onChange with oldest option", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<VoteSort value="votes" onChange={onChange} />);

    await user.click(screen.getByTestId("sort-option-oldest"));
    expect(onChange).toHaveBeenCalledWith("oldest");
  });

  it("calls onChange with votes option", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<VoteSort value="newest" onChange={onChange} />);

    await user.click(screen.getByTestId("sort-option-votes"));
    expect(onChange).toHaveBeenCalledWith("votes");
  });

  it("renders the sort icon", () => {
    render(<VoteSort value="votes" onChange={vi.fn()} />);
    expect(screen.getByTestId("vote-sort-icon")).toBeInTheDocument();
  });

  it("renders the sort options container", () => {
    render(<VoteSort value="votes" onChange={vi.fn()} />);
    expect(screen.getByTestId("vote-sort-options")).toBeInTheDocument();
  });

  it("displays correct label text", () => {
    render(<VoteSort value="votes" onChange={vi.fn()} />);
    expect(screen.getByTestId("sort-option-votes")).toHaveTextContent("Top voted");
    expect(screen.getByTestId("sort-option-newest")).toHaveTextContent("Newest");
    expect(screen.getByTestId("sort-option-oldest")).toHaveTextContent("Oldest");
  });
});
