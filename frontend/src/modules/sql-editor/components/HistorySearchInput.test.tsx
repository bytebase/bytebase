import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

import { HistorySearchInput } from "./HistorySearchInput";

const render = (el: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  act(() => createRoot(container).render(el));
  return container;
};

afterEach(() => {
  document.body.innerHTML = "";
});

describe("HistorySearchInput", () => {
  test("renders a textarea with the placeholder", () => {
    const c = render(
      <HistorySearchInput value="" onChange={() => {}} placeholder="search…" />
    );
    const ta = c.querySelector("textarea") as HTMLTextAreaElement;
    expect(ta).not.toBeNull();
    expect(ta.placeholder).toBe("search…");
  });

  test("typing calls onChange with the new value", () => {
    const onChange = vi.fn();
    const c = render(<HistorySearchInput value="" onChange={onChange} />);
    const ta = c.querySelector("textarea") as HTMLTextAreaElement;
    act(() => {
      Object.defineProperty(ta, "value", { writable: true, value: "select" });
      ta.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(onChange).toHaveBeenCalledWith("select");
  });

  test("no clear button when empty", () => {
    const c = render(<HistorySearchInput value="" onChange={() => {}} />);
    expect(c.querySelector("[data-clear-search]")).toBeNull();
  });

  test("clear button appears with content and clears on click", () => {
    const onChange = vi.fn();
    const c = render(
      <HistorySearchInput value="select 1" onChange={onChange} />
    );
    const clear = c.querySelector("[data-clear-search]") as HTMLButtonElement;
    expect(clear).not.toBeNull();
    act(() => clear.click());
    expect(onChange).toHaveBeenCalledWith("");
  });

  test("Enter does not insert a newline (preventDefault)", () => {
    const onChange = vi.fn();
    const c = render(<HistorySearchInput value="" onChange={onChange} />);
    const ta = c.querySelector("textarea") as HTMLTextAreaElement;
    const evt = new KeyboardEvent("keydown", {
      key: "Enter",
      bubbles: true,
      cancelable: true,
    });
    act(() => {
      ta.dispatchEvent(evt);
    });
    expect(evt.defaultPrevented).toBe(true);
  });
});
