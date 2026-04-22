import dayjs from "dayjs";
import { ExternalLink, Loader2, Package } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "@/react/lib/utils";
import { useReleaseByName } from "@/store/modules/release";
import { getDateForPbTimestampProtoEs, isValidReleaseName } from "@/types";
import { State, VCSType } from "@/types/proto-es/v1/common_pb";
import { Release_Type } from "@/types/proto-es/v1/release_service_pb";

export function DeployReleaseInfoCard({
  className,
  compact = true,
  releaseName,
}: {
  className?: string;
  compact?: boolean;
  releaseName: string;
}) {
  const { t } = useTranslation();
  const { release, ready } = useReleaseByName(releaseName);
  const releaseValue = release.value;
  const isDeleted = releaseValue?.state === State.DELETED;
  const maxDisplayedFiles = compact ? 4 : 6;
  const releaseTitle = useMemo(() => {
    const name = releaseValue?.name || releaseName;
    const parts = name.split("/");
    return parts[parts.length - 1] || name;
  }, [releaseName, releaseValue?.name]);
  const displayedFiles = releaseValue?.files?.slice(0, maxDisplayedFiles) ?? [];

  if (!ready.value) {
    return (
      <div className={cn("flex flex-col gap-y-2", className)}>
        <div className="rounded-md border bg-gray-50 p-2">
          <div className="flex items-center gap-x-2">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span className="text-xs text-gray-600">{t("common.loading")}</span>
          </div>
        </div>
      </div>
    );
  }

  if (!isValidReleaseName(releaseValue?.name ?? "")) {
    return (
      <div className={cn("flex flex-col gap-y-2", className)}>
        <div className="rounded-md border bg-red-50 p-2">
          <div className="text-xs text-red-600">{t("release.not-found")}</div>
        </div>
      </div>
    );
  }

  return (
    <div className={cn("flex flex-col gap-y-2", className)}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-x-2 text-sm font-medium">
          <Package className="h-4 w-4" />
          <span
            className={cn(
              !compact && "text-base",
              isDeleted && "text-control-light line-through"
            )}
          >
            {releaseTitle}
          </span>
        </div>
        <a
          className="inline-flex items-center gap-x-1 text-xs text-accent hover:underline"
          href={`/${releaseValue.name}`}
          rel="noreferrer"
          target="_blank"
        >
          {t("common.view")}
          <ExternalLink className="h-3 w-3" />
        </a>
      </div>

      <div
        className={cn(
          "rounded-md border bg-gray-50",
          compact ? "px-3 py-2" : "px-4 py-3"
        )}
      >
        <div className={cn("flex flex-col", compact ? "gap-y-2" : "gap-y-3")}>
          <div className="flex items-start justify-between">
            <div>
              <h3 className="text-sm font-medium text-gray-900">
                {releaseTitle}
              </h3>
            </div>
          </div>

          {releaseValue.files && releaseValue.files.length > 0 && (
            <div className="flex flex-col gap-y-1">
              <div className="flex items-center justify-between">
                <h4 className="text-xs font-medium text-gray-700">
                  {t("release.files")} ({releaseValue.files.length})
                </h4>
                {releaseValue.files.length > maxDisplayedFiles && (
                  <a
                    className="text-xs text-accent hover:underline"
                    href={`/${releaseValue.name}`}
                    rel="noreferrer"
                    target="_blank"
                  >
                    {t("release.view-all-files")}
                  </a>
                )}
              </div>
              <div
                className={cn(
                  "grid w-full gap-1",
                  compact
                    ? "grid-cols-1 sm:grid-cols-2"
                    : "grid-cols-1 gap-2 sm:grid-cols-2 md:grid-cols-3"
                )}
              >
                {displayedFiles.map((file) => (
                  <div
                    key={file.path}
                    className={cn(
                      "flex w-full items-center justify-between rounded-sm bg-white text-xs",
                      compact ? "p-1.5" : "p-2"
                    )}
                  >
                    <div className="mr-2 min-w-0 flex-1">
                      <div className="truncate font-medium">{file.path}</div>
                      <div className="text-gray-500">{file.version}</div>
                    </div>
                    <div
                      className={cn(
                        "inline-flex shrink-0 items-center rounded-sm bg-blue-100 text-xs text-blue-800",
                        compact ? "px-1 py-0.5" : "px-1.5 py-0.5"
                      )}
                    >
                      {getReleaseFileTypeText(releaseValue.type, t)}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {releaseValue.vcsSource && (
            <div className="flex flex-col gap-y-0.5">
              <h4 className="text-xs font-medium text-gray-700">
                {t("release.vcs-source")}
              </h4>
              <div className="text-xs">
                <span className="text-gray-500">
                  {getVCSTypeText(releaseValue.vcsSource.vcsType, t)}:
                </span>
                {releaseValue.vcsSource.url && (
                  <a
                    className="ml-1 text-blue-600 hover:text-blue-800"
                    href={releaseValue.vcsSource.url}
                    rel="noreferrer"
                    target="_blank"
                  >
                    {releaseValue.vcsSource.url}
                  </a>
                )}
              </div>
            </div>
          )}

          <div className="text-xs text-gray-500">
            {dayjs(
              getDateForPbTimestampProtoEs(releaseValue.createTime)
            ).format("YYYY-MM-DD HH:mm:ss")}
          </div>
        </div>
      </div>
    </div>
  );
}

function getReleaseFileTypeText(
  fileType: Release_Type,
  t: (key: string, options?: Record<string, unknown>) => string
) {
  switch (fileType) {
    case Release_Type.VERSIONED:
      return t("release.file-type.versioned");
    case Release_Type.DECLARATIVE:
      return t("release.file-type.declarative");
    default:
      return t("release.file-type.unspecified");
  }
}

function getVCSTypeText(
  vcsType: VCSType,
  t: (key: string, options?: Record<string, unknown>) => string
) {
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
