import type * as monaco from "monaco-editor";
import {
  type SetStateAction,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { MonacoViewZone } from "@/components/monaco/MonacoViewZone";
import type {
  IStandaloneCodeEditor,
  MonacoModule,
} from "@/components/monaco/types";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { placementsForSheet } from "../../shared/stores/placementSlice";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { InlineThreadComposer } from "./InlineThreadComposer";
import "./statementThreads.css";
import { ThreadStack } from "./ThreadStack";
import {
  buildWholeLineAnchor,
  defaultExpandedThread,
  type EditorThread,
  groupMarkersByLine,
  groupThreads,
  type LineRange,
  lineRangeLabel,
  selectEditorThreads,
  selectionLineRange,
} from "./threadModel";
import { canReplyToThread, useThreadActions } from "./useThreadActions";

// The thread to expand when a line opens: its first unresolved thread, or
// its first thread when all are resolved.
const firstToExpand = (threads: EditorThread[] | undefined): string[] => {
  const entry = threads && (defaultExpandedThread(threads) ?? threads[0]);
  return entry ? [entry.thread.root.name] : [];
};

// Binds comment threads to the read-only statement editor with Monaco's own
// primitives, the way VS Code's comment controller does: glyph-margin
// decorations for markers and the comment affordance, whole-line decorations
// for highlights, and view zones for the composer and the expanded threads.
//
// Creation is a two-step gesture. Selecting lines (a press or drag in the
// gutter, or a text selection) shows the comment action on the last selected
// line. The creation action sits between the line numbers and SQL; thread
// markers stay in the glyph lane and always open the existing discussions.
export function StatementThreadsLayer({
  editor,
  issue,
  monaco: monacoModule,
  sheetSha256,
  spec,
}: {
  editor: IStandaloneCodeEditor;
  issue: Issue;
  monaco: MonacoModule;
  sheetSha256: string;
  spec: Plan_Spec;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const project = useProjectByName(`${projectNamePrefix}${page.projectId}`);
  const comments = useAppStore((state) => state.getIssueComments(issue.name));
  // Placements computed for another revision of this spec say nothing about
  // the displayed one; until the run for this sheet lands, only the no-diff
  // rule applies.
  const placements = usePlanDetailStore((s) =>
    placementsForSheet(s, spec.id, sheetSha256)
  );
  const actions = useThreadActions(issue.name);
  const canCreate = canReplyToThread(project);

  const editorThreads = useMemo(
    () =>
      selectEditorThreads(
        groupThreads(comments),
        { specId: spec.id, sheetSha256 },
        placements
      ),
    [comments, placements, sheetSha256, spec.id]
  );
  const markersByLine = useMemo(
    () => groupMarkersByLine(editorThreads),
    [editorThreads]
  );

  // A union of commented lines, so overlapping threads never darken a line.
  const commentedLines = useMemo(() => {
    const lines = new Set<number>();
    for (const { range } of editorThreads) {
      for (let line = range.startLine; line <= range.endLine; line++)
        lines.add(line);
    }
    return [...lines].sort((a, b) => a - b);
  }, [editorThreads]);

  // Which line's threads are open, and which of them are expanded. By
  // default the first line with an unresolved thread opens with its first
  // unresolved thread expanded and the others collapsed; the user's choices
  // win once they open or close anything.
  const [openLine, setOpenLine] = useState<number | undefined>();
  const [expandedRoots, setExpandedRoots] = useState<ReadonlySet<string>>(
    () => new Set()
  );
  const expandedThreadRef = useRef<HTMLDivElement>(null);
  const initialExpansionAppliedRef = useRef(false);
  useEffect(() => {
    if (initialExpansionAppliedRef.current) return;
    const line = defaultExpandedThread(editorThreads)?.range.endLine;
    if (line === undefined) return;
    initialExpansionAppliedRef.current = true;
    setOpenLine(line);
    setExpandedRoots(new Set(firstToExpand(markersByLine.get(line))));
  }, [editorThreads, markersByLine]);

  const openThreads = useCallback(
    (line: number | undefined, roots: Iterable<string>) => {
      initialExpansionAppliedRef.current = true;
      setOpenLine(line);
      setExpandedRoots(new Set(roots));
    },
    []
  );

  // Keep drafts across card collapses and closing/reopening a gutter group.
  const [replyDrafts, setReplyDrafts] = useState<Record<string, string>>({});
  const updateReplyDraft = useCallback(
    (root: string, update: SetStateAction<string>) => {
      setReplyDrafts((previous) => {
        const draft =
          typeof update === "function" ? update(previous[root] ?? "") : update;
        if (previous[root] === draft) return previous;
        const next = { ...previous };
        if (draft) next[root] = draft;
        else delete next[root];
        return next;
      });
    },
    []
  );

  const [hoveredLine, setHoveredLine] = useState<number | undefined>();
  const [selection, setSelection] = useState<LineRange | undefined>();
  const dragRef = useRef<LineRange | undefined>(undefined);
  const [composerRange, setComposerRange] = useState<LineRange | undefined>();

  // The line that carries the comment action: the end of the selection, or
  // the hovered line when nothing is selected.
  const actionLine = canCreate
    ? (selection?.endLine ?? hoveredLine)
    : undefined;

  // "View in Statement" from the timeline.
  const threadFocus = usePlanDetailStore((s) => s.threadFocus);
  const clearThreadFocus = usePlanDetailStore((s) => s.clearThreadFocus);
  useEffect(() => {
    if (!threadFocus || threadFocus.specId !== spec.id) return;
    const target = editorThreads.find(
      (entry) => entry.thread.root.name === threadFocus.commentName
    );
    if (!target) {
      // Anchors are current only for the saved version; a thread that is not
      // placed here has nothing to reveal.
      if (comments.length > 0) clearThreadFocus(threadFocus.nonce);
      return;
    }
    openThreads(target.range.endLine, [target.thread.root.name]);
    editor.revealLineInCenter(target.range.startLine);
    editor
      .getDomNode()
      ?.scrollIntoView({ behavior: "smooth", block: "center" });
    clearThreadFocus(threadFocus.nonce);
  }, [
    clearThreadFocus,
    comments.length,
    editor,
    editorThreads,
    openThreads,
    spec.id,
    threadFocus,
  ]);

  // Track the editor's selection as a line range (empty = none).
  useEffect(() => {
    const read = () => {
      const current = editor.getSelection();
      if (!current || current.isEmpty()) {
        setSelection(undefined);
        return;
      }
      setSelection(selectionLineRange(current));
    };
    read();
    const subscription = editor.onDidChangeCursorSelection(read);
    return () => subscription.dispose();
  }, [editor]);

  // Decorations: highlights, markers, the comment action, the composer range.
  useEffect(() => {
    const collection = editor.createDecorationsCollection();
    const decorations: monaco.editor.IModelDeltaDecoration[] = [];
    const stickiness =
      monacoModule.editor.TrackedRangeStickiness.NeverGrowsWhenTypingAtEdges;
    // Exactly one background per line: passive, current, or creation.
    // Higher-priority states replace the base instead of layering over it.
    const lineClasses = new Map<number, string>(
      commentedLines.map((line) => [line, "bb-thread-line--passive"])
    );
    const activeThread = editorThreads.find(
      (entry) =>
        entry.range.endLine === openLine &&
        expandedRoots.has(entry.thread.root.name)
    );
    const creationRange = composerRange ?? (canCreate ? selection : undefined);
    const range = creationRange ?? activeThread?.range;
    if (range) {
      for (let line = range.startLine; line <= range.endLine; line++) {
        lineClasses.set(
          line,
          creationRange ? "bb-thread-line--selecting" : "bb-thread-line"
        );
      }
    }
    for (const [line, className] of [...lineClasses].sort(
      ([a], [b]) => a - b
    )) {
      decorations.push({
        range: new monacoModule.Range(line, 1, line, 1),
        options: {
          isWholeLine: true,
          className,
          stickiness,
        },
      });
    }
    for (const [line, threads] of markersByLine) {
      const count = threads.length;
      const classes = ["bb-thread-glyph"];
      if (count > 1) {
        classes.push(
          count > 9
            ? "bb-thread-glyph--count-many"
            : `bb-thread-glyph--count-${count}`
        );
      }
      if (threads.every((entry) => entry.thread.resolved)) {
        classes.push("bb-thread-glyph--resolved");
      }
      if (line === openLine) classes.push("bb-thread-glyph--active");
      decorations.push({
        range: new monacoModule.Range(line, 1, line, 1),
        options: {
          glyphMarginClassName: classes.join(" "),
          glyphMargin: { position: monacoModule.editor.GlyphMarginLane.Center },
          glyphMarginHoverMessage: {
            value: t("plan.review.thread.threads-on-line", { count }),
          },
          stickiness,
        },
      });
    }
    if (actionLine !== undefined && !composerRange) {
      const actionRange = selection ?? {
        startLine: actionLine,
        endLine: actionLine,
      };
      decorations.push({
        range: new monacoModule.Range(actionLine, 1, actionLine, 1),
        options: {
          linesDecorationsClassName: "bb-thread-add-glyph",
          linesDecorationsTooltip: `${t("plan.review.thread.add-comment")} · ${lineRangeLabel(t, actionRange)}`,
          stickiness,
        },
      });
    }
    collection.set(decorations);
    return () => {
      collection.clear();
    };
  }, [
    actionLine,
    canCreate,
    commentedLines,
    composerRange,
    editor,
    editorThreads,
    expandedRoots,
    markersByLine,
    monacoModule,
    openLine,
    selection,
    t,
  ]);

  const openComposer = useCallback(
    (range: LineRange) => {
      setComposerRange(range);
      // The composer range takes over the highlight; drop the text selection.
      editor.setPosition({ lineNumber: range.startLine, column: 1 });
      // Empty-line gutter selections may already have this cursor position,
      // so Monaco won't emit a selection-change event to clear our range.
      setSelection(undefined);
      setHoveredLine(undefined);
    },
    [editor]
  );

  // Gutter interaction. A press in the gutter selects lines (drag to extend)
  // through the editor's own selection; the drag follows the pointer with
  // the editor's hit test so it keeps working past the gutter and the
  // editor's edge.
  useEffect(() => {
    const GutterTypes = new Set<monaco.editor.MouseTargetType>([
      monacoModule.editor.MouseTargetType.GUTTER_GLYPH_MARGIN,
      monacoModule.editor.MouseTargetType.GUTTER_LINE_NUMBERS,
      monacoModule.editor.MouseTargetType.GUTTER_LINE_DECORATIONS,
    ]);
    const HoverTypes = new Set([
      ...GutterTypes,
      monacoModule.editor.MouseTargetType.CONTENT_TEXT,
      monacoModule.editor.MouseTargetType.CONTENT_EMPTY,
    ]);
    const lineOf = (target: monaco.editor.IMouseTarget | null) =>
      target?.position?.lineNumber ??
      (target?.range ? target.range.startLineNumber : undefined);
    const selectLines = (range: LineRange) => {
      const start = Math.min(range.startLine, range.endLine);
      const end = Math.max(range.startLine, range.endLine);
      const model = editor.getModel();
      const endColumn = model ? model.getLineMaxColumn(end) : 1;
      editor.focus();
      editor.setSelection(new monacoModule.Range(start, 1, end, endColumn));
      // An empty line has an empty Monaco text selection, but a deliberate
      // gutter selection must still offer creation for that whole line.
      setSelection({ startLine: start, endLine: end });
    };

    const onDragMove = (event: MouseEvent) => {
      const current = dragRef.current;
      if (!current) return;
      const line = lineOf(
        editor.getTargetAtClientPoint(event.clientX, event.clientY)
      );
      if (line === undefined || current.endLine === line) return;
      dragRef.current = { startLine: current.startLine, endLine: line };
      selectLines(dragRef.current);
    };
    const finishDrag = () => {
      dragRef.current = undefined;
    };

    // Escape clears the explicit gutter selection while this editor owns focus.
    const node = editor.getDomNode();
    const cancelSelection = (event: KeyboardEvent) => {
      if (
        event.key !== "Escape" ||
        !selection ||
        !(event.target instanceof Node) ||
        !node?.contains(event.target)
      )
        return;
      event.preventDefault();
      event.stopPropagation();
      const current = editor.getSelection();
      if (current) {
        editor.setPosition({
          lineNumber: current.endLineNumber,
          column: current.endColumn,
        });
      }
      dragRef.current = undefined;
      setSelection(undefined);
      setHoveredLine(undefined);
    };
    node?.addEventListener("keydown", cancelSelection, true);

    const subscriptions = [
      editor.onMouseMove((e) => {
        if (dragRef.current) return;
        const line = lineOf(e.target);
        setHoveredLine(
          line !== undefined &&
            HoverTypes.has(e.target.type) &&
            !(
              e.target.type ===
                monacoModule.editor.MouseTargetType.CONTENT_EMPTY &&
              e.target.detail?.isAfterLines
            )
            ? line
            : undefined
        );
      }),
      editor.onMouseLeave(() => {
        if (!dragRef.current) setHoveredLine(undefined);
      }),
      editor.onMouseDown((e) => {
        if (!GutterTypes.has(e.target.type) || !e.event.leftButton) return;
        const line = lineOf(e.target);
        if (line === undefined) return;
        const isGlyph =
          e.target.type ===
          monacoModule.editor.MouseTargetType.GUTTER_GLYPH_MARGIN;
        const isCreationAction =
          e.target.type ===
          monacoModule.editor.MouseTargetType.GUTTER_LINE_DECORATIONS;
        if (isCreationAction && actionLine === line && !composerRange) {
          e.event.preventDefault();
          openComposer(selection ?? { startLine: line, endLine: line });
          return;
        }
        const threads = isGlyph ? markersByLine.get(line) : undefined;
        if (threads) {
          e.event.preventDefault();
          editor.setPosition({ lineNumber: line, column: 1 });
          setSelection(undefined);
          if (openLine === line) {
            openThreads(undefined, []);
          } else {
            openThreads(line, firstToExpand(threads));
          }
          return;
        }
        if (!canCreate) return;
        // Keeps the browser from starting a text selection during the drag.
        e.event.preventDefault();
        dragRef.current = { startLine: line, endLine: line };
        selectLines(dragRef.current);
        setHoveredLine(undefined);
      }),
    ];
    document.addEventListener("mousemove", onDragMove);
    document.addEventListener("mouseup", finishDrag);
    return () => {
      node?.removeEventListener("keydown", cancelSelection, true);
      document.removeEventListener("mousemove", onDragMove);
      document.removeEventListener("mouseup", finishDrag);
      for (const subscription of subscriptions) subscription.dispose();
    };
  }, [
    actionLine,
    canCreate,
    composerRange,
    editor,
    markersByLine,
    monacoModule,
    openComposer,
    openLine,
    openThreads,
    selection,
  ]);

  // Keyboard and context-menu path: comment on the selected lines.
  useEffect(() => {
    if (!canCreate) return;
    const action = editor.addAction({
      id: "bytebase.comment-on-lines",
      label: t("plan.review.thread.comment-on-lines"),
      contextMenuGroupId: "navigation",
      contextMenuOrder: 0,
      run: () => {
        const current = editor.getSelection();
        if (!current) return;
        openComposer(selectionLineRange(current));
      },
    });
    return () => action.dispose();
  }, [canCreate, editor, openComposer, t]);

  useEffect(() => {
    const node = editor.getDomNode();
    if (!node) return;
    node.classList.toggle("bb-thread-creatable", canCreate);
    return () => node.classList.remove("bb-thread-creatable");
  }, [canCreate, editor]);

  const publish = useCallback(
    async (comment: string) => {
      if (!composerRange) return false;
      const created = await actions.publish(
        comment,
        buildWholeLineAnchor({
          spec: spec.id,
          sheetSha256,
          startLine: composerRange.startLine,
          endLine: composerRange.endLine,
        })
      );
      if (!created) return false;
      setComposerRange(undefined);
      openThreads(composerRange.endLine, [created.name]);
      return true;
    },
    [actions, composerRange, openThreads, sheetSha256, spec.id]
  );

  const openEntries: EditorThread[] =
    openLine !== undefined ? (markersByLine.get(openLine) ?? []) : [];

  return (
    <>
      {openLine !== undefined && openEntries.length > 0 && (
        <MonacoViewZone
          afterLineNumber={openLine}
          editor={editor}
          revealKey={expandedRoots.size > 0 ? expandedRoots : undefined}
          revealTarget={expandedThreadRef}
        >
          <div className="py-2 pr-2 sm:pr-4">
            <ThreadStack
              key={openLine}
              expandedRoots={expandedRoots}
              expandedThreadRef={expandedThreadRef}
              issueName={issue.name}
              onCollapse={(rootName) =>
                openThreads(
                  openLine,
                  Array.from(expandedRoots).filter((r) => r !== rootName)
                )
              }
              onExpand={(rootName) => openThreads(openLine, [rootName])}
              replyDrafts={replyDrafts}
              onReplyDraftChange={updateReplyDraft}
              project={project}
              threads={openEntries}
            />
          </div>
        </MonacoViewZone>
      )}
      {composerRange && (
        <MonacoViewZone
          afterLineNumber={composerRange.endLine}
          editor={editor}
          revealKey={`${composerRange.startLine}-${composerRange.endLine}`}
        >
          <div className="py-2 pr-2 sm:pr-4">
            <InlineThreadComposer
              onCancel={() => setComposerRange(undefined)}
              onPublish={publish}
              pending={actions.pending}
              range={composerRange}
            />
          </div>
        </MonacoViewZone>
      )}
    </>
  );
}
