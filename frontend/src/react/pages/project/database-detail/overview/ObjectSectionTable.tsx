import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";

export interface ObjectSectionRow {
  key: string;
  name: string;
  description?: ReactNode;
  comment?: string;
  onClick?: () => void;
}

export function ObjectSectionTable({
  rows,
  emptyText,
  loading = false,
}: {
  rows: ObjectSectionRow[];
  emptyText?: string;
  loading?: boolean;
}) {
  const { t } = useTranslation();

  if (loading) {
    return (
      <div className="rounded border border-dashed border-block-border px-4 py-6 text-sm text-control-light">
        {t("common.loading")}
      </div>
    );
  }

  if (rows.length === 0) {
    return (
      <div className="rounded border border-dashed border-block-border px-4 py-6 text-sm text-control-light">
        {emptyText || "-"}
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded border border-block-border">
      <Table>
        <TableHeader className="bg-control-bg">
          <TableRow>
            <TableHead>{t("common.name")}</TableHead>
            <TableHead>{t("common.definition")}</TableHead>
            <TableHead>{t("common.comment")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow
              key={row.key}
              className={row.onClick ? "cursor-pointer" : ""}
              role={row.onClick ? "button" : undefined}
              tabIndex={row.onClick ? 0 : undefined}
              onClick={row.onClick}
              onKeyDown={
                row.onClick
                  ? (event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        row.onClick?.();
                      }
                    }
                  : undefined
              }
            >
              <TableCell className="text-main">{row.name}</TableCell>
              <TableCell>{row.description}</TableCell>
              <TableCell>{row.comment || "-"}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
