import { useTranslation } from "react-i18next";
import { Checkbox } from "@/react/components/ui/checkbox";
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
  selectedNames,
  onSelectedNamesChange,
}: {
  loading: boolean;
  revisions: Revision[];
  selectedNames: Set<string>;
  onSelectedNamesChange: (names: Set<string>) => void;
}) {
  const { t } = useTranslation();

  const allSelected =
    revisions.length > 0 && selectedNames.size === revisions.length;
  const someSelected =
    selectedNames.size > 0 && selectedNames.size < revisions.length;
  const toggleSelectAll = () => {
    if (allSelected) {
      onSelectedNamesChange(new Set());
    } else {
      onSelectedNamesChange(new Set(revisions.map((r) => r.name)));
    }
  };

  const toggleSelection = (name: string) => {
    const next = new Set(selectedNames);
    if (next.has(name)) {
      next.delete(name);
    } else {
      next.add(name);
    }
    onSelectedNamesChange(next);
  };

  if (loading) {
    return (
      <div className="text-sm text-control-light">{t("common.loading")}</div>
    );
  }

  return (
    <Table>
      <TableHeader className="bg-control-bg">
        <TableRow>
          <TableHead className="w-12">
            <Checkbox
              checked={someSelected ? "indeterminate" : allSelected}
              onCheckedChange={toggleSelectAll}
            />
          </TableHead>
          <TableHead>{t("common.version")}</TableHead>
          <TableHead>{t("common.type")}</TableHead>
          <TableHead>{t("common.created-at")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {revisions.map((revision) => (
          <TableRow
            key={revision.name}
            className="cursor-pointer"
            data-state={
              selectedNames.has(revision.name) ? "selected" : undefined
            }
            onClick={() => void router.push(revisionLink(revision))}
          >
            <TableCell className="w-12">
              <Checkbox
                checked={selectedNames.has(revision.name)}
                onCheckedChange={() => toggleSelection(revision.name)}
                onClick={(e) => e.stopPropagation()}
              />
            </TableCell>
            <TableCell className="text-main">
              {revision.version || revision.name}
            </TableCell>
            <TableCell>{getRevisionType(revision.type)}</TableCell>
            <TableCell>
              {getDateForPbTimestampProtoEs(revision.createTime)
                ? humanizeDate(
                    getDateForPbTimestampProtoEs(revision.createTime) as Date
                  )
                : "-"}
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
  );
}
