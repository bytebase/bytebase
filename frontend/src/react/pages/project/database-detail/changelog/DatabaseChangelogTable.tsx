import { useTranslation } from "react-i18next";
import { router } from "@/router";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import { humanizeDate } from "@/utils";
import { changelogLink } from "@/utils/v1/changelog";

export function DatabaseChangelogTable({
  changelogs,
  loading,
}: {
  changelogs: Changelog[];
  loading: boolean;
}) {
  const { t } = useTranslation();

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
            <th className="px-4 py-2 font-medium">{t("common.name")}</th>
            <th className="px-4 py-2 font-medium">{t("common.created-at")}</th>
            <th className="px-4 py-2 font-medium">{t("common.rollout")}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-block-border bg-white">
          {changelogs.map((changelog) => (
            <tr key={changelog.name}>
              <td className="px-4 py-3 text-sm text-main">
                <button
                  className="cursor-pointer text-left hover:text-accent"
                  type="button"
                  onClick={() => void router.push(changelogLink(changelog))}
                >
                  {changelog.name}
                </button>
              </td>
              <td className="px-4 py-3 text-sm text-control">
                {getDateForPbTimestampProtoEs(changelog.createTime)
                  ? humanizeDate(
                      getDateForPbTimestampProtoEs(changelog.createTime) as Date
                    )
                  : "-"}
              </td>
              <td className="px-4 py-3 text-sm text-control">
                {changelog.planTitle || "-"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
