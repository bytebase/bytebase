import dayjs from "dayjs";
import type { TFunction } from "i18next";
import { ExternalLink, Loader2, Package } from "lucide-react";
import type { ReactNode } from "react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/react/lib/utils";
import { State, VCSType } from "@/types/proto-es/v1/common_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import { Release_Type } from "@/types/proto-es/v1/release_service_pb";
import { isValidReleaseName } from "@/types/release";
import { getDateForPbTimestampProtoEs } from "@/types/timestamp";

const MAX_DISPLAYED_RELEASE_FILES = 4;

export function ReleaseInfoCard({
  className,
  isLoading = false,
  release,
  releaseName,
}: Readonly<{
  className?: string;
  isLoading?: boolean;
  release?: Release;
  releaseName: string;
}>) {
  const { t } = useTranslation();
  // useReleaseByName returns a sentinel unknownRelease() when the release
  // can't be found, so `release` is truthy even on a miss. Treat it as
  // missing when the name doesn't parse as a real release resource name.
  const effectiveRelease =
    release && isValidReleaseName(release.name) ? release : undefined;
  const releaseTitle = useMemo(() => {
    const name = effectiveRelease?.name || releaseName;
    const parts = name.split("/");
    return parts[parts.length - 1] || name;
  }, [effectiveRelease?.name, releaseName]);
  const isDeleted = effectiveRelease?.state === State.DELETED;

  let body: ReactNode;
  if (isLoading) {
    body = <LoadingBlock />;
  } else if (effectiveRelease) {
    body = <ReleaseBlock release={effectiveRelease} />;
  } else {
    body = <NotFoundBlock />;
  }

  return (
    <div className={cn("flex flex-col gap-y-2", className)}>
      <div className="flex items-center justify-between gap-x-2">
        <div className="flex items-center gap-x-1 text-base font-medium">
          <Package className="h-4 w-4" />
          <span className={cn(isDeleted && "text-control-light line-through")}>
            {releaseTitle}
          </span>
        </div>
        {effectiveRelease && (
          <a
            className="inline-flex items-center gap-x-1 text-sm text-accent hover:underline"
            href={`/${effectiveRelease.name}`}
            rel="noreferrer"
            target="_blank"
          >
            <span>{t("common.view")}</span>
            <ExternalLink className="h-4 w-4" />
          </a>
        )}
      </div>
      {body}
    </div>
  );
}

function LoadingBlock() {
  const { t } = useTranslation();
  return (
    <div className="rounded-md border border-control-border bg-gray-50 px-4 py-3 text-sm text-control-light">
      <div className="flex items-center gap-x-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>{t("common.loading")}</span>
      </div>
    </div>
  );
}

function NotFoundBlock() {
  const { t } = useTranslation();
  return (
    <div className="rounded-md border border-error/30 bg-error/5 px-4 py-3 text-sm text-error">
      {t("release.not-found")}
    </div>
  );
}

function ReleaseBlock({ release }: Readonly<{ release: Release }>) {
  const { t } = useTranslation();
  const displayedFiles = release.files.slice(0, MAX_DISPLAYED_RELEASE_FILES);
  const createdTime = release.createTime
    ? dayjs(getDateForPbTimestampProtoEs(release.createTime)).format(
        "YYYY-MM-DD HH:mm:ss"
      )
    : undefined;

  return (
    <div className="rounded-md border border-control-border bg-gray-50 px-4 py-3">
      <div className="flex flex-col gap-y-3">
        {release.files.length > 0 && (
          <div className="flex flex-col gap-y-2">
            <div className="flex items-center justify-between">
              <div className="text-sm font-medium text-control">
                {t("release.files")} ({release.files.length})
              </div>
              {release.files.length > MAX_DISPLAYED_RELEASE_FILES && (
                <a
                  className="text-sm text-accent hover:underline"
                  href={`/${release.name}`}
                  rel="noreferrer"
                  target="_blank"
                >
                  {t("release.view-all-files")}
                </a>
              )}
            </div>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 md:grid-cols-3">
              {displayedFiles.map((file) => (
                <div
                  key={file.path}
                  className="flex items-center justify-between rounded-sm bg-white p-2 text-xs"
                >
                  <div className="mr-2 min-w-0 flex-1">
                    <div className="truncate font-medium">{file.path}</div>
                    <div className="text-control-light">{file.version}</div>
                  </div>
                  <div className="shrink-0 rounded-sm bg-blue-100 px-1.5 py-0.5 text-xs text-blue-800">
                    {getReleaseFileTypeText(release.type, t)}
                  </div>
                </div>
              ))}
              {release.files.length > MAX_DISPLAYED_RELEASE_FILES && (
                <div className="flex items-center justify-center rounded-sm border border-dashed border-control-border bg-white p-2 text-xs text-control-light">
                  {t("release.and-n-more-files", {
                    count: release.files.length - MAX_DISPLAYED_RELEASE_FILES,
                  })}
                </div>
              )}
            </div>
          </div>
        )}

        {release.vcsSource && (
          <div className="flex flex-col gap-y-1">
            <div className="text-sm font-medium text-control">
              {t("release.vcs-source")}
            </div>
            <div className="text-xs">
              <span className="text-control-light">
                {getVCSTypeText(release.vcsSource.vcsType, t)}:
              </span>
              {release.vcsSource.url && (
                <a
                  className="ml-1 text-accent hover:underline"
                  href={release.vcsSource.url}
                  rel="noreferrer"
                  target="_blank"
                >
                  {release.vcsSource.url}
                </a>
              )}
            </div>
          </div>
        )}

        {createdTime && (
          <div className="text-xs text-control-light">{createdTime}</div>
        )}
      </div>
    </div>
  );
}

function getReleaseFileTypeText(fileType: Release_Type, t: TFunction) {
  switch (fileType) {
    case Release_Type.VERSIONED:
      return t("release.file-type.versioned");
    case Release_Type.DECLARATIVE:
      return t("release.file-type.declarative");
    default:
      return t("release.file-type.unspecified");
  }
}

function getVCSTypeText(vcsType: VCSType, t: TFunction) {
  switch (vcsType) {
    case VCSType.GITHUB:
      return "GitHub";
    case VCSType.GITLAB:
      return "GitLab";
    case VCSType.BITBUCKET:
      return "Bitbucket";
    case VCSType.AZURE_DEVOPS:
      return "Azure DevOps";
    default:
      return t("release.vcs-type.unspecified");
  }
}
