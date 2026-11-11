import { create } from "@bufbuild/protobuf";
import { cleanup, render, waitFor } from "@testing-library/react";
import type { ComponentProps, ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { Plan_ChangeDatabaseConfigSchema, Plan_SpecSchema, PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { type Sheet, SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { current, OUTDATED, UNAVAILABLE, UNAVAILABLE_SIZE_LIMIT } from "./placement/place";
import { StatementAnchorContext } from "./StatementAnchorContext";
import type { LineRange } from "./threadModel";
import { buildWholeLineAnchor } from "./threadModel";

const mocks = vi.hoisted(() => ({
  sheets: {} as Record<string, Sheet>,
  fetchSheet: vi.fn(),
  colorize: vi.fn(async (text: string) => text.split("\n").map((line) => `<span>${line}</span>`).join("<br/>")),
}));
vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: { count?: number; range?: string; title?: string }) => {
      if (key === "plan.review.thread.anchor.line") {
        return `${(options?.count ?? 1) > 1 ? "Lines" : "Line"} ${options?.range}`;
      }
      if (key === "plan.review.thread.fold-label") {
        return `${options?.title}, ${(options?.count ?? 1) > 1 ? "lines" : "line"} ${options?.range}`;
      }
      return key;
    },
  }),
}));
vi.mock("@/components/monaco/core", () => ({ colorizeStatement: (text: string) => mocks.colorize(text) }));
vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children, content }: { children: ReactNode; content: string }) => (
    <span data-tooltip={content}>{children}</span>
  ),
}));
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
  config: { case: "changeDatabaseConfig", value: create(Plan_ChangeDatabaseConfigSchema, {
    sheet: currentName,
    targets: ["instances/prod/databases/alpha_prod"],
  }) },
})] });
const anchor = (startLine: number, endLine: number) => buildWholeLineAnchor({
  spec: "spec", sheetSha256: savedHash, startLine, endLine,
});
const props = {
  plan,
  project,
  onViewInStatement: vi.fn(),
  changeReferenceResources: { databasesByName: {}, databaseGroupsByName: {}, environmentList: [] },
};

// Prints what the context hands the card, so a test reads the bar's facts and
// sees the excerpt apart from them.
const renderAnchor = (
  overrides: Partial<ComponentProps<typeof StatementAnchorContext>> &
    Pick<ComponentProps<typeof StatementAnchorContext>, "anchor" | "placement">
) => {
  const ui = (o: typeof overrides) => (
    <StatementAnchorContext {...props} {...o}>
      {({ bar, excerpt }) => (
        <div>
          <div
            data-testid="bar"
            data-anchor-state={bar.anchorState}
            data-fold-label={bar.foldLabel}
            data-lines={bar.lines ? rangeText(bar.lines) : undefined}
            data-link={bar.onViewInStatement ? "yes" : undefined}
            data-tooltip-text={bar.stateTooltip}
          >
            {bar.title}
          </div>
          <div data-testid="body">{excerpt}</div>
        </div>
      )}
    </StatementAnchorContext>
  );
  const result = render(ui(overrides));
  const bar = () => result.container.querySelector("[data-testid='bar']") as HTMLElement;
  return {
    ...result,
    bar,
    state: () => bar().getAttribute("data-anchor-state"),
    body: () => result.container.querySelector("[data-testid='body']") as HTMLElement,
    rerenderAnchor: (o: typeof overrides) => result.rerender(ui(o)),
  };
};
const rangeText = (range: LineRange) => `${range.startLine}-${range.endLine}`;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.sheets = {
    [savedName]: create(SheetSchema, { name: savedName, content: new TextEncoder().encode(original) }),
    [currentName]: create(SheetSchema, { name: currentName, content: new TextEncoder().encode("CURRENT REVISION") }),
  };
});
afterEach(cleanup);

describe("StatementAnchorContext", () => {
  test("titles a thread that is not on lines for the plan, with no lines, state or excerpt", () => {
    const { bar, body } = renderAnchor({ anchor: undefined, placement: undefined });
    expect(bar().textContent).toBe("plan.review.thread.comment-on-plan");
    expect(bar().getAttribute("data-fold-label")).toBe("plan.review.thread.comment-on-plan");
    expect(bar().getAttribute("data-lines")).toBeNull();
    expect(bar().getAttribute("data-anchor-state")).toBeNull();
    expect(bar().getAttribute("data-link")).toBeNull();
    expect(body().textContent).toBe("");
  });

  test("titles a thread on a deleted change as such and names its fold control with its lines", () => {
    const deleted = buildWholeLineAnchor({ spec: "gone", sheetSha256: savedHash, startLine: 4, endLine: 5 });
    const { bar } = renderAnchor({ anchor: deleted, placement: undefined });
    expect(bar().textContent).toBe("plan.review.thread.anchor.unavailable-change");
    expect(bar().getAttribute("data-fold-label")).toBe("plan.review.thread.anchor.unavailable-change, lines 4–5");
    expect(bar().getAttribute("data-lines")).toBe("4-5");
  });

  test("titles a thread on a change by the change's label, and two threads on it get different fold names", () => {
    const first = renderAnchor({ anchor: anchor(4, 5), placement: OUTDATED });
    expect(first.bar().textContent).toContain("alpha_prod");
    // The visible label (not its hidden measuring copy) draws icon and index
    // in the bar's text color.
    const visible = Array.from(first.bar().querySelectorAll("[data-plan-change-reference] *")).filter(
      (node) => !node.closest("[aria-hidden='true']")
    );
    expect(visible.length).toBeGreaterThan(0);
    const muted = visible.filter((node) =>
      /\btext-control-(light|placeholder)\b/.test(node.getAttribute("class") ?? "")
    );
    expect(muted).toEqual([]);
    const firstName = first.bar().getAttribute("data-fold-label");
    expect(firstName).toBe("alpha_prod, lines 4–5");
    cleanup();
    const second = renderAnchor({ anchor: anchor(2, 2), placement: OUTDATED });
    expect(second.bar().getAttribute("data-fold-label")).toBe("alpha_prod, line 2");
  });

  test("hands the excerpt to the body and keeps it out of the bar", () => {
    const { bar, body } = renderAnchor({ anchor: anchor(4, 5), placement: OUTDATED });
    expect(body().querySelector("[data-line-number]")).not.toBeNull();
    expect(bar().querySelector("[data-line-number]")).toBeNull();
  });

  test.each([
    { start: 1, end: 1, displayed: [1], selected: [1], lines: "1-1" },
    { start: 2, end: 2, displayed: [1, 2], selected: [2], lines: "2-2" },
    { start: 4, end: 5, displayed: [2, 3, 4, 5], selected: [4, 5], lines: "4-5" },
  ])("adds at most two preceding lines to lines $lines without expanding the anchor", async ({start, end, displayed, selected, lines}) => {
    const { container, bar } = renderAnchor({ anchor: anchor(start, end), placement: OUTDATED });
    const numbers = (selector: string) => Array.from(container.querySelectorAll(selector)).map((el) => Number(el.getAttribute("data-line-number")));
    expect(numbers("[data-line-number]")).toEqual(displayed);
    expect(numbers('[data-selected="true"]')).toEqual(selected);
    expect(bar().getAttribute("data-lines")).toBe(lines);
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
    const card = () => (
      <StatementAnchorContext {...props} anchor={shared} placement={OUTDATED}>
        {({ excerpt }) => excerpt}
      </StatementAnchorContext>
    );
    const { container } = render(<>{card()}{card()}</>);
    await waitFor(() => expect(container.querySelectorAll("code span").length).toBeGreaterThan(0));
    expect(mocks.colorize).toHaveBeenCalledTimes(1);
    // Only through the excerpt's last line, from the document start.
    expect(mocks.colorize).toHaveBeenCalledWith(
      `${original.split("\n").slice(0, 5).join("\n")}\n`
    );
  });

  test("mapped placements still render context from the recorded revision", () => {
    const { container, bar } = renderAnchor({ anchor: anchor(4, 5), placement: current(10, 11) });
    expect(bar().getAttribute("data-lines")).toBe("4-5");
    expect(bar().getAttribute("data-link")).toBe("yes");
    expect(container.textContent).toContain("comment line");
    expect(container.textContent).not.toContain("CURRENT REVISION");
  });

  test("is Outdated with a reason when the comparison exceeds the limit", () => {
    const { bar, state, rerenderAnchor } = renderAnchor({ anchor: anchor(4, 5), placement: UNAVAILABLE_SIZE_LIMIT });
    expect(state()).toBe("OUTDATED");
    expect(bar().getAttribute("data-link")).toBeNull();
    expect(bar().getAttribute("data-tooltip-text")).toBe("plan.review.thread.anchor.comparison-size-limit");
    rerenderAnchor({ anchor: anchor(4, 5), placement: UNAVAILABLE });
    expect(state()).toBe("OUTDATED");
    expect(bar().getAttribute("data-tooltip-text")).toBeNull();
  });

  test("keeps the anchor unavailable when its original lines are outside the preview", () => {
    mocks.sheets[savedName] = create(SheetSchema, {
      name: savedName,
      content: new TextEncoder().encode("SELECT 1;"),
    });
    const { bar, state } = renderAnchor({ anchor: anchor(4, 5), placement: UNAVAILABLE_SIZE_LIMIT });
    expect(state()).toBe("UNAVAILABLE");
    expect(bar().getAttribute("data-tooltip-text")).toBeNull();
  });

  test.each([
    { sheet: "complete", text: original, extra: 0, start: 1, end: 1, state: "CURRENT", action: true },
    { sheet: "visible preview line", text: "SELECT 1;\nSELECT", extra: 1, start: 1, end: 1, state: "CURRENT", action: true },
    { sheet: "partial preview line", text: "SELECT 1;\nSELECT", extra: 1, start: 2, end: 2, state: "UNAVAILABLE", action: false },
    { sheet: "range crossing the preview end", text: "SELECT 1;\nSELECT", extra: 1, start: 1, end: 2, state: "UNAVAILABLE", action: false },
    { sheet: "empty", text: "", extra: 0, start: 1, end: 1, state: "UNAVAILABLE", action: false },
  ])("a hash-matched anchor on a $sheet sheet is $state", ({ text, extra, start, end, state: expected, action }) => {
    const content = new TextEncoder().encode(text);
    mocks.sheets[currentName] = create(SheetSchema, {
      name: currentName,
      content,
      contentSize: BigInt(content.byteLength + extra),
    });
    const onCurrent = buildWholeLineAnchor({ spec: "spec", sheetSha256: currentHash, startLine: start, endLine: end });
    const { bar, state } = renderAnchor({ anchor: onCurrent, placement: undefined });
    expect(state()).toBe(expected);
    expect(bar().getAttribute("data-link") !== null).toBe(action);
  });

  test("a hash-matched anchor stays pending until its sheet has loaded", async () => {
    const content = new TextEncoder().encode(original);
    const sheet = create(SheetSchema, { name: currentName, content, contentSize: BigInt(content.byteLength) });
    delete mocks.sheets[currentName];
    mocks.fetchSheet.mockResolvedValueOnce(sheet);
    const onCurrent = buildWholeLineAnchor({ spec: "spec", sheetSha256: currentHash, startLine: 1, endLine: 1 });
    const { bar, state } = renderAnchor({ anchor: onCurrent, placement: undefined });
    expect(state()).toBe("PENDING");
    expect(bar().getAttribute("data-link")).toBeNull();
    await waitFor(() => expect(state()).toBe("CURRENT"));
    expect(bar().getAttribute("data-link")).toBe("yes");
  });

  test("fetches a missing recorded revision instead of using the current SQL", async () => {
    delete mocks.sheets[savedName];
    const { container } = renderAnchor({ anchor: anchor(4, 5), placement: OUTDATED });
    await waitFor(() => expect(mocks.fetchSheet).toHaveBeenCalledWith(savedName));
    expect(container.querySelector("[data-line-number]")).toBeNull();
    expect(container.textContent).not.toContain("CURRENT REVISION");
  });
});
