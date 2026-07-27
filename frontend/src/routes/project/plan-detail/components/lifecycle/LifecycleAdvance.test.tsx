import { fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { describe, expect, test, vi } from "vitest";
import type { AdvanceBlocker } from "./advanceState";
import {
  LifecycleAdvanceButton,
  type LifecycleAdvanceProps,
} from "./LifecycleAdvance";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    appearance: _appearance,
    children,
    size: _size,
    ...props
  }: ButtonHTMLAttributes<HTMLButtonElement> & {
    appearance?: string;
    size?: string;
  }) => <button {...props}>{children}</button>,
}));

vi.mock("@/components/ui/popover", () => ({
  Popover: ({
    children,
    onOpenChange,
    open,
  }: {
    children: ReactNode;
    onOpenChange?: (open: boolean) => void;
    open?: boolean;
  }) =>
    open ? (
      <div data-testid="popover">
        <button onClick={() => onOpenChange?.(false)} type="button">
          outside-press
        </button>
        {children}
      </div>
    ) : null,
  PopoverContent: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

const fix = (id: string, message = id): AdvanceBlocker => ({
  id,
  kind: "fix",
  message,
});

const DECISION = {
  body: "issue.checks-warning-hint",
  headline: "plan.lifecycle.gate-checks-failed",
  verb: "plan.submit-review-anyway",
};

function Harness(props: Partial<LifecycleAdvanceProps> = {}) {
  return (
    <LifecycleAdvanceButton
      blockers={[]}
      heading="plan.cannot-create"
      onAdvance={vi.fn()}
      verb="Create"
      {...props}
    />
  );
}

const pressPrimary = () =>
  fireEvent.click(screen.getByRole("button", { name: "Create" }));

describe("LifecycleAdvance tier 0", () => {
  test("advances on press and opens nothing", () => {
    const onAdvance = vi.fn();
    render(<Harness onAdvance={onAdvance} />);

    pressPrimary();

    expect(onAdvance).toHaveBeenCalledOnce();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.queryByTestId("popover")).not.toBeInTheDocument();
  });

});

describe("LifecycleAdvance tier 1", () => {
  test("shows nothing before the first press", () => {
    render(<Harness blockers={[fix("title")]} />);

    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  test("keeps the action enabled and states every blocker on press", () => {
    const onAdvance = vi.fn();
    render(
      <Harness
        blockers={[fix("title", "plan.title-required"), fix("statement")]}
        onAdvance={onAdvance}
      />
    );

    expect(screen.getByRole("button", { name: "Create" })).toBeEnabled();
    pressPrimary();

    expect(onAdvance).not.toHaveBeenCalled();
    const notice = screen.getByRole("alert");
    expect(notice).toHaveTextContent("plan.cannot-create");
    expect(notice).toHaveTextContent("plan.title-required");
    expect(notice).toHaveTextContent("statement");
  });

  test("runs the blocked-press nudge", () => {
    const onBlocked = vi.fn();
    render(<Harness blockers={[fix("title")]} onBlocked={onBlocked} />);

    pressPrimary();

    expect(onBlocked).toHaveBeenCalledOnce();
  });

  test("empties itself when the last blocker resolves, with no second press", () => {
    const { rerender } = render(<Harness blockers={[fix("title")]} />);
    pressPrimary();
    expect(screen.getByRole("alert")).toBeInTheDocument();

    rerender(<Harness blockers={[]} />);

    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  test("shortens as blockers resolve one at a time", () => {
    const { rerender } = render(
      <Harness blockers={[fix("title"), fix("statement")]} />
    );
    pressPrimary();

    rerender(<Harness blockers={[fix("statement")]} />);

    const notice = screen.getByRole("alert");
    expect(notice).toHaveTextContent("statement");
    expect(notice).not.toHaveTextContent("title");
  });

  test("marks a self-resolving blocker so it does not read as a chore", () => {
    render(
      <Harness
        blockers={[
          { id: "checks-running", kind: "wait", message: "checks are running" },
        ]}
      />
    );
    pressPrimary();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "plan.blocker.clears-on-its-own"
    );
  });

  test("does not mark a blocker the reader has to resolve", () => {
    render(<Harness blockers={[fix("title")]} />);
    pressPrimary();

    expect(screen.getByRole("alert")).not.toHaveTextContent(
      "plan.blocker.clears-on-its-own"
    );
  });

  test("outranks a decision", () => {
    const onAdvance = vi.fn();
    render(
      <Harness
        blockers={[fix("statement")]}
        decision={DECISION}
        onAdvance={onAdvance}
      />
    );

    pressPrimary();

    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "plan.submit-review-anyway" })
    ).not.toBeInTheDocument();
    expect(onAdvance).not.toHaveBeenCalled();
  });

  test("closes on outside press", () => {
    render(<Harness blockers={[fix("title")]} />);
    pressPrimary();
    expect(screen.getByRole("alert")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "outside-press" }));

    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

});

describe("LifecycleAdvance tier 2", () => {
  test("asks before advancing", () => {
    const onAdvance = vi.fn();
    render(<Harness decision={DECISION} onAdvance={onAdvance} />);

    pressPrimary();

    expect(onAdvance).not.toHaveBeenCalled();
    expect(screen.getByTestId("popover")).toHaveTextContent(
      "plan.lifecycle.gate-checks-failed"
    );
    expect(screen.getByTestId("popover")).toHaveTextContent(
      "issue.checks-warning-hint"
    );
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  test("names the override after what it does", () => {
    render(<Harness decision={DECISION} />);
    pressPrimary();

    expect(
      screen.getByRole("button", { name: "plan.submit-review-anyway" })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "common.confirm" })
    ).not.toBeInTheDocument();
  });

  test("advances once when confirmed and closes the confirmation", () => {
    const onAdvance = vi.fn();
    render(<Harness decision={DECISION} onAdvance={onAdvance} />);
    pressPrimary();

    fireEvent.click(
      screen.getByRole("button", { name: "plan.submit-review-anyway" })
    );

    expect(onAdvance).toHaveBeenCalledOnce();
    expect(screen.queryByTestId("popover")).not.toBeInTheDocument();
  });

  test("does not advance when cancelled", () => {
    const onAdvance = vi.fn();
    render(<Harness decision={DECISION} onAdvance={onAdvance} />);
    pressPrimary();

    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));

    expect(onAdvance).not.toHaveBeenCalled();
    expect(screen.queryByTestId("popover")).not.toBeInTheDocument();
  });

  test("asks no acknowledgement checkbox and offers no metadata fields", () => {
    render(<Harness decision={DECISION} />);
    pressPrimary();

    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  test("switches an open decision to blockers when checks restart", () => {
    const { rerender } = render(<Harness decision={DECISION} />);
    pressPrimary();

    rerender(
      <Harness
        blockers={[
          { id: "checks-running", kind: "wait", message: "checks are running" },
        ]}
      />
    );

    expect(screen.getByRole("alert")).toHaveTextContent("checks are running");
    expect(
      screen.queryByRole("button", { name: "plan.submit-review-anyway" })
    ).not.toBeInTheDocument();
  });

  test("lets new blockers outrank a still-valid open decision", () => {
    const { rerender } = render(<Harness decision={DECISION} />);
    pressPrimary();

    rerender(
      <Harness blockers={[fix("statement")]} decision={DECISION} />
    );

    expect(screen.getByRole("alert")).toHaveTextContent("statement");
    expect(
      screen.queryByRole("button", { name: "plan.submit-review-anyway" })
    ).not.toBeInTheDocument();

    rerender(<Harness decision={DECISION} />);

    expect(screen.queryByTestId("popover")).not.toBeInTheDocument();
  });

  test("closes an open decision when it is no longer required", () => {
    const { rerender } = render(<Harness decision={DECISION} />);
    pressPrimary();

    rerender(<Harness />);

    expect(screen.queryByTestId("popover")).not.toBeInTheDocument();

    rerender(<Harness decision={DECISION} />);

    expect(screen.queryByTestId("popover")).not.toBeInTheDocument();
  });
});

describe("LifecycleAdvance in flight", () => {
  test("disables the action while a request is running", () => {
    const onAdvance = vi.fn();
    render(<Harness busy onAdvance={onAdvance} />);

    const button = screen.getByRole("button", { name: "Create" });
    expect(button).toBeDisabled();
    fireEvent.click(button);
    expect(onAdvance).not.toHaveBeenCalled();
  });

});
