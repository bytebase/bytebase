import { Check } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/router";
import { getDateForPbTimestampProtoEs } from "@/types";
import {
  type Changelog,
  Changelog_Status,
} from "@/types/proto-es/v1/database_service_pb";
import { humanizeDate } from "@/utils";
import { changelogLink } from "@/utils/v1/changelog";

function ChangelogStatusIcon({ status }: { status: Changelog_Status }) {
  if (status === Changelog_Status.PENDING) {
    return (
      <span className="flex h-5 w-5 items-center justify-center rounded-full border-2 border-info bg-white text-info">
        <span
          className="h-2 w-2 rounded-full bg-info"
          style={{
            animation: "pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite",
          }}
        />
      </span>
    );
  }
  if (status === Changelog_Status.DONE) {
    return (
      <span className="flex h-5 w-5 items-center justify-center rounded-full bg-success text-white">
        <Check className="h-4 w-4" />
      </span>
    );
  }
  if (status === Changelog_Status.FAILED) {
    return (
      <span className="flex h-5 w-5 items-center justify-center rounded-full bg-error text-white">
        <span className="text-base font-normal">!</span>
      </span>
    );
  }
  return null;
}

export function DatabaseChangelogTable({
  changelogs,
  loading,
}: {
  changelogs: Changelog[];
  loading: boolean;
}) {
  const { t } = useTranslation();

  const handleRowClick = useCallback(
    (changelog: Changelog, e: React.MouseEvent) => {
      const url = changelogLink(changelog);
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        void router.push(url);
      }
    },
    []
  );

  if (loading) {
    return (
      <div className="text-sm text-control-light">{t("common.loading")}</div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border border-block-border">
      <table className="min-w-full divide-y divide-block-border">
        <thead className="bg-control-bg">
          <tr className="text-left text-sm text-control-light">
            <th className="w-12 px-4 py-2" />
            <th className="w-[180px] px-4 py-2 font-medium">
              {t("common.created-at")}
            </th>
            <th className="min-w-[200px] px-4 py-2 font-medium">
              {t("common.rollout")}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-block-border bg-white">
          {changelogs.map((changelog) => (
            <tr
              key={changelog.name}
              className="cursor-pointer hover:bg-gray-50"
              onClick={(e) => handleRowClick(changelog, e)}
            >
              <td className="px-4 py-3 text-center">
                <ChangelogStatusIcon status={changelog.status} />
              </td>
              <td className="px-4 py-3 text-sm text-main">
                {getDateForPbTimestampProtoEs(changelog.createTime)
                  ? humanizeDate(
                      getDateForPbTimestampProtoEs(changelog.createTime) as Date
                    )
                  : "-"}
              </td>
              <td className="truncate px-4 py-3 text-sm text-main">
                {changelog.planTitle || "-"}
              </td>
            </tr>
          ))}
          {changelogs.length === 0 && (
            <tr>
              <td
                className="px-4 py-6 text-center text-sm text-control-light"
                colSpan={3}
              >
                {t("common.no-data")}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
