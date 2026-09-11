import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import type * as monaco from "monaco-editor";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type {
  IStandaloneCodeEditor,
  MonacoModule,
} from "@/components/monaco/types";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  IssueComment_ThreadState,
  IssueCommentSchema,
  IssueSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { StatementAnchorSchema } from "@/types/proto-es/v1/issue_service_pb";

const SHA = "c".repeat(64);
const ISSUE = "projects/p/issues/1";

const mocks = vi.hoisted(() => ({
  comments: [] as unknown[],
  createIssueComment: vi.fn(),
  threadFocus: undefined as
    | { commentName: string; specId: string; nonce: number }
    | undefined,
  clearThreadFocus: vi.fn(),
  placements: new Map<string, unknown>(),
  placementTargets: new Map<string, string>(),
  hasPermission: vi.fn((_project: unknown, _permission: unknown) => true),
  findController: null as null | {
    getState: () => {
      isRevealed: boolean;
      onFindReplaceStateChange: (listener: () => void) => { dispose: () => void };
    };
  },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: { count?: number }) =>
      options?.count !== undefined ? `${key}:${options.count}` : key,
  }),
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/p" }),
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
  extractUserEmail: (identifier: string) => identifier.replace(/^users\//, ""),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({ getIssueComments: () => mocks.comments }),
    {
      getState: () => ({ createIssueComment: mocks.createIssueComment }),
    }
  ),
}));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: (project: unknown, permission: unknown) =>
    mocks.hasPermission(project, permission),
}));

vi.mock("../../hooks/usePlanChangeReferenceData", () => ({
  usePlanChangeReferenceData: () => ({}),
}));

vi.mock("../../shared/stores/usePlanDetailStore", () => ({
  usePlanDetailStore: (selector: (state: unknown) => unknown) =>
    selector({
      threadFocus: mocks.threadFocus,
      clearThreadFocus: mocks.clearThreadFocus,
      placements: mocks.placements,
      placementTargets: mocks.placementTargets,
    }),
}));

vi.mock("../../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({
    projectId: "p",
    plan: create(PlanSchema, {
      specs: [
        create(Plan_SpecSchema, {
          id: "spec-1",
          config: {
            case: "changeDatabaseConfig",
            value: create(Plan_ChangeDatabaseConfigSchema, {
              sheet: `projects/p/sheets/${"c".repeat(64)}`,
            }),
          },
        }),
      ],
    }),
  }),
}));

vi.mock("../PlanChangeReference", () => ({
  PlanSpecChangeReference: () => <span>hr-prod</span>,
}));

vi.mock("./CommentThreadCard", () => ({
  CommentThreadCard: ({
    onClose,
    thread,
  }: {
    onClose: () => void;
    thread: { root: { name: string } };
  }) => (
    <div data-testid="thread-card" data-root={thread.root.name}>
      <button data-testid="close-thread" onClick={onClose} type="button" />
    </div>
  ),
}));

vi.mock("./InlineThreadComposer", () => ({
  InlineThreadComposer: ({
    draft,
    onCancel,
    onDraftChange,
    onPublish,
    range,
  }: {
    draft: string;
    onCancel: () => void;
    onDraftChange: (draft: string) => void;
    onPublish: (comment: string) => Promise<boolean>;
    range: { startLine: number; endLine: number };
  }) => (
    <div data-testid="composer" data-draft={draft} data-range={`${range.startLine}-${range.endLine}`}>
      <button data-testid="cancel-composer" onClick={onCancel} type="button" />
      <button
        data-testid="type-draft"
        onClick={() => onDraftChange(`Draft ${range.startLine}-${range.endLine}`)}
        type="button"
      />
      <button
        data-testid="publish"
        onClick={() => void onPublish("No LIMIT")}
        type="button"
      />
    </div>
  ),
}));

vi.mock("./ThreadStack", () => ({
  ThreadStack: ({
    onExpand,
    onCollapse,
    replyDrafts,
    onReplyDraftChange,
    expandedRoots,
    threads,
  }: {
    onExpand: (rootName: string) => void;
    onCollapse: (rootName: string) => void;
    replyDrafts: Record<string, string>;
    onReplyDraftChange: (rootName: string, draft: string | ((current: string) => string)) => void;
    expandedRoots: ReadonlySet<string>;
    threads: { thread: { root: { name: string } } }[];
  }) => threads.length === 1 && expandedRoots.has(threads[0].thread.root.name) ? (
    <div data-testid="thread-card" data-root={threads[0].thread.root.name}>
      <button data-testid="close-thread" onClick={() => onCollapse(threads[0].thread.root.name)} type="button" />
      <button data-testid="save-draft" onClick={() => onReplyDraftChange(threads[0].thread.root.name, "unfinished reply")} type="button" />
      <button data-testid="clear-submitted-draft" onClick={() => onReplyDraftChange(threads[0].thread.root.name, current => current === "unfinished reply" ? "" : current)} type="button" />
      <input data-testid="saved-draft" readOnly value={replyDrafts[threads[0].thread.root.name] ?? ""} />
    </div>
  ) : (
    <ul data-testid="thread-picker">
      {threads.map((entry) => (
        <li key={entry.thread.root.name}>
          <button
            data-testid="pick-thread"
            data-expanded={expandedRoots.has(entry.thread.root.name)}
            onClick={() => onExpand(entry.thread.root.name)}
            type="button"
          >
            {entry.thread.root.name}
          </button>
          {expandedRoots.has(entry.thread.root.name) && (
            <button data-testid="close-thread" onClick={() => onCollapse(entry.thread.root.name)} type="button" />
          )}
        </li>
      ))}
    </ul>
  ),
}));

import { StatementThreadsLayer } from "./StatementThreadsLayer";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

type MouseHandler = (e: monaco.editor.IEditorMouseEvent) => void;

class FakeRange {
  constructor(
    public startLineNumber: number,
    public startColumn: number,
    public endLineNumber: number,
    public endColumn: number
  ) {}
}

const MouseTargetType = {
  GUTTER_GLYPH_MARGIN: 2,
  GUTTER_LINE_NUMBERS: 3,
  GUTTER_LINE_DECORATIONS: 4,
  CONTENT_TEXT: 6,
  CONTENT_EMPTY: 7,
};

const fakeMonaco = {
  Range: FakeRange,
  editor: {
    TrackedRangeStickiness: { NeverGrowsWhenTypingAtEdges: 1 },
    MouseTargetType,
    GlyphMarginLane: { Center: 2 },
  },
} as unknown as MonacoModule;

function createFakeEditor(lineMaxColumn = 20) {
  const handlers: Record<string, MouseHandler[]> = {
    move: [],
    down: [],
    up: [],
    leave: [],
  };
  const zones = new Map<string, { afterLineNumber: number; domNode: HTMLElement }>();
  const widgets = new Set<{ getDomNode: () => HTMLElement }>();
  let zoneSeq = 0;
  // The DOM hit test used by the drag; tests set the line the pointer is on.
  const pointer = { line: undefined as number | undefined };
  let selection: monaco.Selection | null = null;
  const selectionListeners = new Set<() => void>();
  const setSelection = (range: FakeRange) => {
    if (selection?.startLineNumber === range.startLineNumber &&
        selection.startColumn === range.startColumn &&
        selection.endLineNumber === range.endLineNumber &&
        selection.endColumn === range.endColumn) return;
    selection = { ...range, isEmpty: () => range.startLineNumber === range.endLineNumber && range.startColumn === range.endColumn } as monaco.Selection;
    for (const listener of selectionListeners) listener();
  };
  const decorations = { current: [] as monaco.editor.IModelDeltaDecoration[] };
  const domNode = document.createElement("div");
  document.body.append(domNode);
  domNode.scrollIntoView = vi.fn();
  const subscribe = (key: string) => (handler: MouseHandler) => {
    handlers[key].push(handler);
    return {
      dispose: () => {
        handlers[key] = handlers[key].filter((h) => h !== handler);
      },
    };
  };
  const editor = {
    focus: vi.fn(),
    addAction: vi.fn(() => ({ dispose: vi.fn() })),
    addOverlayWidget: (widget: { getDomNode: () => HTMLElement }) => {
      widgets.add(widget);
    },
    removeOverlayWidget: (widget: { getDomNode: () => HTMLElement }) => {
      widgets.delete(widget);
    },
    getContribution: () => mocks.findController,
    getLayoutInfo: () => ({
      contentLeft: 52,
      width: 800,
      verticalScrollbarWidth: 14,
      minimap: { minimapWidth: 0 },
    }),
    getTargetAtClientPoint: () =>
      pointer.line === undefined
        ? null
        : { type: MouseTargetType.CONTENT_TEXT, position: { lineNumber: pointer.line, column: 1 }, range: null },
    onDidLayoutChange: () => ({ dispose: vi.fn() }),
    changeViewZones: (
      callback: (accessor: {
        addZone: (zone: { afterLineNumber: number; domNode: HTMLElement }) => string;
        removeZone: (id: string) => void;
        layoutZone: (id: string) => void;
      }) => void
    ) =>
      callback({
        addZone: (zone) => {
          const id = `z${zoneSeq++}`;
          zones.set(id, zone);
          return id;
        },
        removeZone: (id) => {
          zones.delete(id);
        },
        layoutZone: () => {},
      }),
    createDecorationsCollection: () => ({
      set: (next: monaco.editor.IModelDeltaDecoration[]) => {
        decorations.current = next;
      },
      clear: () => {
        decorations.current = [];
      },
    }),
    getDomNode: () => domNode,
    getModel: () => ({ getLineMaxColumn: () => lineMaxColumn }),
    getSelection: () => selection,
    setSelection,
    setPosition: ({ lineNumber, column }: { lineNumber: number; column: number }) =>
      setSelection(new FakeRange(lineNumber, column, lineNumber, column)),
    onDidChangeCursorSelection: (listener: () => void) => {
      selectionListeners.add(listener);
      return { dispose: () => selectionListeners.delete(listener) };
    },
    onMouseDown: subscribe("down"),
    onMouseLeave: subscribe("leave"),
    onMouseMove: subscribe("move"),
    onMouseUp: subscribe("up"),
    revealLineInCenter: vi.fn(),
    revealLineNearTop: vi.fn(),
  };
  const fire = (key: string, line: number | undefined, type: number, detail?: Record<string, unknown>) => {
    const event = {
      event: { leftButton: true, preventDefault: vi.fn(), stopPropagation: vi.fn() },
      target: {
        type,
        detail,
        position: line === undefined ? null : { lineNumber: line, column: 1 },
        range: null,
      },
    } as unknown as monaco.editor.IEditorMouseEvent;
    act(() => {
      for (const handler of [...handlers[key]]) handler(event);
    });
    return event;
  };
  // Drag steps arrive through the document, not the editor's mouse events.
  const dragTo = (line: number) => {
    pointer.line = line;
    act(() => {
      document.dispatchEvent(new MouseEvent("mousemove", { bubbles: true }));
    });
  };
  const release = () => {
    act(() => {
      document.dispatchEvent(new MouseEvent("mouseup", { bubbles: true }));
    });
  };
  return {
    editor: editor as unknown as IStandaloneCodeEditor,
    decorations,
    pressEscape: () => act(() => {
      domNode.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
    }),
    dragTo,
    fire,
    release,
    widgets,
    zones,
  };
}

const at = (seconds: number) =>
  create(TimestampSchema, { seconds: BigInt(seconds) });

const threadRoot = (
  id: string,
  startLine: number,
  endLine: number,
  extra: { createdAt?: number; resolved?: boolean; sheetSha256?: string } = {}
) =>
  create(IssueCommentSchema, {
    name: `${ISSUE}/issueComments/${id}`,
    comment: id,
    createTime: at(extra.createdAt ?? 1),
    threadState: extra.resolved
      ? IssueComment_ThreadState.RESOLVED
      : IssueComment_ThreadState.OPEN,
    statementAnchor: create(StatementAnchorSchema, {
      spec: "spec-1",
      sheetSha256: extra.sheetSha256 ?? SHA,
      startPosition: create(PositionSchema, { line: startLine, column: 0 }),
      endPosition: create(PositionSchema, { line: endLine, column: 0 }),
    }),
  });

const issue = create(IssueSchema, { name: ISSUE });
const spec = create(Plan_SpecSchema, { id: "spec-1" });

let container: HTMLDivElement;
let root: Root;

beforeEach(() => {
  mocks.findController = null;
  vi.clearAllMocks();
  mocks.threadFocus = undefined;
  mocks.hasPermission.mockReturnValue(true);
  container = document.createElement("div");
  root = createRoot(container);
});

afterEach(() => {
  act(() => root.unmount());
  document.body.replaceChildren();
});

const mount = (editor: IStandaloneCodeEditor) =>
  act(() =>
    root.render(
      <StatementThreadsLayer
        editor={editor}
        issue={issue}
        monaco={fakeMonaco}
        sheetSha256={SHA}
        spec={spec}
      />
    )
  );

const zoneAfter = (zones: Map<string, { afterLineNumber: number; domNode: HTMLElement }>) =>
  Array.from(zones.values()).map((zone) => zone.afterLineNumber);

const hosted = (
  widgets: Set<{ getDomNode: () => HTMLElement }>,
  selector: string
) =>
  Array.from(widgets)
    .map((widget) => widget.getDomNode().querySelector(selector))
    .find(Boolean) ?? null;

const cardRoot = (widgets: Set<{ getDomNode: () => HTMLElement }>) =>
  hosted(widgets, "[data-testid='thread-card']")?.getAttribute("data-root");

const classesOf = (
  decorations: monaco.editor.IModelDeltaDecoration[],
  line: number
) =>
  decorations
    .filter((d) => d.range.startLineNumber === line)
    .map(
      (d) => d.options.className ?? d.options.glyphMarginClassName ?? d.options.linesDecorationsClassName ?? ""
    );

const lines = (
  decorations: monaco.editor.IModelDeltaDecoration[],
  className: string
) =>
  decorations
    .filter((d) => d.options.className === className)
    .map((d) => d.range.startLineNumber);

const walkerNode = (widgets: Set<{ getDomNode: () => HTMLElement }>) =>
  Array.from(widgets)
    .map((widget) => widget.getDomNode())
    .find((node) => node.querySelector("[data-testid='thread-walker']"));

const pressWalker = (widgets: Set<{ getDomNode: () => HTMLElement }>, label: string) =>
  act(() => {
    hosted(widgets, `button[aria-label='plan.review.thread.walker.${label}']`)?.dispatchEvent(
      new MouseEvent("click", { bubbles: true })
    );
  });

describe("StatementThreadsLayer", () => {
  test("decorates current anchors and expands the earliest unresolved thread", () => {
    mocks.comments = [
      threadRoot("later", 6, 9, { createdAt: 5 }),
      threadRoot("first", 3, 4, { createdAt: 2 }),
      threadRoot("resolved", 1, 1, { createdAt: 1, resolved: true }),
      threadRoot("also-on-9", 8, 9, { createdAt: 7 }),
    ];
    const fake = createFakeEditor();
    mount(fake.editor);

    expect(classesOf(fake.decorations.current, 3)).toEqual(["bb-thread-line"]);
    expect(classesOf(fake.decorations.current, 8)).toEqual(["bb-thread-line--passive"]);
    expect(classesOf(fake.decorations.current, 4)).toContain("bb-thread-glyph bb-thread-glyph--active");
    expect(classesOf(fake.decorations.current, 1)).toContain(
      "bb-thread-glyph bb-thread-glyph--resolved"
    );
    expect(classesOf(fake.decorations.current, 9)).toContain(
      "bb-thread-glyph bb-thread-glyph--count-2"
    );
    expect(zoneAfter(fake.zones)).toEqual([4]);
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/first`);
  });

  test("distinguishes the current range from non-stacking passive ranges", () => {
    mocks.comments = [threadRoot("single", 2, 2), threadRoot("multi", 1, 2, { createdAt: 2, resolved: true })];
    const fake = createFakeEditor();
    mount(fake.editor);
    const highlighted = () => fake.decorations.current.filter((d) => d.options.className === "bb-thread-line").map((d) => d.range.startLineNumber);
    expect(highlighted()).toEqual([2]);
    expect(classesOf(fake.decorations.current, 1)).toContain("bb-thread-line--passive");
    expect(fake.decorations.current.filter((d) => d.options.isWholeLine && d.range.startLineNumber === 2)).toHaveLength(1);
    fake.fire("move", 1, MouseTargetType.GUTTER_LINE_NUMBERS);
    expect(highlighted()).toEqual([2]);
    const picker = hosted(fake.widgets, "[data-testid='thread-picker']");
    act(() => picker?.querySelectorAll("[data-testid='pick-thread']")[1].dispatchEvent(new MouseEvent("click", { bubbles: true })));
    expect(highlighted()).toEqual([1, 2]);
    act(() => hosted(fake.widgets, "[data-testid='close-thread']")?.dispatchEvent(new MouseEvent("click", { bubbles: true })));
    expect(highlighted()).toEqual([]);
    expect(fake.decorations.current.filter((d) => d.options.className === "bb-thread-line--passive").map((d) => d.range.startLineNumber)).toEqual([1, 2]);
    expect(fake.decorations.current.filter((d) => d.options.glyphMarginClassName?.includes("bb-thread-glyph--count-2"))).toHaveLength(1);
  });

  test("a pending selection replaces the reading highlight; an open composer keeps both", () => {
    mocks.comments = [threadRoot("single", 2, 2)];
    const fake = createFakeEditor(1);
    mount(fake.editor);
    fake.fire("down", 1, MouseTargetType.GUTTER_LINE_NUMBERS);
    fake.dragTo(2);
    fake.release();
    expect(lines(fake.decorations.current, "bb-thread-line")).toEqual([]);
    expect(lines(fake.decorations.current, "bb-thread-line--selecting")).toEqual([1, 2]);
    fake.pressEscape();
    expect(lines(fake.decorations.current, "bb-thread-line")).toEqual([2]);
    expect(lines(fake.decorations.current, "bb-thread-line--selecting")).toEqual([]);
    fake.fire("down", 3, MouseTargetType.GUTTER_LINE_NUMBERS);
    fake.release();
    fake.fire("down", 3, MouseTargetType.GUTTER_LINE_DECORATIONS);
    expect(lines(fake.decorations.current, "bb-thread-line")).toEqual([2]);
    expect(lines(fake.decorations.current, "bb-thread-line--selecting")).toEqual([3]);
    expect(lines(fake.decorations.current, "bb-thread-line--passive")).toEqual([]);
    act(() => hosted(fake.widgets, "[data-testid='cancel-composer']")?.dispatchEvent(new MouseEvent("click", { bubbles: true })));
    expect(lines(fake.decorations.current, "bb-thread-line")).toEqual([2]);
    expect(lines(fake.decorations.current, "bb-thread-line--selecting")).toEqual([]);
  });

  test("composers on different lines stay open together and close independently", async () => {
    mocks.comments = [];
    mocks.createIssueComment.mockResolvedValue(
      create(IssueCommentSchema, { name: `${ISSUE}/issueComments/new` })
    );
    const fake = createFakeEditor();
    mount(fake.editor);
    const composers = () =>
      Array.from(fake.widgets)
        .flatMap((widget) => Array.from(widget.getDomNode().querySelectorAll("[data-testid='composer']")))
        .map((node) => node.getAttribute("data-range"));
    const composerAt = (range: string) => hosted(fake.widgets, `[data-testid='composer'][data-range='${range}']`);

    fake.fire("move", 2, MouseTargetType.CONTENT_TEXT);
    fake.fire("down", 2, MouseTargetType.GUTTER_LINE_DECORATIONS);
    expect(composers()).toEqual(["2-2"]);
    act(() => composerAt("2-2")?.querySelector("[data-testid='type-draft']")?.dispatchEvent(new MouseEvent("click", { bubbles: true })));
    expect(composerAt("2-2")?.getAttribute("data-draft")).toBe("Draft 2-2");

    // The add action stays available on other lines while a form is open.
    fake.fire("move", 5, MouseTargetType.CONTENT_TEXT);
    expect(classesOf(fake.decorations.current, 5)).toContain("bb-thread-add-glyph");
    fake.fire("down", 5, MouseTargetType.GUTTER_LINE_NUMBERS);
    fake.dragTo(6);
    fake.release();
    fake.fire("down", 6, MouseTargetType.GUTTER_LINE_DECORATIONS);
    expect(composers()).toEqual(["2-2", "5-6"]);
    expect(zoneAfter(fake.zones)).toEqual([2, 6]);
    expect(lines(fake.decorations.current, "bb-thread-line--selecting")).toEqual([2, 5, 6]);
    expect(composerAt("2-2")?.getAttribute("data-draft")).toBe("Draft 2-2");

    // The action is hidden on a line that already carries a form.
    fake.fire("move", 2, MouseTargetType.CONTENT_TEXT);
    expect(classesOf(fake.decorations.current, 2)).not.toContain("bb-thread-add-glyph");

    act(() => composerAt("5-6")?.querySelector("[data-testid='cancel-composer']")?.dispatchEvent(new MouseEvent("click", { bubbles: true })));
    expect(composers()).toEqual(["2-2"]);
    expect(lines(fake.decorations.current, "bb-thread-line--selecting")).toEqual([2]);

    await act(async () => {
      composerAt("2-2")?.querySelector("[data-testid='publish']")?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
    expect(mocks.createIssueComment.mock.calls[0][0].statementAnchor).toMatchObject({
      startPosition: { line: 2, column: 0 },
      endPosition: { line: 2, column: 0 },
    });
    expect(composers()).toEqual([]);
  });

  test("the comment action on a form's own line reopens it with its draft and range intact", () => {
    mocks.comments = [];
    const fake = createFakeEditor();
    mount(fake.editor);
    fake.fire("down", 3, MouseTargetType.GUTTER_LINE_NUMBERS);
    fake.dragTo(4);
    fake.release();
    fake.fire("down", 4, MouseTargetType.GUTTER_LINE_DECORATIONS);
    const composer = () => hosted(fake.widgets, "[data-testid='composer']");
    expect(composer()?.getAttribute("data-range")).toBe("3-4");
    act(() => composer()?.querySelector("[data-testid='type-draft']")?.dispatchEvent(new MouseEvent("click", { bubbles: true })));

    // A fresh single-line selection ending on the same line targets the
    // existing form instead of replacing it.
    fake.fire("down", 4, MouseTargetType.GUTTER_LINE_NUMBERS);
    fake.release();
    expect(classesOf(fake.decorations.current, 4)).not.toContain("bb-thread-add-glyph");
    const addAction = (fake.editor as unknown as { addAction: ReturnType<typeof vi.fn> }).addAction;
    const commentOnLines = addAction.mock.calls.at(-1)?.[0] as { run: () => void };
    act(() => commentOnLines.run());
    expect(fake.zones.size).toBe(1);
    expect(composer()?.getAttribute("data-range")).toBe("3-4");
    expect(composer()?.getAttribute("data-draft")).toBe("Draft 3-4");
  });

  test("a marker opens its thread, a combined marker offers a picker, and closing returns to markers", () => {
    mocks.comments = [
      threadRoot("first", 3, 4, { createdAt: 2 }),
      threadRoot("a", 6, 9, { createdAt: 5 }),
      threadRoot("b", 8, 9, { createdAt: 7 }),
    ];
    const fake = createFakeEditor();
    mount(fake.editor);
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/first`);

    // Moving onto an occupied line must not replace its marker with creation.
    fake.fire("move", 9, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect(classesOf(fake.decorations.current, 9)).toContain("bb-thread-add-glyph");
    fake.fire("down", 9, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect(hosted(fake.widgets, "[data-testid='composer']")).toBeNull();
    expect(zoneAfter(fake.zones)).toEqual([9]);
    const picker = hosted(fake.widgets, "[data-testid='thread-picker']");
    const pick = picker?.querySelectorAll("[data-testid='pick-thread']");
    expect(Array.from(pick ?? []).map((node) => node.textContent)).toEqual([
      `${ISSUE}/issueComments/a`,
      `${ISSUE}/issueComments/b`,
    ]);
    act(() => {
      pick?.[1].dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(zoneAfter(fake.zones)).toEqual([9]);
    expect(pick?.[1].getAttribute("data-expanded")).toBe("true");

    // The marker of the expanded thread toggles it closed.
    fake.fire("down", 4, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/first`);
    fake.fire("down", 4, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect(fake.zones.size).toBe(0);
  });

  test("selecting lines reveals the add action, which opens and publishes a whole-line anchor", async () => {
    mocks.comments = [];
    mocks.createIssueComment.mockResolvedValue(
      create(IssueCommentSchema, { name: `${ISSUE}/issueComments/new` })
    );
    const fake = createFakeEditor();
    mount(fake.editor);

    fake.fire("move", 2, MouseTargetType.GUTTER_LINE_NUMBERS);
    expect(classesOf(fake.decorations.current, 2)).toEqual([
      "bb-thread-add-glyph",
    ]);
    fake.fire("move", 2, MouseTargetType.CONTENT_TEXT);
    expect(classesOf(fake.decorations.current, 2)).toEqual(["bb-thread-add-glyph"]);
    fake.fire("leave", undefined, MouseTargetType.CONTENT_TEXT);
    expect(classesOf(fake.decorations.current, 2)).toEqual([]);

    const down = fake.fire("down", 2, MouseTargetType.GUTTER_LINE_NUMBERS);
    expect(down.event.preventDefault).toHaveBeenCalled();
    fake.dragTo(4);
    fake.release();
    expect(fake.zones.size).toBe(0);
    expect(classesOf(fake.decorations.current, 4)).toContain("bb-thread-add-glyph");
    fake.fire("down", 4, MouseTargetType.GUTTER_LINE_DECORATIONS);
    expect(classesOf(fake.decorations.current, 3)).toEqual([
      "bb-thread-line--selecting",
    ]);

    expect(zoneAfter(fake.zones)).toEqual([4]);
    const composer = hosted(fake.widgets, "[data-testid='composer']");
    expect(composer?.getAttribute("data-range")).toBe("2-4");

    await act(async () => {
      hosted(fake.widgets, "[data-testid='publish']")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(mocks.createIssueComment).toHaveBeenCalledTimes(1);
    const request = mocks.createIssueComment.mock.calls[0][0];
    expect(request).toMatchObject({ issueName: ISSUE, comment: "No LIMIT" });
    expect(request.statementAnchor).toMatchObject({
      spec: "spec-1",
      sheetSha256: SHA,
      startPosition: { line: 2, column: 0 },
      endPosition: { line: 4, column: 0 },
    });
    expect(fake.zones.size).toBe(0);
  });

  test("a press on the line numbers also selects comment lines", () => {
    mocks.comments = [];
    const fake = createFakeEditor();
    mount(fake.editor);
    fake.fire("down", 5, MouseTargetType.GUTTER_LINE_NUMBERS);
    fake.dragTo(3);
    fake.release();
    expect(fake.zones.size).toBe(0);
    fake.fire("down", 5, MouseTargetType.GUTTER_LINE_DECORATIONS);
    expect(zoneAfter(fake.zones)).toEqual([5]);
    expect(
      hosted(fake.widgets, "[data-testid='composer']")?.getAttribute("data-range")
    ).toBe("3-5");
  });

  test("creation from an occupied line preserves the selected range", () => {
    mocks.comments = [threadRoot("existing", 4, 4)];
    const fake = createFakeEditor();
    mount(fake.editor);
    fake.fire("down", 2, MouseTargetType.GUTTER_LINE_NUMBERS);
    fake.dragTo(4);
    fake.release();
    expect(classesOf(fake.decorations.current, 4)).toContain("bb-thread-add-glyph");
    expect(fake.decorations.current.filter((d) => d.range.startLineNumber === 4 && d.options.glyphMarginClassName)).toHaveLength(1);
    fake.fire("down", 4, MouseTargetType.GUTTER_LINE_DECORATIONS);
    expect(hosted(fake.widgets, "[data-testid='composer']")?.getAttribute("data-range")).toBe("2-4");
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/existing`);
  });

  test("Escape releases the pinned creation action without replacing the count", () => {
    mocks.comments = [threadRoot("a", 4, 4), threadRoot("b", 4, 4)];
    const fake = createFakeEditor(1);
    mount(fake.editor);
    fake.fire("move", 4, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect(classesOf(fake.decorations.current, 4)).toContain("bb-thread-add-glyph");
    fake.fire("down", 4, MouseTargetType.GUTTER_LINE_NUMBERS);
    fake.release();
    expect(classesOf(fake.decorations.current, 4)).toContain("bb-thread-add-glyph");
    expect(classesOf(fake.decorations.current, 4)).toContain("bb-thread-glyph bb-thread-glyph--count-2 bb-thread-glyph--active");
    fake.pressEscape();
    expect(classesOf(fake.decorations.current, 4)).not.toContain("bb-thread-add-glyph");
    expect(classesOf(fake.decorations.current, 4)).toContain("bb-thread-glyph bb-thread-glyph--count-2 bb-thread-glyph--active");
    expect(hosted(fake.widgets, "[data-testid='composer']")).toBeNull();
  });

  test("whole-line hover exposes creation beside stable thread markers", () => {
    mocks.comments = [threadRoot("a", 4, 4), threadRoot("b", 4, 4)];
    const fake = createFakeEditor();
    mount(fake.editor);
    fake.fire("move", 4, MouseTargetType.CONTENT_TEXT);
    const marker = fake.decorations.current.find((d) => d.options.glyphMarginClassName);
    expect(marker?.options.glyphMarginClassName).toContain("bb-thread-glyph--count-2");
    const add = fake.decorations.current.find((d) => d.options.linesDecorationsClassName === "bb-thread-add-glyph");
    expect(add?.range.startLineNumber).toBe(4);
    expect(add?.options.glyphMarginClassName).toBeUndefined();
    fake.fire("down", 4, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect(fake.zones.size).toBe(0);
    expect(hosted(fake.widgets, "[data-testid='composer']")).toBeNull();
    fake.fire("down", 4, MouseTargetType.GUTTER_LINE_DECORATIONS);
    expect(hosted(fake.widgets, "[data-testid='composer']")?.getAttribute("data-range")).toBe("4-4");
  });

  test("selection pins the action to its last line and the glyph remains a viewing action", () => {
    mocks.comments = [threadRoot("a", 4, 4)];
    const fake = createFakeEditor();
    mount(fake.editor);
    fake.fire("down", 2, MouseTargetType.GUTTER_LINE_NUMBERS);
    fake.dragTo(4);
    fake.release();
    fake.fire("move", 7, MouseTargetType.CONTENT_TEXT);
    expect(fake.decorations.current.find((d) => d.options.linesDecorationsClassName === "bb-thread-add-glyph")?.range.startLineNumber).toBe(4);
    fake.fire("down", 4, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect(fake.decorations.current.filter((d) => d.options.className === "bb-thread-line--selecting")).toHaveLength(0);
    expect(hosted(fake.widgets, "[data-testid='composer']")).toBeNull();
  });

  test("blank line hover offers creation but space after the file does not", () => {
    mocks.comments = [];
    const fake = createFakeEditor();
    mount(fake.editor);
    fake.fire("move", 4, MouseTargetType.CONTENT_EMPTY, { isAfterLines: false });
    expect(classesOf(fake.decorations.current, 4)).toContain("bb-thread-add-glyph");
    fake.fire("move", 4, MouseTargetType.CONTENT_EMPTY, { isAfterLines: true });
    expect(classesOf(fake.decorations.current, 4)).toEqual([]);
  });

  test("an all-resolved group keeps its count and resolved state", () => {
    mocks.comments = [threadRoot("a", 2, 4, { resolved: true }), threadRoot("b", 4, 4, { resolved: true })];
    const fake = createFakeEditor();
    mount(fake.editor);
    const marker = fake.decorations.current.find((d) => d.options.glyphMarginClassName);
    expect(marker?.options.glyphMarginClassName).toBe("bb-thread-glyph bb-thread-glyph--count-2 bb-thread-glyph--resolved");
    expect(marker?.options.glyphMarginHoverMessage).toEqual({value: "plan.review.thread.threads-on-line:2"});
  });

  test("without create permission the gutter offers no affordance", () => {
    mocks.comments = [];
    mocks.hasPermission.mockReturnValue(false);
    const fake = createFakeEditor();
    mount(fake.editor);
    fake.fire("move", 2, MouseTargetType.GUTTER_LINE_NUMBERS);
    expect(fake.decorations.current).toEqual([]);
    fake.fire("down", 2, MouseTargetType.GUTTER_GLYPH_MARGIN);
    fake.release();
    expect(fake.zones.size).toBe(0);
    expect(
      (fake.editor as unknown as { addAction: ReturnType<typeof vi.fn> }).addAction
    ).not.toHaveBeenCalled();
  });

  test("status refreshes do not switch or close the automatically expanded thread", () => {
    mocks.comments = [threadRoot("first", 3, 4), threadRoot("other", 6, 9)];
    const fake = createFakeEditor();
    mount(fake.editor);
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/first`);
    mocks.comments = [threadRoot("first", 3, 4, { resolved: true }), threadRoot("other", 6, 9)];
    mount(fake.editor);
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/first`);
    mocks.comments = [threadRoot("first", 3, 4, { resolved: true }), threadRoot("other", 6, 9, { resolved: true })];
    mount(fake.editor);
    expect(zoneAfter(fake.zones)).toEqual([4]);
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/first`);
  });

  test("keeps reply drafts when the entire gutter group closes and reopens", () => {
    mocks.comments = [threadRoot("first", 3, 4)];
    const fake = createFakeEditor();
    mount(fake.editor);
    act(() => hosted(fake.widgets, "[data-testid='save-draft']")?.dispatchEvent(new MouseEvent("click", {bubbles: true})));
    fake.fire("down", 4, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect(fake.zones.size).toBe(0);
    fake.fire("down", 4, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect((hosted(fake.widgets, "[data-testid='saved-draft']") as HTMLInputElement)?.value).toBe("unfinished reply");
    act(() => hosted(fake.widgets, "[data-testid='clear-submitted-draft']")?.dispatchEvent(new MouseEvent("click", {bubbles: true})));
    expect((hosted(fake.widgets, "[data-testid='saved-draft']") as HTMLInputElement)?.value).toBe("");
  });

  test("a focus request expands the thread, reveals its line, and is cleared", () => {
    mocks.comments = [
      threadRoot("first", 3, 4, { createdAt: 2 }),
      threadRoot("target", 10, 12, { createdAt: 5 }),
    ];
    mocks.threadFocus = {
      commentName: `${ISSUE}/issueComments/target`,
      specId: "spec-1",
      nonce: 7,
    };
    const fake = createFakeEditor();
    mount(fake.editor);
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/target`);
    expect(
      (fake.editor as unknown as { revealLineInCenter: ReturnType<typeof vi.fn> })
        .revealLineInCenter
    ).toHaveBeenCalledWith(10);
    expect(mocks.clearThreadFocus).toHaveBeenCalledWith(7);
  });
  test("the walker steps through unresolved threads in editor order and wraps", () => {
    mocks.comments = [
      threadRoot("a", 2, 2, { createdAt: 1 }),
      threadRoot("done", 5, 5, { createdAt: 2, resolved: true }),
      threadRoot("c", 7, 8, { createdAt: 3 }),
    ];
    const fake = createFakeEditor();
    mount(fake.editor);
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/a`);
    const control = walkerNode(fake.widgets);
    expect(control?.textContent).toContain("2");
    expect([control?.style.top, control?.style.right]).toEqual(["8px", "22px"]);
    expect(hosted(fake.widgets, "[data-testid='thread-walker']")?.getAttribute("title")).toBe(
      "plan.review.thread.walker.count:2"
    );

    pressWalker(fake.widgets, "next");
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/c`);
    expect(zoneAfter(fake.zones)).toEqual([8]);
    expect(fake.editor.revealLineNearTop).toHaveBeenLastCalledWith(7);
    expect(fake.editor.getDomNode()?.scrollIntoView).toHaveBeenCalledWith({ block: "nearest" });
    pressWalker(fake.widgets, "next");
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/a`);
    pressWalker(fake.widgets, "previous");
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/c`);

    // Closing the open line leaves no current thread: down starts over, up ends.
    fake.fire("down", 8, MouseTargetType.GUTTER_GLYPH_MARGIN);
    expect(fake.zones.size).toBe(0);
    pressWalker(fake.widgets, "previous");
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/c`);
  });

  test("the walker is absent without unresolved placed threads and reports unplaced ones", () => {
    mocks.comments = [threadRoot("done", 2, 2, { resolved: true })];
    const fake = createFakeEditor();
    mount(fake.editor);
    expect(hosted(fake.widgets, "[data-testid='thread-walker']")).toBeNull();

    mocks.comments = [
      threadRoot("placed", 2, 2),
      threadRoot("elsewhere", 4, 4, { sheetSha256: "d".repeat(64) }),
    ];
    mount(fake.editor);
    expect(hosted(fake.widgets, "[data-testid='thread-walker']")?.getAttribute("title")).toBe(
      "plan.review.thread.walker.count:1 · plan.review.thread.walker.remainder:1"
    );
    pressWalker(fake.widgets, "next");
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/placed`);
  });

  test("resolving the current thread moves the walker on, and the last resolution removes it", () => {
    mocks.comments = [threadRoot("a", 2, 2, { createdAt: 1 }), threadRoot("b", 5, 5, { createdAt: 2 })];
    const fake = createFakeEditor();
    mount(fake.editor);
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/a`);
    mocks.comments = [threadRoot("a", 2, 2, { createdAt: 1, resolved: true }), threadRoot("b", 5, 5, { createdAt: 2 })];
    mount(fake.editor);
    expect(hosted(fake.widgets, "[data-testid='thread-walker']")?.textContent).toContain("1");
    pressWalker(fake.widgets, "next");
    expect(cardRoot(fake.widgets)).toBe(`${ISSUE}/issueComments/b`);
    mocks.comments = [threadRoot("a", 2, 2, { createdAt: 1, resolved: true }), threadRoot("b", 5, 5, { createdAt: 2, resolved: true })];
    mount(fake.editor);
    expect(hosted(fake.widgets, "[data-testid='thread-walker']")).toBeNull();
  });

  test("the walker yields the corner to the find widget", () => {
    let notify = () => {};
    const state = {
      isRevealed: true,
      onFindReplaceStateChange: (listener: () => void) => {
        notify = listener;
        return { dispose: vi.fn() };
      },
    };
    mocks.findController = { getState: () => state };
    mocks.comments = [threadRoot("a", 2, 2)];
    const fake = createFakeEditor();
    mount(fake.editor);
    expect(hosted(fake.widgets, "[data-testid='thread-walker']")).toBeNull();
    state.isRevealed = false;
    act(() => notify());
    expect(hosted(fake.widgets, "[data-testid='thread-walker']")).not.toBeNull();
  });
});

