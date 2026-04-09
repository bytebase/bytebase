import { useTranslation } from "react-i18next";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import type { TableMetadata } from "@/types/proto-es/v1/database_service_pb";
import { bytesToString } from "@/utils";

export function TableDetailDialog({
  table,
  open,
  onOpenChange,
}: {
  table?: TableMetadata;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation();

  if (!table) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl p-6">
        <DialogTitle>{table.name}</DialogTitle>
        <div className="mt-4 grid gap-4 sm:grid-cols-3">
          <div>
            <div className="text-sm font-medium text-control-light">
              {t("database.row-count-est")}
            </div>
            <div className="mt-1 text-sm text-main">
              {String(table.rowCount)}
            </div>
          </div>
          <div>
            <div className="text-sm font-medium text-control-light">
              {t("database.data-size")}
            </div>
            <div className="mt-1 text-sm text-main">
              {bytesToString(Number(table.dataSize))}
            </div>
          </div>
          <div>
            <div className="text-sm font-medium text-control-light">
              {t("database.index-size")}
            </div>
            <div className="mt-1 text-sm text-main">
              {bytesToString(Number(table.indexSize))}
            </div>
          </div>
        </div>
        <div className="mt-6">
          <div className="text-sm font-medium text-control-light">
            {t("database.columns")}
          </div>
          <div className="mt-2 rounded-lg border border-block-border">
            <table className="min-w-full divide-y divide-block-border">
              <thead className="bg-control-bg">
                <tr className="text-left text-sm text-control-light">
                  <th className="px-4 py-2 font-medium">{t("common.name")}</th>
                  <th className="px-4 py-2 font-medium">
                    {t("database.type")}
                  </th>
                  <th className="px-4 py-2 font-medium">
                    {t("common.comment")}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-block-border bg-white">
                {table.columns.map((column) => (
                  <tr key={column.name}>
                    <td className="px-4 py-3 text-sm text-main">
                      {column.name}
                    </td>
                    <td className="px-4 py-3 text-sm text-control">
                      {column.type}
                    </td>
                    <td className="px-4 py-3 text-sm text-control">
                      {column.comment || "-"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
