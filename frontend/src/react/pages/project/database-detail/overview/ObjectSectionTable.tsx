import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

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
      <div className="rounded-lg border border-dashed border-block-border px-4 py-6 text-sm text-control-light">
        {t("common.loading")}
      </div>
    );
  }

  if (rows.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-block-border px-4 py-6 text-sm text-control-light">
        {emptyText || "-"}
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border border-block-border">
      <table className="min-w-full divide-y divide-block-border">
        <thead className="bg-control-bg">
          <tr className="text-left text-sm text-control-light">
            <th className="px-4 py-2 font-medium">{t("common.name")}</th>
            <th className="px-4 py-2 font-medium">{t("common.definition")}</th>
            <th className="px-4 py-2 font-medium">{t("common.comment")}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-block-border bg-white">
          {rows.map((row) => (
            <tr
              key={row.key}
              className={
                row.onClick ? "cursor-pointer hover:bg-control-bg" : ""
              }
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
              <td className="px-4 py-3 text-sm text-main">{row.name}</td>
              <td className="px-4 py-3 text-sm text-control">
                {row.description}
              </td>
              <td className="px-4 py-3 text-sm text-control">
                {row.comment || "-"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
