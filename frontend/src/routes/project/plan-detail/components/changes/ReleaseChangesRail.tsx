import { ChevronDown, ChevronRight } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  ReleaseFileViewer,
  type ReleaseFileViewerTarget,
} from "@/components/release/ReleaseFileViewer";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  type Release,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";
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
  const [viewerTarget, setViewerTarget] = useState<ReleaseFileViewerTarget>();
  const open = (file: ReleaseChangeFile) => setViewerTarget({ file });
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

  const viewer = (
    <ReleaseFileViewer
      target={viewerTarget}
      vcsSource={release.vcsSource}
      onClose={() => setViewerTarget(undefined)}
    />
  );

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
            <Button size="xs" appearance="outline" onClick={() => open(file)}>
              {t("common.open")}
            </Button>
          </>
        ) : null}
        {viewer}
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
      {viewer}
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
  onOpen: (file: ReleaseChangeFile) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(defaultOpen);
  const [shown, setShown] = useState(CHANGES_ROW_LIMIT);
  const remaining = group.files.length - shown;
  const rows = (
    <div className="flex flex-col">
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
          <Button
            type="button"
            appearance="link"
            size="sm"
            className="h-auto min-w-0 truncate p-0 font-mono text-xs"
            onClick={() => onOpen(file)}
          >
            {file.path}
          </Button>
        </div>
      ))}
      {remaining > 0 ? (
        <div className="px-3 py-1">
          <Button
            type="button"
            appearance="link"
            size="sm"
            className="h-auto p-0 text-xs"
            onClick={() => setShown((n) => n + CHANGES_ROW_LIMIT)}
          >
            {t("release.changes.n-more", { count: remaining })}
          </Button>
        </div>
      ) : null}
    </div>
  );
  // The one flat list has no header.
  if (!group.key) return rows;
  return (
    <div className="border-block-border border-b last:border-b-0">
      <Button
        type="button"
        appearance="secondary"
        size="sm"
        aria-expanded={open}
        className="w-full justify-start rounded-none bg-background px-3 hover:bg-control-bg"
        data-testid="release-change-group"
        data-group={group.key}
        onClick={() => setOpen((value) => !value)}
      >
        {open ? (
          <ChevronDown className="size-3.5 shrink-0 text-control-placeholder" />
        ) : (
          <ChevronRight className="size-3.5 shrink-0 text-control-placeholder" />
        )}
        <span className="tabular-nums text-control">{group.key}</span>
        <span className="text-control-placeholder">{group.files.length}</span>
      </Button>
      {open ? rows : null}
    </div>
  );
}
