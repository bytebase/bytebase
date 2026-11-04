import { ExternalLink, X } from "lucide-react";
import {
  type CSSProperties,
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { ReadonlyMonaco } from "@/components/monaco";
import type { AdviceOption } from "@/components/monaco/types";
import type { ReleaseFileOpenContext } from "@/components/task-run-log/ReleaseSpineView";
import { Alert } from "@/components/ui/alert";
import { Button, buttonVariants } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useEscapeKey } from "@/hooks/useEscapeKey";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import type { Release_VCSSource } from "@/types/proto-es/v1/release_service_pb";
import type { Sheet as SheetMessage } from "@/types/proto-es/v1/sheet_service_pb";
import { fileAtCommitUrl, parseRepoCommit, platformName } from "./vcsSource";

// From this page width the viewer docks as a column on the page's right;
// narrower, it is a sheet over a scrim (D8).
export const VIEWER_DOCK_MIN_WIDTH = 1280;
// On a phone the statement's lines wrap.
const PHONE_MEDIA_QUERY = "(max-width: 639px)";

export interface ReleaseFileViewerTarget {
  file: { version: string; path: string; sheet: string };
  // Where it was opened from; absent from Changes, which has no database.
  context?: ReleaseFileOpenContext;
  // The stage of the failed task, for the header.
  stageTitle?: string;
}

// The 1-based line and column of a byte offset into UTF-8 `bytes`.
const positionOfByte = (bytes: Uint8Array, offset: number) => {
  const before = new TextDecoder().decode(bytes.subarray(0, offset));
  const lines = before.split("\n");
  return { line: lines.length, column: (lines.at(-1)?.length ?? 0) + 1 };
};

// The error as an editor diagnostic on the failed command, from the command's
// byte range into the sheet (CommandExecute.range).
export const diagnosticOf = (
  statement: string,
  error: string,
  range: { start: number; end: number }
): AdviceOption | undefined => {
  const bytes = new TextEncoder().encode(statement);
  if (range.end <= range.start || range.end > bytes.length) return undefined;
  const start = positionOfByte(bytes, range.start);
  const end = positionOfByte(bytes, range.end);
  return {
    severity: "ERROR",
    message: error,
    startLineNumber: start.line,
    startColumn: start.column,
    endLineNumber: end.line,
    endColumn: end.column,
  };
};

type ViewerLayout = "sheet" | "column";

// A read-only view of one release file (D8): its statement with the failed
// command's error drawn on it, and a link to the file at the commit. The same
// content fills the sheet and the docked column.
function ReleaseFileViewerContent({
  target,
  vcsSource,
  layout,
  onClose,
}: {
  target: ReleaseFileViewerTarget;
  vcsSource?: Release_VCSSource;
  layout: ViewerLayout;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const phone = useMediaQuery(PHONE_MEDIA_QUERY);
  const fetchSheet = useAppStore((state) => state.fetchSheet);
  const sheetName = target.file.sheet;
  // `value` is undefined once the fetch has settled without a sheet.
  const [sheet, setSheet] = useState<{ name: string; value?: SheetMessage }>();

  useEffect(() => {
    if (!sheetName) return;
    let canceled = false;
    void fetchSheet(sheetName, true)
      .catch(() => undefined)
      .then((value) => {
        if (!canceled) setSheet({ name: sheetName, value });
      });
    return () => {
      canceled = true;
    };
  }, [fetchSheet, sheetName]);

  const settled = !sheetName || sheet?.name === sheetName;
  // A new key for every opening, even of the same file.
  const openings = useRef(new WeakMap<ReleaseFileViewerTarget, number>());
  const openingCount = useRef(0);
  let openingKey = openings.current.get(target);
  if (openingKey === undefined) {
    openingKey = ++openingCount.current;
    openings.current.set(target, openingKey);
  }
  const loaded = sheet?.name === sheetName ? sheet.value : undefined;
  const statement = useMemo(
    () => (loaded ? new TextDecoder().decode(loaded.content) : ""),
    [loaded]
  );
  const context = target.context;
  const failed = context?.kind === "failed" ? context : undefined;
  const failedError = failed?.error ?? "";
  const rangeStart = failed?.range?.start;
  const rangeEnd = failed?.range?.end;
  const diagnostic = useMemo(
    () =>
      rangeStart !== undefined && rangeEnd !== undefined && statement
        ? diagnosticOf(statement, failedError, {
            start: rangeStart,
            end: rangeEnd,
          })
        : undefined,
    [statement, failedError, rangeStart, rangeEnd]
  );
  // The editor draws advices only when the prop changes after it exists, and
  // reveals a line only when told. So each opening mounts its own editor, and
  // the diagnostic is handed over once that editor is ready.
  const [readyFor, setReadyFor] = useState<ReleaseFileViewerTarget>();
  const editorReady = readyFor === target;
  const advices = useMemo(
    () => (diagnostic && editorReady ? [diagnostic] : []),
    [diagnostic, editorReady]
  );
  const commit = parseRepoCommit(vcsSource);
  const link = fileAtCommitUrl(vcsSource, target.file.path);
  const platform = platformName(vcsSource?.vcsType);
  const databaseLine =
    context?.kind === "applied"
      ? t("release.viewer.applied-by-run")
      : context?.kind === "already-applied"
        ? context.run
          ? t("release.viewer.already-applied")
          : t("release.viewer.already-applied-before-run")
        : undefined;

  const title =
    layout === "sheet" ? (
      <SheetTitle className="shrink-0 tabular-nums">
        {target.file.version}
      </SheetTitle>
    ) : (
      <h2 className="shrink-0 text-lg font-semibold tabular-nums text-main">
        {target.file.version}
      </h2>
    );
  const header = (
    <div className="flex min-w-0 flex-col gap-1">
      {failed && target.stageTitle ? (
        <div className="text-sm text-error">
          {t("release.viewer.failed-on", { stage: target.stageTitle })}
        </div>
      ) : null}
      <div className="flex min-w-0 items-center gap-x-2">
        {title}
        <span
          className="min-w-0 truncate font-mono text-sm text-control"
          data-testid="release-file-path"
        >
          {target.file.path}
        </span>
        <CopyButton content={target.file.path} />
      </div>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-control-light">
        {loaded ? (
          <span>
            {t("release.viewer.size", {
              size: Number(loaded.contentSize).toLocaleString(),
            })}
          </span>
        ) : null}
        {databaseLine ? (
          <span data-testid="release-file-database-line">{databaseLine}</span>
        ) : null}
        {link && commit && platform ? (
          <a
            href={link}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              buttonVariants({
                size: "xs",
                appearance: failed ? "solid" : "outline",
              })
            )}
          >
            <ExternalLink className="size-3" />
            {t("release.viewer.open-at-commit", {
              platform,
              sha: commit.sha.slice(0, 7),
            })}
          </a>
        ) : null}
      </div>
    </div>
  );
  const body = (
    <>
      {failed && !diagnostic && failed.error ? (
        <Alert variant="error" data-testid="release-file-error">
          <span className="whitespace-pre-wrap">{failed.error}</span>
        </Alert>
      ) : null}
      {loaded ? (
        <div className="relative">
          <div className="absolute right-1 top-1">
            <CopyButton content={statement} />
          </div>
          <ReadonlyMonaco
            key={openingKey}
            className="rounded-sm border text-sm"
            content={statement}
            advices={advices}
            max={10000}
            options={{ wordWrap: phone ? "on" : "off" }}
            onReady={(_monaco, editor) => {
              setReadyFor(target);
              if (diagnostic) {
                editor.revealLineInCenter(diagnostic.startLineNumber);
              }
            }}
          />
        </div>
      ) : (
        <div
          className="text-sm text-control-light"
          data-testid="release-file-statement-state"
        >
          {settled
            ? t("release.viewer.statement-unavailable")
            : t("common.loading")}
        </div>
      )}
    </>
  );

  if (layout === "sheet") {
    return (
      <>
        <SheetHeader>{header}</SheetHeader>
        <SheetBody className="flex flex-col gap-2">{body}</SheetBody>
      </>
    );
  }
  return (
    <>
      <div className="flex shrink-0 items-start gap-x-2 border-b px-4 py-3">
        <div className="min-w-0 flex-1">{header}</div>
        <Button size="xs" appearance="secondary" onClick={onClose}>
          <X className="size-4" />
          {t("common.close")}
        </Button>
      </div>
      <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto p-4">
        {body}
      </div>
    </>
  );
}

// The viewer as a sheet over a scrim, on a page narrower than
// VIEWER_DOCK_MIN_WIDTH.
export function ReleaseFileViewer({
  target,
  vcsSource,
  onClose,
}: {
  target?: ReleaseFileViewerTarget;
  vcsSource?: Release_VCSSource;
  onClose: () => void;
}) {
  // The last opened file and its commit stay rendered through the sheet's
  // exit animation; the portal unmounts them once the sheet has slid away.
  const lastOpenedRef = useRef({ target, vcsSource });
  if (target) lastOpenedRef.current = { target, vcsSource };
  const shown = lastOpenedRef.current;
  return (
    <Sheet open={Boolean(target)} onOpenChange={(open) => !open && onClose()}>
      <SheetContent
        width="standard"
        data-testid="release-file-viewer"
        data-layout="sheet"
      >
        {shown.target ? (
          <ReleaseFileViewerContent
            target={shown.target}
            vcsSource={shown.vcsSource}
            layout="sheet"
            onClose={onClose}
          />
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

type OpenReleaseFileTarget = (
  target: ReleaseFileViewerTarget,
  vcsSource?: Release_VCSSource
) => void;

interface OpenedFile {
  target: ReleaseFileViewerTarget;
  vcsSource?: Release_VCSSource;
}

const ReleaseFileViewerContext = createContext<
  OpenReleaseFileTarget | undefined
>(undefined);
const OpenedFileContext = createContext<
  { opened?: OpenedFile; close: () => void; docked: boolean } | undefined
>(undefined);

// One viewer for the whole plan page: opening another file, from Changes or
// any task, swaps its content (D8). When `docked`, the page renders it as
// ReleaseFileViewerColumn; otherwise it is a sheet.
export function ReleaseFileViewerProvider({
  children,
  docked = false,
}: {
  children: ReactNode;
  docked?: boolean;
}) {
  const [opened, setOpened] = useState<OpenedFile>();
  const open = useCallback<OpenReleaseFileTarget>(
    (target, vcsSource) => setOpened({ target, vcsSource }),
    []
  );
  const close = useCallback(() => setOpened(undefined), []);
  const state = useMemo(
    () => ({ opened, close, docked }),
    [opened, close, docked]
  );
  return (
    <ReleaseFileViewerContext.Provider value={open}>
      <OpenedFileContext.Provider value={state}>
        {children}
        {docked ? null : (
          <ReleaseFileViewer
            target={opened?.target}
            vcsSource={opened?.vcsSource}
            onClose={close}
          />
        )}
      </OpenedFileContext.Provider>
    </ReleaseFileViewerContext.Provider>
  );
}

// Wraps the plan and, while a file is open on a docked page, splits it with
// the viewer column on the right half (D8). The wrapper is the same element
// open or closed, so the plan never remounts and keeps its state.
export function ReleaseFileViewerDock({
  columnStyle,
  children,
}: {
  columnStyle?: CSSProperties;
  children: ReactNode;
}) {
  const state = useContext(OpenedFileContext);
  const split = Boolean(state?.docked && state.opened);
  return (
    <div
      className={cn("min-w-0 flex-1", split ? "grid" : "flex flex-col")}
      style={
        split
          ? { gridTemplateColumns: "minmax(0, 1fr) minmax(0, 50%)" }
          : undefined
      }
      data-testid="release-file-viewer-dock"
    >
      {children}
      <ReleaseFileViewerColumn style={columnStyle} />
    </div>
  );
}

// The viewer docked on the page's right; `style` places it under the page's
// sticky title row.
export function ReleaseFileViewerColumn({ style }: { style?: CSSProperties }) {
  const state = useContext(OpenedFileContext);
  // Not docked, the provider renders the sheet instead.
  const opened = state?.docked ? state.opened : undefined;
  const close = state?.close ?? noop;
  useEscapeKey(Boolean(opened), close);
  if (!opened) return null;
  return (
    <aside
      className="sticky flex min-w-0 flex-col self-start border-l bg-background"
      style={style}
      data-testid="release-file-viewer"
      data-layout="side-panel"
    >
      <ReleaseFileViewerContent
        target={opened.target}
        vcsSource={opened.vcsSource}
        layout="column"
        onClose={close}
      />
    </aside>
  );
}

const noop = () => {};

// Undefined outside a plan page: file paths then render as plain text.
export const useOpenReleaseFile = () => useContext(ReleaseFileViewerContext);
