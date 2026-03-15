import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { DateRangePicker } from "./date-range-picker";

const from = new Date(2026, 2, 1); // Mar 1, 2026
const to = new Date(2026, 2, 31); // Mar 31, 2026

describe("DateRangePicker", () => {
  it("renders trigger button with formatted date range", () => {
    render(<DateRangePicker from={from} to={to} onChange={vi.fn()} />);
    const trigger = screen.getByTestId("date-range-trigger");
    expect(trigger).toBeInTheDocument();
    expect(trigger).toHaveTextContent("Mar 1, 2026");
    expect(trigger).toHaveTextContent("Mar 31, 2026");
  });

  it("does not show popover by default", () => {
    render(<DateRangePicker from={from} to={to} onChange={vi.fn()} />);
    expect(screen.queryByTestId("date-range-popover")).not.toBeInTheDocument();
  });

  it("opens popover on click", async () => {
    const user = userEvent.setup();
    render(<DateRangePicker from={from} to={to} onChange={vi.fn()} />);
    await user.click(screen.getByTestId("date-range-trigger"));
    expect(screen.getByTestId("date-range-popover")).toBeInTheDocument();
  });

  it("closes popover on Apply click", async () => {
    const user = userEvent.setup();
    render(<DateRangePicker from={from} to={to} onChange={vi.fn()} />);
    await user.click(screen.getByTestId("date-range-trigger"));
    expect(screen.getByTestId("date-range-popover")).toBeInTheDocument();
    await user.click(screen.getByTestId("date-range-close"));
    expect(screen.queryByTestId("date-range-popover")).not.toBeInTheDocument();
  });

  it("calls onChange with new from date", () => {
    const onChange = vi.fn();
    render(<DateRangePicker from={from} to={to} onChange={onChange} />);

    // Open popover.
    fireEvent.click(screen.getByTestId("date-range-trigger"));

    // Change from date.
    fireEvent.change(screen.getByTestId("date-range-from"), {
      target: { value: "2026-02-15" },
    });

    expect(onChange).toHaveBeenCalledTimes(1);
    const call = onChange.mock.calls[0] as [{ from: Date; to: Date }];
    expect(call[0].from.getFullYear()).toBe(2026);
    expect(call[0].from.getMonth()).toBe(1); // Feb
    expect(call[0].from.getDate()).toBe(15);
    expect(call[0].to).toBe(to);
  });

  it("calls onChange with new to date", () => {
    const onChange = vi.fn();
    render(<DateRangePicker from={from} to={to} onChange={onChange} />);

    fireEvent.click(screen.getByTestId("date-range-trigger"));
    fireEvent.change(screen.getByTestId("date-range-to"), {
      target: { value: "2026-04-15" },
    });

    expect(onChange).toHaveBeenCalledTimes(1);
    const call = onChange.mock.calls[0] as [{ from: Date; to: Date }];
    expect(call[0].from).toBe(from);
    expect(call[0].to.getFullYear()).toBe(2026);
    expect(call[0].to.getMonth()).toBe(3); // Apr
  });

  it("does not call onChange for invalid from date", () => {
    const onChange = vi.fn();
    render(<DateRangePicker from={from} to={to} onChange={onChange} />);

    fireEvent.click(screen.getByTestId("date-range-trigger"));
    fireEvent.change(screen.getByTestId("date-range-from"), {
      target: { value: "" },
    });

    expect(onChange).not.toHaveBeenCalled();
  });

  it("does not call onChange for invalid to date", () => {
    const onChange = vi.fn();
    render(<DateRangePicker from={from} to={to} onChange={onChange} />);

    fireEvent.click(screen.getByTestId("date-range-trigger"));
    fireEvent.change(screen.getByTestId("date-range-to"), {
      target: { value: "" },
    });

    expect(onChange).not.toHaveBeenCalled();
  });

  it("renders the picker container", () => {
    render(<DateRangePicker from={from} to={to} onChange={vi.fn()} />);
    expect(screen.getByTestId("date-range-picker")).toBeInTheDocument();
  });

  it("renders From and To labels in popover", async () => {
    const user = userEvent.setup();
    render(<DateRangePicker from={from} to={to} onChange={vi.fn()} />);
    await user.click(screen.getByTestId("date-range-trigger"));
    expect(screen.getByText("From")).toBeInTheDocument();
    expect(screen.getByText("To")).toBeInTheDocument();
  });

  it("shows correct input values in popover", async () => {
    const user = userEvent.setup();
    render(<DateRangePicker from={from} to={to} onChange={vi.fn()} />);
    await user.click(screen.getByTestId("date-range-trigger"));

    const fromInput = screen.getByTestId("date-range-from") as HTMLInputElement;
    expect(fromInput.value).toBe("2026-03-01");
    const toInput = screen.getByTestId("date-range-to") as HTMLInputElement;
    expect(toInput.value).toBe("2026-03-31");
  });
});
