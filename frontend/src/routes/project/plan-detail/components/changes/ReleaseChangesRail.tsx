import { ChevronRight } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { useKeepEndInPlace } from "@/hooks/useKeepEndInPlace";
import { cn } from "@/lib/utils";
import {
  type Release,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";
import { useOpenReleaseFile } from "../release/ReleaseFileViewer";
import {
  CHANGES_ROW_LIMIT,
  groupReleaseChanges,
  type ReleaseChangeFile,
  type ReleaseChangeGroup,
} from "./releaseChanges";

// The release's files with no per-database status (D1, D2): status is a fact
// about one database, and lives on each task.
export function ReleaseChangesRail({
  className,
  release,
}: {
  className?: string;
  release: Release;
}) {
  const { t } = useTranslation();
  const openViewer = useOpenReleaseFile();
  // Opened from Changes there is no database, so no context.
  const open = openViewer
    ? (file: ReleaseChangeFile) => openViewer({ file }, release.vcsSource)
    : undefined;
  const files = useMemo(
    () =>
      release.files.map((file) => ({
        version: file.version,
        path: file.path,
        sheet: file.sheet,
      })),
    [release]
  );
  const groups = useMemo(() => groupReleaseChanges(files), [files]);

  if (release.type === Release_Type.DECLARATIVE) {
    const file = files[0];
    return (
      <div
        className={cn("flex items-center gap-x-2 text-sm", className)}
        data-testid="release-changes"
      >
        {file ? (
          <>
            <span className="text-control-light">
              {t("release.changes.schema")}
            </span>
            <span className="truncate font-mono text-control">{file.path}</span>
            <span className="shrink-0 font-mono tabular-nums text-control-light">
              {file.version}
            </span>
            {open ? (
              <Button size="xs" appearance="outline" onClick={() => open(file)}>
                {t("common.open")}
              </Button>
            ) : null}
          </>
        ) : null}
      </div>
    );
  }

  return (
    <div
      className={cn("flex flex-col gap-2 text-sm", className)}
      data-testid="release-changes"
    >
      <div className="text-control-light">
        {t("release.changes.manifest", {
          count: files.length,
          version: groups[0]?.files[0]?.version ?? "",
        })}
      </div>
      <div className="flex flex-col rounded-sm border border-block-border">
        {groups.map((group, index) => (
          <ChangeGroup
            key={group.key || "all"}
            group={group}
            defaultOpen={index === 0}
            onOpen={open}
          />
        ))}
      </div>
    </div>
  );
}

function ChangeGroup({
  group,
  defaultOpen,
  onOpen,
}: {
  group: ReleaseChangeGroup;
  defaultOpen: boolean;
  // Absent outside a plan page; the path is then plain text.
  onOpen?: (file: ReleaseChangeFile) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(defaultOpen);
  const [shown, setShown] = useState(CHANGES_ROW_LIMIT);
  const listRef = useRef<HTMLDivElement>(null);
  const keepEndInPlace = useKeepEndInPlace(listRef);
  const remaining = Math.max(group.files.length - shown, 0);
  const extended =
    shown > CHANGES_ROW_LIMIT && group.files.length > CHANGES_ROW_LIMIT;
  const rows = (
    <div ref={listRef} className="flex flex-col">
      {group.files.slice(0, shown).map((file) => (
        <div
          key={file.version}
          className="flex min-w-0 items-center gap-x-3 px-3 py-1"
          data-testid="release-change-row"
          data-version={file.version}
        >
          <span className="shrink-0 font-mono text-xs tabular-nums text-control-light">
            {file.version}
          </span>
          {onOpen ? (
            <Button
              type="button"
              appearance="link"
              size="sm"
              className="h-auto min-w-0 truncate p-0 font-mono text-xs"
              onClick={() => onOpen(file)}
            >
              {file.path}
            </Button>
          ) : (
            <span className="min-w-0 truncate font-mono text-xs">
              {file.path}
            </span>
          )}
        </div>
      ))}
      {remaining > 0 || extended ? (
        <div
          className="flex items-center gap-x-4 px-3 py-1"
          data-testid="release-change-load"
        >
          {remaining > 0 ? (
            <Button
              type="button"
              appearance="link"
              size="sm"
              className="h-auto p-0 text-xs"
              onClick={() => setShown((n) => n + CHANGES_ROW_LIMIT)}
            >
              {t("release.changes.load-more", {
                count: Math.min(remaining, CHANGES_ROW_LIMIT),
              })}
            </Button>
          ) : null}
          {remaining > CHANGES_ROW_LIMIT ? (
            <Button
              type="button"
              appearance="link"
              size="sm"
              className="h-auto p-0 text-xs"
              onClick={() => setShown(group.files.length)}
            >
              {t("release.changes.load-all", { count: remaining })}
            </Button>
          ) : null}
          {extended ? (
            <Button
              type="button"
              appearance="link"
              size="sm"
              className="h-auto p-0 text-xs"
              onClick={() => keepEndInPlace(() => setShown(CHANGES_ROW_LIMIT))}
            >
              {t("release.changes.show-fewer")}
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
  // The one flat list has no header.
  if (!group.key) return rows;
  return (
    <Collapsible
      open={open}
      onOpenChange={setOpen}
      className="border-block-border border-b last:border-b-0"
    >
      <CollapsibleTrigger
        className="bg-background px-3 py-1.5 hover:bg-control-bg"
        data-testid="release-change-group"
        data-group={group.key}
      >
        <ChevronRight
          className={cn(
            "size-3.5 shrink-0 text-control-placeholder transition-transform",
            open && "rotate-90"
          )}
        />
        <span className="tabular-nums text-control">{group.key}</span>
        <span className="text-control-placeholder">{group.files.length}</span>
      </CollapsibleTrigger>
      <CollapsiblePanel>{rows}</CollapsiblePanel>
    </Collapsible>
  );
}
