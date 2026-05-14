import { Clock4, EllipsisVertical, Link2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { ReleaseFileDetailPanel } from "@/react/components/release/ReleaseFileDetailPanel";
import { ReleaseFileTable } from "@/react/components/release/ReleaseFileTable";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { useVueState } from "@/react/hooks/useVueState";
import { pushNotification, useProjectV1Store, useReleaseStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { State, VCSType } from "@/types/proto-es/v1/common_pb";
import type { Release_File } from "@/types/proto-es/v1/release_service_pb";
import { setDocumentTitle } from "@/utils";

export function ProjectReleaseDetailPage({
  projectId,
  releaseId,
}: {
  projectId: string;
  releaseId: string;
}) {
  const { t } = useTranslation();
  const releaseStore = useReleaseStore();
  const projectV1Store = useProjectV1Store();

  const projectName = `${projectNamePrefix}${projectId}`;
  const releaseName = `${projectName}/releases/${releaseId}`;

  const release = useVueState(() => releaseStore.getReleaseByName(releaseName));
  const project = useVueState(() =>
    projectV1Store.getProjectByName(projectName)
  );

  useEffect(() => {
    let cancelled = false;
    void projectV1Store.getOrFetchProjectByName(projectName).catch((error) => {
      if (!cancelled) console.error("Failed to fetch project", error);
    });
    void releaseStore.fetchReleaseByName(releaseName).catch((error) => {
      if (!cancelled) console.error("Failed to fetch release", error);
    });
    return () => {
      cancelled = true;
    };
  }, [projectV1Store, releaseStore, projectName, releaseName]);

  useEffect(() => {
    if (project?.title) {
      setDocumentTitle(t("release.releases"), project.title);
    }
  }, [project?.title, t]);

  const [selectedReleaseFile, setSelectedReleaseFile] = useState<
    Release_File | undefined
  >();
  const [archiveOpen, setArchiveOpen] = useState(false);

  const releaseDisplayName = useMemo(() => {
    const parts = release.name.split("/");
    return parts[parts.length - 1] || release.name;
  }, [release.name]);

  const isActive = release.state === State.ACTIVE;
  const isDeleted = release.state === State.DELETED;

  const handleArchive = async () => {
    try {
      await releaseStore.deleteRelease(release.name);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    } finally {
      setArchiveOpen(false);
    }
  };

  const handleRestore = async () => {
    try {
      await releaseStore.undeleteRelease(release.name);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: error instanceof Error ? error.message : String(error),
      });
    }
  };

  return (
    <div className="flex flex-col items-start gap-y-4 p-4 relative">
      {isDeleted && (
        <div className="h-8 w-full text-base font-medium bg-gray-700 text-white flex justify-center items-center">
          {t("common.archived")}
        </div>
      )}

      <div className="w-full flex flex-row items-center justify-between gap-x-4">
        <div className="flex-1 p-0.5 overflow-hidden">
          <h1 className="text-xl font-medium truncate">{releaseDisplayName}</h1>
        </div>
        <div className="flex items-center justify-end gap-x-2">
          {(isActive || isDeleted) && (
            <DropdownMenu>
              <DropdownMenuTrigger
                className="inline-flex items-center justify-center rounded-xs p-1 text-control hover:bg-control-bg focus:outline-hidden"
                aria-label="More actions"
              >
                <EllipsisVertical className="size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                {isActive && (
                  <DropdownMenuItem onClick={() => setArchiveOpen(true)}>
                    {t("common.archive")}
                  </DropdownMenuItem>
                )}
                {isDeleted && (
                  <DropdownMenuItem onClick={handleRestore}>
                    {t("common.restore")}
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>

      <ReleaseBasicInfo
        createTime={
          release.createTime
            ? getTimeForPbTimestampProtoEs(release.createTime) / 1000
            : undefined
        }
        vcsType={release.vcsSource?.vcsType}
        vcsUrl={release.vcsSource?.url}
      />

      <ReleaseFileTable
        files={release.files}
        releaseType={release.type}
        showSelection={false}
        onRowClick={(file) => setSelectedReleaseFile(file)}
      />

      <Sheet
        open={!!selectedReleaseFile}
        onOpenChange={(next) => !next && setSelectedReleaseFile(undefined)}
      >
        <SheetContent width="wide">
          <SheetHeader>
            <SheetTitle>{t("release.file")}</SheetTitle>
          </SheetHeader>
          <SheetBody>
            {selectedReleaseFile && (
              <ReleaseFileDetailPanel releaseFile={selectedReleaseFile} />
            )}
          </SheetBody>
        </SheetContent>
      </Sheet>

      <AlertDialog
        open={archiveOpen}
        onOpenChange={(next) => setArchiveOpen(next)}
      >
        <AlertDialogContent>
          <AlertDialogTitle>{t("common.confirm-archive")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("common.archive-description", { name: releaseDisplayName })}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button variant="outline" onClick={() => setArchiveOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleArchive}>{t("common.confirm")}</Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ReleaseBasicInfo
// ---------------------------------------------------------------------------

function beautifyUrl(url: string): string {
  try {
    const parsed = new URL(url);
    return parsed.pathname.length > 0
      ? parsed.pathname.substring(1)
      : parsed.pathname;
  } catch {
    return url;
  }
}

function ReleaseBasicInfo({
  createTime,
  vcsType,
  vcsUrl,
}: {
  createTime: number | undefined;
  vcsType: VCSType | undefined;
  vcsUrl: string | undefined;
}) {
  const showVcs =
    vcsType !== undefined && vcsType !== VCSType.VCS_TYPE_UNSPECIFIED && vcsUrl;

  return (
    <div className="flex flex-row items-center pl-1 gap-4">
      <div className="flex items-center gap-1">
        <Clock4 className="size-4 text-control-light" />
        {createTime !== undefined && (
          <HumanizeTs ts={createTime} className="text-sm text-control" />
        )}
      </div>
      {showVcs && (
        <div className="flex flex-row items-center gap-1">
          <Link2 className="size-4 text-control-light" />
          <a
            href={vcsUrl}
            target="_blank"
            rel="noreferrer"
            className="text-sm text-accent hover:underline truncate"
          >
            {beautifyUrl(vcsUrl)}
          </a>
        </div>
      )}
    </div>
  );
}
