import { Check } from "lucide-react";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { HumanizeTs } from "@/components/HumanizeTs";
import { TIMESTAMP_COLUMN } from "@/components/timestampColumn";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useColumnWidths } from "@/hooks/useColumnWidths";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type Changelog,
  Changelog_Status,
} from "@/types/proto-es/v1/changelog_service_pb";
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
  // A fixed-layout table spreads any spare width across every column that
  // has one, so the rollout title alone is left unsized: it takes what the
  // others leave, and the timestamp keeps exactly the width its form needs
  // at any viewport. Its default is only the floor the table keeps before it
  // scrolls.
  const columns = useMemo(
    () => [
      { key: "status", title: "", defaultWidth: 48, resizable: false },
      {
        key: "created",
        title: t("common.created-at"),
        defaultWidth: TIMESTAMP_COLUMN.compact.width,
        minWidth: TIMESTAMP_COLUMN.compact.minWidth,
        resizable: true,
      },
      {
        key: "rollout",
        title: t("common.rollout"),
        defaultWidth: 200,
        resizable: false,
        fills: true,
      },
    ],
    [t]
  );
  const { widths, totalWidth, onResizeStart } = useColumnWidths(columns);

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
    <div className="overflow-hidden rounded-sm border border-block-border">
      <Table className="table-fixed" style={{ minWidth: `${totalWidth}px` }}>
        <colgroup>
          {widths.map((w, index) => (
            <col
              key={columns[index].key}
              style={{
                width: "fills" in columns[index] ? undefined : `${w}px`,
              }}
            />
          ))}
        </colgroup>
        <TableHeader className="bg-control-bg">
          <TableRow className="text-left text-sm text-control-light hover:bg-control-bg">
            {columns.map((column, index) => (
              <TableHead
                key={column.key}
                className="whitespace-nowrap"
                resizable={column.resizable}
                onResizeStart={
                  column.resizable ? (e) => onResizeStart(index, e) : undefined
                }
              >
                {column.title}
              </TableHead>
            ))}
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
              <TableCell className="truncate text-main">
                {changelog.createTime ? (
                  <HumanizeTs
                    mode="compact"
                    tsMs={getTimeForPbTimestampProtoEs(changelog.createTime)}
                  />
                ) : (
                  "-"
                )}
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
