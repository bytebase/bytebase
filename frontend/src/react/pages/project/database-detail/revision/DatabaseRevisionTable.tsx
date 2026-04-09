import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { router } from "@/router";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import { humanizeDate } from "@/utils";
import { getRevisionType, revisionLink } from "@/utils/v1/revision";

export function DatabaseRevisionTable({
  loading,
  revisions,
  onDelete,
}: {
  loading: boolean;
  revisions: Revision[];
  onDelete: (name: string) => void | Promise<void>;
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
            <th className="px-4 py-2 font-medium">{t("common.version")}</th>
            <th className="px-4 py-2 font-medium">{t("common.type")}</th>
            <th className="px-4 py-2 font-medium">{t("common.created-at")}</th>
            <th className="px-4 py-2 font-medium">{t("common.operations")}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-block-border bg-white">
          {revisions.map((revision) => (
            <tr key={revision.name}>
              <td className="px-4 py-3 text-sm text-main">
                <button
                  className="cursor-pointer text-left hover:text-accent"
                  type="button"
                  onClick={() => void router.push(revisionLink(revision))}
                >
                  {revision.version || revision.name}
                </button>
              </td>
              <td className="px-4 py-3 text-sm text-control">
                {getRevisionType(revision.type)}
              </td>
              <td className="px-4 py-3 text-sm text-control">
                {getDateForPbTimestampProtoEs(revision.createTime)
                  ? humanizeDate(
                      getDateForPbTimestampProtoEs(revision.createTime) as Date
                    )
                  : "-"}
              </td>
              <td className="px-4 py-3 text-sm text-control">
                <Button
                  data-name={revision.name}
                  size="sm"
                  type="button"
                  variant="ghost"
                  onClick={() => void onDelete(revision.name)}
                >
                  {t("common.delete")}
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
