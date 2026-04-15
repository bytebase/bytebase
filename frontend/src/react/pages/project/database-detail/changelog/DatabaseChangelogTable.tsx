import { Check } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
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
import {
  type Changelog,
  Changelog_Status,
} from "@/types/proto-es/v1/database_service_pb";
import { humanizeDate } from "@/utils";
import { changelogLink } from "@/utils/v1/changelog";

function ChangelogStatusIcon({ status }: { status: Changelog_Status }) {
  if (status === Changelog_Status.PENDING) {
    return (
      <span className="flex size-5 items-center justify-center rounded-full border-2 border-info bg-background text-info">
        <span
          className="size-2 rounded-full bg-info"
          style={{
            animation: "pulse 2.5s cubic-bezier(0.4, 0, 0.6, 1) infinite",
          }}
        />
      </span>
    );
  }
  if (status === Changelog_Status.DONE) {
    return (
      <span className="flex size-5 items-center justify-center rounded-full bg-success text-accent-text">
        <Check className="size-4" />
      </span>
    );
  }
  if (status === Changelog_Status.FAILED) {
    return (
      <span className="flex size-5 items-center justify-center rounded-full bg-error text-accent-text">
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
    <div className="overflow-hidden rounded border border-block-border">
      <Table className="min-w-full">
        <TableHeader className="bg-control-bg">
          <TableRow className="text-left text-sm text-control-light hover:bg-control-bg">
            <TableHead className="w-12" />
            <TableHead className="w-[180px]">
              {t("common.created-at")}
            </TableHead>
            <TableHead className="min-w-[200px]">
              {t("common.rollout")}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {changelogs.map((changelog) => (
            <TableRow
              key={changelog.name}
              className="cursor-pointer hover:bg-control-bg"
              onClick={(e) => handleRowClick(changelog, e)}
            >
              <TableCell className="text-center">
                <ChangelogStatusIcon status={changelog.status} />
              </TableCell>
              <TableCell className="text-main">
                {getDateForPbTimestampProtoEs(changelog.createTime)
                  ? humanizeDate(
                      getDateForPbTimestampProtoEs(changelog.createTime) as Date
                    )
                  : "-"}
              </TableCell>
              <TableCell className="truncate text-main">
                {changelog.planTitle || "-"}
              </TableCell>
            </TableRow>
          ))}
          {changelogs.length === 0 && (
            <TableRow striped={false}>
              <TableCell
                className="py-6 text-center text-control-light"
                colSpan={3}
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
