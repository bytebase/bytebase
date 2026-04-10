import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
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
    <div className="overflow-hidden rounded border border-block-border">
      <Table>
        <TableHeader className="bg-control-bg">
          <TableRow>
            <TableHead>{t("common.version")}</TableHead>
            <TableHead>{t("common.type")}</TableHead>
            <TableHead>{t("common.created-at")}</TableHead>
            <TableHead>{t("common.operations")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {revisions.map((revision) => (
            <TableRow key={revision.name}>
              <TableCell className="text-main">
                <button
                  className="cursor-pointer text-left hover:text-accent"
                  type="button"
                  onClick={() => void router.push(revisionLink(revision))}
                >
                  {revision.version || revision.name}
                </button>
              </TableCell>
              <TableCell>{getRevisionType(revision.type)}</TableCell>
              <TableCell>
                {getDateForPbTimestampProtoEs(revision.createTime)
                  ? humanizeDate(
                      getDateForPbTimestampProtoEs(revision.createTime) as Date
                    )
                  : "-"}
              </TableCell>
              <TableCell>
                <Button
                  data-name={revision.name}
                  size="sm"
                  type="button"
                  variant="ghost"
                  onClick={() => void onDelete(revision.name)}
                >
                  {t("common.delete")}
                </Button>
              </TableCell>
            </TableRow>
          ))}
          {revisions.length === 0 && (
            <TableRow>
              <TableCell
                className="py-6 text-center text-control-light"
                colSpan={4}
              >
                {t("common.no-data")}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  );
}
