import { create } from "@bufbuild/protobuf";
import { cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { Plan_ChangeDatabaseConfigSchema, Plan_SpecSchema, PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { type Sheet, SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { current, OUTDATED } from "./placement/place";
import { StatementAnchorContext } from "./StatementAnchorContext";
import { buildWholeLineAnchor } from "./threadModel";

const mocks = vi.hoisted(() => ({
  sheets: {} as Record<string, Sheet>,
  fetchSheet: vi.fn(),
  colorize: vi.fn(async (text: string) => text.split("\n").map((line) => `<span>${line}</span>`).join("<br/>")),
}));
vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({t: (key: string, options?: { count?: number; range: string }) =>
    options ? `${(options.count ?? 1) > 1 ? "Lines" : "Line"} ${options.range}` : key}),
}));
vi.mock("@/components/monaco/core", () => ({ colorizeStatement: (text: string) => mocks.colorize(text) }));
vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) => selector({ sheetsByName: mocks.sheets }),
    {
      getState: () => ({
        getSheetByName: (name: string) => mocks.sheets[name],
        getOrFetchSheetByName: mocks.fetchSheet,
      }),
    }
  ),
}));

const savedHash = "c".repeat(64);
const currentHash = "d".repeat(64);
const savedName = `projects/p/sheets/${savedHash}`;
const currentName = `projects/p/sheets/${currentHash}`;
const original = "/* original revision\ncomment line\ncomment line 2\n*/\nSELECT 1;\nSELECT 2;";
const project = create(ProjectSchema, { name: "projects/p" });
const plan = create(PlanSchema, { specs: [create(Plan_SpecSchema, {
  id: "spec",
  config: { case: "changeDatabaseConfig", value: create(Plan_ChangeDatabaseConfigSchema, { sheet: currentName }) },
})] });
const anchor = (startLine: number, endLine: number) => buildWholeLineAnchor({
  spec: "spec", sheetSha256: savedHash, startLine, endLine,
});
const props = {
  plan,
  project,
  onViewInStatement: vi.fn(),
  renderPlanChangeReference: () => <span>Change 1</span>,
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.sheets = {
    [savedName]: create(SheetSchema, { name: savedName, content: new TextEncoder().encode(original) }),
    [currentName]: create(SheetSchema, { name: currentName, content: new TextEncoder().encode("CURRENT REVISION") }),
  };
});
afterEach(cleanup);

describe("StatementAnchorContext", () => {
  test.each([
    { start: 1, end: 1, displayed: [1], selected: [1], label: "Line 1" },
    { start: 2, end: 2, displayed: [1, 2], selected: [2], label: "Line 2" },
    { start: 4, end: 5, displayed: [2, 3, 4, 5], selected: [4, 5], label: "Lines 4–5" },
  ])("adds at most two preceding lines to $label without expanding the anchor", async ({start, end, displayed, selected, label}) => {
    const { container, getByText } = render(<StatementAnchorContext {...props} anchor={anchor(start, end)} placement={OUTDATED} />);
    const numbers = (selector: string) => Array.from(container.querySelectorAll(selector)).map((el) => Number(el.getAttribute("data-line-number")));
    expect(numbers("[data-line-number]")).toEqual(displayed);
    expect(numbers('[data-selected="true"]')).toEqual(selected);
    expect(getByText(label)).toBeTruthy();
    // Colorized from the recorded revision, never the current one.
    await waitFor(() => expect(container.querySelector("code span")).not.toBeNull());
    expect(container.textContent).not.toContain("CURRENT REVISION");
    expect(container.textContent).not.toContain("SELECT 2;");
  });

  test("tokenizes a recorded revision once, from the document start, however many cards excerpt it", async () => {
    const hash = "e".repeat(64);
    const name = `projects/p/sheets/${hash}`;
    mocks.sheets[name] = create(SheetSchema, { name, content: new TextEncoder().encode(original) });
    const shared = buildWholeLineAnchor({ spec: "spec", sheetSha256: hash, startLine: 5, endLine: 5 });
    const { container } = render(
      <>
        <StatementAnchorContext {...props} anchor={shared} placement={OUTDATED} />
        <StatementAnchorContext {...props} anchor={shared} placement={OUTDATED} />
      </>
    );
    await waitFor(() => expect(container.querySelectorAll("code span").length).toBeGreaterThan(0));
    expect(mocks.colorize).toHaveBeenCalledTimes(1);
    // Only through the excerpt's last line, from the document start.
    expect(mocks.colorize).toHaveBeenCalledWith(
      `${original.split("\n").slice(0, 5).join("\n")}\n`
    );
  });

  test("mapped placements still render context from the recorded revision", () => {
    const { container, getByText } = render(<StatementAnchorContext {...props} anchor={anchor(4, 5)} placement={current(10, 11)} />);
    expect(getByText("Lines 4–5")).toBeTruthy();
    expect(container.textContent).toContain("comment line");
    expect(container.textContent).not.toContain("CURRENT REVISION");
  });

  test.each([
    { truncated: false, state: "CURRENT", action: true },
    { truncated: true, state: "UNAVAILABLE", action: false },
  ])("a hash-matched anchor on a truncated sheet ($truncated) is $state", ({ truncated, state, action }) => {
    const content = new TextEncoder().encode(original);
    mocks.sheets[currentName] = create(SheetSchema, {
      name: currentName,
      content,
      contentSize: BigInt(content.byteLength + (truncated ? 1 : 0)),
    });
    const onCurrent = buildWholeLineAnchor({ spec: "spec", sheetSha256: currentHash, startLine: 5, endLine: 5 });
    const { container, queryByText } = render(<StatementAnchorContext {...props} anchor={onCurrent} placement={undefined} />);
    expect(container.querySelector("[data-anchor-state]")?.getAttribute("data-anchor-state")).toBe(state);
    expect(queryByText("plan.review.thread.anchor.view-in-statement") !== null).toBe(action);
  });

  test("fetches a missing recorded revision instead of using the current SQL", async () => {
    delete mocks.sheets[savedName];
    const { container } = render(<StatementAnchorContext {...props} anchor={anchor(4, 5)} placement={OUTDATED} />);
    await waitFor(() => expect(mocks.fetchSheet).toHaveBeenCalledWith(savedName));
    expect(container.querySelector("[data-line-number]")).toBeNull();
    expect(container.textContent).not.toContain("CURRENT REVISION");
  });
});
