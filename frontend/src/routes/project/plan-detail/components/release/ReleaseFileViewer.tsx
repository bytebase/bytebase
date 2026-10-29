import { ExternalLink } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ReadonlyMonaco } from "@/components/monaco";
import type { AdviceOption } from "@/components/monaco/types";
import type { ReleaseFileOpenContext } from "@/components/task-run-log/ReleaseSpineView";
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
  const [sheet, setSheet] = useState<{ name: string; value?: SheetMessage }>();

  useEffect(() => {
    if (!sheetName) return;
    let canceled = false;
    void fetchSheet(sheetName, true).then((value) => {
      if (!canceled) setSheet({ name: sheetName, value });
    });
    return () => {
      canceled = true;
    };
  }, [fetchSheet, sheetName]);

  const loaded = sheet?.name === sheetName ? sheet.value : undefined;
  const statement = useMemo(
    () => (loaded ? new TextDecoder().decode(loaded.content) : ""),
    [loaded]
  );
  const context = target?.context;
  const failed = context?.kind === "failed" ? context : undefined;
  const diagnostic =
    failed?.range && statement
      ? diagnosticOf(statement, failed.error, failed.range)
      : undefined;
  const commit = parseRepoCommit(vcsSource);
  const link = target
    ? fileAtCommitUrl(vcsSource, target.file.path)
    : undefined;
  const platform = platformName(vcsSource?.vcsType);
  const databaseLine =
    context?.kind === "applied"
      ? t("release.viewer.applied-by-run")
      : context?.kind === "already-applied"
        ? t("release.viewer.already-applied")
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
                  <SheetTitle className="truncate font-mono text-base">
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
                <div
                  className="whitespace-pre-wrap rounded-xs border border-error/40 bg-error/10 px-3 py-2 text-sm text-error"
                  data-testid="release-file-error"
                >
                  {failed.error}
                </div>
              ) : null}
              {loaded ? (
                <div className="relative">
                  <div className="absolute right-1 top-1">
                    <CopyButton content={statement} />
                  </div>
                  <ReadonlyMonaco
                    className="rounded-sm border text-sm"
                    content={statement}
                    advices={diagnostic ? [diagnostic] : []}
                    max={10000}
                    options={{ wordWrap: sidePanel ? "off" : "on" }}
                    onReady={(_monaco, editor) => {
                      if (diagnostic) {
                        editor.revealLineInCenter(diagnostic.startLineNumber);
                      }
                    }}
                  />
                </div>
              ) : (
                <div className="text-sm text-control-light">
                  {t("common.loading")}
                </div>
              )}
            </SheetBody>
          </>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}
