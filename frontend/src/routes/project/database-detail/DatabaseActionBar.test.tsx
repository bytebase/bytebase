import { act, fireEvent, render, screen } from "@testing-library/react";
import { Pencil } from "lucide-react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { DatabaseActionBar, type DatabaseAction } from "./DatabaseActionBar";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

let availableWidth = 600;
const resizeCallbacks = new Set<() => void>();
beforeEach(() => {
  availableWidth = 600;
  vi.stubGlobal("ResizeObserver", class {
    callback: () => void;
    constructor(callback: () => void) { this.callback = callback; resizeCallbacks.add(callback); }
    observe() {}
    unobserve() {}
    disconnect() { resizeCallbacks.delete(this.callback); }
  });
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (this: HTMLElement) {
    let width = 0;
    if (this.classList.contains("relative")) width = availableWidth;
    else if (this.parentElement?.closest("[inert]")) width = this.textContent === "common.more" ? 60 : 100;
    else if (this.querySelector("a")) width = 140;
    return { width, height: 36, top: 0, left: 0, bottom: 36, right: width, x: 0, y: 0, toJSON() {} };
  });
});
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); resizeCallbacks.clear(); });

function resize(width: number) {
  act(() => { availableWidth = width; for (const callback of resizeCallbacks) callback(); });
}

test("keeps actions visible when they fit and overflows by priority as space shrinks", async () => {
  const sync = vi.fn();
  const exportSchema = vi.fn();
  const actions: DatabaseAction[] = [
    { key: "change", label: "Change Database" },
    { key: "sync", label: "Sync Database", onClick: sync },
    { key: "export", label: "Export Schema", options: [{ key: "single", label: "Single file", onClick: exportSchema }] },
    { key: "transfer", label: "Transfer Project", disabled: true },
  ].map((action) => ({ ...action, icon: Pencil, wrap: (content) => content }));
  render(<DatabaseActionBar actions={actions} primary={<a href="/sql-editor">Open SQL Editor</a>} />);
  expect(screen.queryByRole("button", { name: "common.more" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Transfer Project" })).toBeDisabled();

  expect(screen.getAllByRole("button").map((button) => button.textContent)).toEqual([
    "Transfer Project", "Export Schema", "Sync Database", "Change Database",
  ]);
  resize(450);
  expect(screen.getAllByRole("button").map((button) => button.textContent)).toEqual([
    "common.more", "Sync Database", "Change Database",
  ]);

  resize(350);
  expect(screen.getByRole("button", { name: "Change Database" })).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Sync Database" })).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "common.more" }));
  fireEvent.click(await screen.findByRole("menuitem", { name: "Sync Database" }));
  expect(sync).toHaveBeenCalledOnce();

  fireEvent.click(screen.getByRole("button", { name: "common.more" }));
  const exportItem = await screen.findByRole("menuitem", { name: "Export Schema" });
  fireEvent.click(exportItem);
  fireEvent.click(await screen.findByRole("menuitem", { name: "Single file" }));
  expect(exportSchema).toHaveBeenCalledOnce();

  resize(220);
  expect(screen.queryByRole("button", { name: "Change Database" })).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: "Open SQL Editor" })).toBeInTheDocument();
  resize(600);
  expect(screen.queryByRole("button", { name: "common.more" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Sync Database" })).toBeInTheDocument();
});
