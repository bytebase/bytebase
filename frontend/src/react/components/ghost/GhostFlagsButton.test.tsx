import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, test, vi } from "vitest";
import { GHOST_PARAMETERS, withFlag } from "./constants";
import { GhostFlagsButton } from "./GhostFlagsButton";
import { GhostFlagsForm } from "./GhostFlagsForm";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

// Render the sheet content inline when open — jsdom has no portal/positioning,
// and the save/cancel batching is what we want to assert.
vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  SheetContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SheetHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
  SheetBody: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

const paramFor = (key: string) => {
  const param = GHOST_PARAMETERS.find((p) => p.key === key);
  if (!param) throw new Error(`unknown test param: ${key}`);
  return param;
};

const toggleFlag = (root: ParentNode, flag: string) => {
  const toggle = root
    .querySelector(`[data-flag="${flag}"]`)
    ?.querySelector('[role="switch"]');
  if (!toggle) throw new Error(`no switch for ${flag}`);
  fireEvent.click(toggle);
};

const clickButton = (name: RegExp | string) =>
  fireEvent.click(screen.getByRole("button", { name }));

describe("withFlag", () => {
  test("sets a non-default override", () => {
    expect(withFlag({}, paramFor("chunk-size"), "500")).toEqual({
      "chunk-size": "500",
    });
  });

  test("removes a key when set back to its default", () => {
    expect(
      withFlag({ "chunk-size": "500" }, paramFor("chunk-size"), "1000")
    ).toEqual({});
  });

  test("writes booleans flipped away from their default", () => {
    expect(withFlag({}, paramFor("allow-on-master"), false)).toEqual({
      "allow-on-master": "false",
    });
  });

  test("removes a boolean restored to its default", () => {
    expect(
      withFlag(
        { "allow-on-master": "false" },
        paramFor("allow-on-master"),
        true
      )
    ).toEqual({});
  });

  test("clearing a string value removes the key", () => {
    expect(
      withFlag({ "max-load": "Threads_running=25" }, paramFor("max-load"), "")
    ).toEqual({});
  });

  test("preserves other flags", () => {
    expect(
      withFlag({ "max-load": "x" }, paramFor("chunk-size"), "500")
    ).toEqual({ "max-load": "x", "chunk-size": "500" });
  });
});

describe("GhostFlagsForm", () => {
  test("renders a control for every supported flag", () => {
    const { container } = render(
      <GhostFlagsForm value={{}} onChange={vi.fn()} />
    );
    expect(container.querySelectorAll("[data-flag]")).toHaveLength(
      GHOST_PARAMETERS.length
    );
  });

  test("toggling a boolean flag emits only that override", () => {
    const onChange = vi.fn();
    const { container } = render(
      <GhostFlagsForm value={{}} onChange={onChange} />
    );
    toggleFlag(container, "allow-on-master");
    expect(onChange).toHaveBeenCalledWith({ "allow-on-master": "false" });
  });
});

describe("GhostFlagsButton", () => {
  const TRIGGER = /plan\.ghost\.configure/;

  test("disables the trigger when disabled", () => {
    render(<GhostFlagsButton value={{}} onChange={vi.fn()} disabled />);
    expect(screen.getByRole("button", { name: TRIGGER })).toBeDisabled();
  });

  test("shows the persisted override count", () => {
    render(
      <GhostFlagsButton value={{ "chunk-size": "500" }} onChange={vi.fn()} />
    );
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  test("buffers edits and persists once on save", () => {
    const onChange = vi.fn();
    const { container } = render(
      <GhostFlagsButton value={{}} onChange={onChange} />
    );
    clickButton(TRIGGER);
    toggleFlag(container, "allow-on-master");
    // No persist until the user saves.
    expect(onChange).not.toHaveBeenCalled();
    clickButton("common.save");
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith({ "allow-on-master": "false" });
  });

  test("cancel discards edits without persisting", () => {
    const onChange = vi.fn();
    const { container } = render(
      <GhostFlagsButton value={{}} onChange={onChange} />
    );
    clickButton(TRIGGER);
    toggleFlag(container, "allow-on-master");
    clickButton("common.cancel");
    expect(onChange).not.toHaveBeenCalled();
  });

  test("save is disabled until the draft changes", () => {
    render(<GhostFlagsButton value={{}} onChange={vi.fn()} />);
    clickButton(TRIGGER);
    expect(screen.getByRole("button", { name: "common.save" })).toBeDisabled();
  });

  test("reset clears the draft, then save persists an empty map", () => {
    const onChange = vi.fn();
    render(
      <GhostFlagsButton value={{ "chunk-size": "500" }} onChange={onChange} />
    );
    clickButton(TRIGGER);
    clickButton("plan.ghost.reset-to-defaults");
    clickButton("common.save");
    expect(onChange).toHaveBeenCalledWith({});
  });
});
