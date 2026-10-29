import { ExternalLink } from "lucide-react";
import {
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
import { buttonVariants } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import type { Release_VCSSource } from "@/types/proto-es/v1/release_service_pb";
import type { Sheet as SheetMessage } from "@/types/proto-es/v1/sheet_service_pb";
import { fileAtCommitUrl, parseRepoCommit, platformName } from "./vcsSource";

// At this width the viewer is a side panel and the page stays usable beside
// it; below, it is a sheet over a scrim (full width on a phone).
const SIDE_PANEL_MEDIA_QUERY = "(min-width: 1280px)";

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

// A read-only view of one release file (D8): its statement with the failed
// command's error drawn on it, and a link to the file at the commit.
export function ReleaseFileViewer({
  target,
  vcsSource,
  onClose,
}: {
  target?: ReleaseFileViewerTarget;
  vcsSource?: Release_VCSSource;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const sidePanel = useMediaQuery(SIDE_PANEL_MEDIA_QUERY);
  const fetchSheet = useAppStore((state) => state.fetchSheet);
  const sheetName = target?.file.sheet ?? "";
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
  let openingKey = 0;
  if (target) {
    let key = openings.current.get(target);
    if (key === undefined) {
      key = ++openingCount.current;
      openings.current.set(target, key);
    }
    openingKey = key;
  }
  const loaded = sheet?.name === sheetName ? sheet.value : undefined;
  const statement = useMemo(
    () => (loaded ? new TextDecoder().decode(loaded.content) : ""),
    [loaded]
  );
  const context = target?.context;
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
  const editorReady = target !== undefined && readyFor === target;
  const advices = useMemo(
    () => (diagnostic && editorReady ? [diagnostic] : []),
    [diagnostic, editorReady]
  );
  const commit = parseRepoCommit(vcsSource);
  const link = target
    ? fileAtCommitUrl(vcsSource, target.file.path)
    : undefined;
  const platform = platformName(vcsSource?.vcsType);
  const databaseLine =
    context?.kind === "applied"
      ? t("release.viewer.applied-by-run")
      : context?.kind === "already-applied"
        ? context.run
          ? t("release.viewer.already-applied")
          : t("release.viewer.already-applied-before-run")
        : undefined;

  return (
    <Sheet
      open={Boolean(target)}
      modal={!sidePanel}
      disablePointerDismissal={sidePanel}
      onOpenChange={(open) => !open && onClose()}
    >
      <SheetContent
        width={sidePanel ? "split" : "standard"}
        scrim={!sidePanel}
        data-testid="release-file-viewer"
        data-layout={sidePanel ? "side-panel" : "sheet"}
      >
        {target ? (
          <>
            <SheetHeader>
              <div className="flex min-w-0 flex-col gap-1">
                {failed && target.stageTitle ? (
                  <div className="text-sm text-error">
                    {t("release.viewer.failed-on", {
                      stage: target.stageTitle,
                    })}
                  </div>
                ) : null}
                <div className="flex min-w-0 items-center gap-x-2">
                  <span className="shrink-0 tabular-nums text-control-light">
                    {target.file.version}
                  </span>
                  <SheetTitle className="truncate font-mono">
                    {target.file.path}
                  </SheetTitle>
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
                    <span data-testid="release-file-database-line">
                      {databaseLine}
                    </span>
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
            </SheetHeader>
            <SheetBody className="flex flex-col gap-2">
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
                    options={{ wordWrap: sidePanel ? "off" : "on" }}
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
            </SheetBody>
          </>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

type OpenReleaseFileTarget = (
  target: ReleaseFileViewerTarget,
  vcsSource?: Release_VCSSource
) => void;

const ReleaseFileViewerContext = createContext<
  OpenReleaseFileTarget | undefined
>(undefined);

// One viewer for the whole plan page: opening another file, from Changes or
// any task, swaps its content (D8).
export function ReleaseFileViewerProvider({
  children,
}: {
  children: ReactNode;
}) {
  const [opened, setOpened] = useState<{
    target: ReleaseFileViewerTarget;
    vcsSource?: Release_VCSSource;
  }>();
  const open = useCallback<OpenReleaseFileTarget>(
    (target, vcsSource) => setOpened({ target, vcsSource }),
    []
  );
  return (
    <ReleaseFileViewerContext.Provider value={open}>
      {children}
      <ReleaseFileViewer
        target={opened?.target}
        vcsSource={opened?.vcsSource}
        onClose={() => setOpened(undefined)}
      />
    </ReleaseFileViewerContext.Provider>
  );
}

// Undefined outside a plan page: file paths then render as plain text.
export const useOpenReleaseFile = () => useContext(ReleaseFileViewerContext);
