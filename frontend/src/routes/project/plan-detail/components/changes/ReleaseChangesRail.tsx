import { ChevronRight, FileText } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
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
import { useKeepEndInPlace } from "./useKeepEndInPlace";

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
        <ChangeRow key={file.version} file={file} onOpen={onOpen} />
      ))}
      {remaining > 0 || extended ? (
        <div
          className="flex items-center gap-x-4 px-3 py-2"
          data-testid="release-change-load"
        >
          {remaining > 0 ? (
            <Button
              type="button"
              appearance="link"
              size="sm"
              className="h-auto p-0"
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
              className="h-auto p-0"
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
              className="h-auto p-0"
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

const ROW_CLASS =
  "group flex w-full min-w-0 items-center gap-x-2 border-block-border border-b px-3 py-2 text-left last:border-b-0";

// A row of a repository's file list: the name leads, the folder and the
// version recede, and the whole row opens the viewer.
function ChangeRow({
  file,
  onOpen,
}: {
  file: ReleaseChangeFile;
  onOpen?: (file: ReleaseChangeFile) => void;
}) {
  const slash = file.path.lastIndexOf("/");
  const name = file.path ? file.path.slice(slash + 1) : file.version;
  const folder = slash > 0 ? file.path.slice(0, slash) : "";
  const content = (
    <>
      <FileText className="size-4 shrink-0 text-control-placeholder" />
      <span
        className={cn(
          "min-w-0 shrink truncate text-main",
          onOpen && "group-hover:text-accent group-hover:underline"
        )}
        data-testid="release-change-name"
      >
        {name}
      </span>
      {folder ? (
        <span
          className="min-w-0 shrink-[2] truncate text-control-light"
          data-testid="release-change-folder"
        >
          {folder}
        </span>
      ) : null}
      <span
        className="ml-auto shrink-0 pl-4 tabular-nums text-control-light"
        data-testid="release-change-version"
      >
        {file.version}
      </span>
    </>
  );
  return onOpen ? (
    <Button
      type="button"
      appearance="secondary"
      className={cn(
        ROW_CLASS,
        "h-auto justify-start rounded-none font-normal focus-visible:ring-inset focus-visible:ring-offset-0"
      )}
      data-testid="release-change-row"
      data-version={file.version}
      onClick={() => onOpen(file)}
    >
      {content}
    </Button>
  ) : (
    <div
      className={ROW_CLASS}
      data-testid="release-change-row"
      data-version={file.version}
    >
      {content}
    </div>
  );
}
