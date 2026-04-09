import { Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { MaskData } from "@/components/SensitiveData/types";
import { getMaskDataIdentifier } from "@/components/SensitiveData/utils";
import { Button } from "@/react/components/ui/button";

export interface SensitiveColumnTableProps {
  checkedColumnList: MaskData[];
  columnList: MaskData[];
  rowSelectable: boolean;
  showOperation: boolean;
  onCheckedColumnListChange: (list: MaskData[]) => void;
  onDelete: (item: MaskData) => void | Promise<void>;
}

export function SensitiveColumnTable({
  checkedColumnList,
  columnList,
  rowSelectable,
  showOperation,
  onCheckedColumnListChange,
  onDelete,
}: SensitiveColumnTableProps) {
  const { t } = useTranslation();
  const checkedSet = new Set(checkedColumnList.map(getMaskDataIdentifier));

  const toggleChecked = (item: MaskData, checked: boolean) => {
    const key = getMaskDataIdentifier(item);
    if (checked) {
      onCheckedColumnListChange(
        checkedColumnList.some((row) => getMaskDataIdentifier(row) === key)
          ? checkedColumnList
          : [...checkedColumnList, item]
      );
      return;
    }
    onCheckedColumnListChange(
      checkedColumnList.filter((row) => getMaskDataIdentifier(row) !== key)
    );
  };

  const handleDelete = async (item: MaskData) => {
    if (
      typeof window !== "undefined" &&
      typeof window.confirm === "function" &&
      !window.confirm(t("settings.sensitive-data.remove-sensitive-column-tips"))
    ) {
      return;
    }
    await onDelete(item);
  };

  return (
    <div className="overflow-x-auto rounded-sm border border-control-border">
      <table className="w-full min-w-[720px] text-sm">
        <thead>
          <tr className="border-b border-control-border bg-gray-50 text-left">
            {rowSelectable && <th className="w-12 px-4 py-2" />}
            <th className="px-4 py-2 font-medium">{t("common.table")}</th>
            <th className="px-4 py-2 font-medium">{t("database.column")}</th>
            <th className="px-4 py-2 font-medium">
              {t("settings.sensitive-data.semantic-types.table.semantic-type")}
            </th>
            <th className="px-4 py-2 font-medium">
              {t("database.classification.self")}
            </th>
            {showOperation && (
              <th className="w-20 px-4 py-2 font-medium">
                {t("common.operation")}
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {columnList.map((item) => {
            const key = getMaskDataIdentifier(item);
            const tableName = item.schema
              ? `${item.schema}.${item.table}`
              : item.table;
            return (
              <tr
                key={key}
                className="border-b border-control-border last:border-0"
              >
                {rowSelectable && (
                  <td className="px-4 py-3">
                    <input
                      type="checkbox"
                      checked={checkedSet.has(key)}
                      onChange={(event) =>
                        toggleChecked(item, event.target.checked)
                      }
                    />
                  </td>
                )}
                <td className="px-4 py-3">{tableName}</td>
                <td className="px-4 py-3">{item.column || "-"}</td>
                <td className="px-4 py-3">{item.semanticTypeId || "-"}</td>
                <td className="px-4 py-3">{item.classificationId || "-"}</td>
                {showOperation && (
                  <td className="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={t("common.remove")}
                      onClick={() => void handleDelete(item)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </td>
                )}
              </tr>
            );
          })}
          {columnList.length === 0 && (
            <tr>
              <td
                className="px-4 py-6 text-center text-control-light"
                colSpan={
                  rowSelectable
                    ? showOperation
                      ? 6
                      : 5
                    : showOperation
                      ? 5
                      : 4
                }
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
