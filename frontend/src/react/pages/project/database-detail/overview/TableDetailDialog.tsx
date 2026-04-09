import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";

export interface TableDetailDialogData {
  classification?: string;
  classificationConfig?: DataClassificationSetting_DataClassificationConfig;
  columns: {
    characterSet?: string;
    classification?: string;
    collation?: string;
    comment?: string;
    defaultValue: string;
    name: string;
    nullable: boolean;
    semanticType?: string;
    type: string;
  }[];
  collation?: string;
  dataSize: string;
  engine?: string;
  indexes: {
    comment?: string;
    expressions: string[];
    name: string;
    unique: boolean;
    visible?: boolean;
  }[];
  indexSize: string;
  name: string;
  rowCount: string;
  showCharacterSet?: boolean;
  showColumnClassification?: boolean;
  showColumnCollation?: boolean;
  showColumns?: boolean;
  showCollation?: boolean;
  showEngine?: boolean;
  showIndexComment?: boolean;
  showIndexes?: boolean;
  showIndexSize?: boolean;
  showIndexVisible?: boolean;
  showSemanticType?: boolean;
}

function getClassificationTitle(
  classification: string | undefined,
  classificationConfig:
    | DataClassificationSetting_DataClassificationConfig
    | undefined
) {
  if (!classification || !classificationConfig) {
    return "-";
  }

  return (
    classificationConfig.classification[classification]?.title || classification
  );
}

export function TableDetailDialog({
  table,
  open,
  onOpenChange,
}: {
  table?: TableDetailDialogData;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation();
  const [columnSearchKeyword, setColumnSearchKeyword] = useState("");

  const filteredColumns = useMemo(() => {
    if (!table) {
      return [];
    }

    const keyword = columnSearchKeyword.trim().toLowerCase();
    if (!keyword) {
      return table.columns;
    }

    return table.columns.filter((column) =>
      column.name.toLowerCase().includes(keyword)
    );
  }, [columnSearchKeyword, table?.columns]);

  useEffect(() => {
    setColumnSearchKeyword("");
  }, [table?.name]);

  if (!table) {
    return null;
  }

  const showCharacterSetColumn = table.showCharacterSet;
  const showColumnClassification = table.showColumnClassification;
  const showColumnCollation = table.showColumnCollation;
  const showColumns = table.showColumns ?? true;
  const showIndexCommentColumn = table.showIndexComment;
  const showIndexVisibleColumn = table.showIndexVisible;
  const showSemanticTypeColumn = table.showSemanticType;
  const showSummaryCollation = table.showCollation;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl p-6">
        <DialogTitle>{table.name}</DialogTitle>
        <div className="mt-4 grid gap-4 sm:grid-cols-3">
          <div>
            <div className="text-sm font-medium text-control-light">
              {t("database.classification.self")}
            </div>
            <div className="mt-1 text-sm text-main">
              {getClassificationTitle(
                table.classification,
                table.classificationConfig
              )}
            </div>
          </div>
          {table.showEngine && (
            <div>
              <div className="text-sm font-medium text-control-light">
                {t("database.engine")}
              </div>
              <div className="mt-1 text-sm text-main">
                {table.engine || "-"}
              </div>
            </div>
          )}
          <div>
            <div className="text-sm font-medium text-control-light">
              {t("database.row-count-estimate")}
            </div>
            <div className="mt-1 text-sm text-main">{table.rowCount}</div>
          </div>
          <div>
            <div className="text-sm font-medium text-control-light">
              {t("database.data-size")}
            </div>
            <div className="mt-1 text-sm text-main">{table.dataSize}</div>
          </div>
          {table.showIndexSize && (
            <div>
              <div className="text-sm font-medium text-control-light">
                {t("database.index-size")}
              </div>
              <div className="mt-1 text-sm text-main">{table.indexSize}</div>
            </div>
          )}
          {showSummaryCollation && (
            <div>
              <div className="text-sm font-medium text-control-light">
                {t("db.collation")}
              </div>
              <div className="mt-1 text-sm text-main">
                {table.collation || "-"}
              </div>
            </div>
          )}
        </div>

        {showColumns && (
          <div className="mt-6 flex flex-col gap-y-4">
            <div className="flex items-center justify-between gap-3">
              <div className="text-sm font-medium text-control-light">
                {t("database.columns")}
              </div>
              <Input
                className="w-full max-w-sm"
                placeholder={t("common.filter-by-name")}
                value={columnSearchKeyword}
                onChange={(event) => setColumnSearchKeyword(event.target.value)}
              />
            </div>
            <div className="rounded-lg border border-block-border">
              <table className="min-w-full divide-y divide-block-border text-sm">
                <thead className="bg-control-bg">
                  <tr className="text-left text-sm text-control-light">
                    <th className="px-4 py-2 font-medium">
                      {t("common.name")}
                    </th>
                    {showSemanticTypeColumn && (
                      <th className="px-4 py-2 font-medium">
                        {t(
                          "settings.sensitive-data.semantic-types.table.semantic-type"
                        )}
                      </th>
                    )}
                    {showColumnClassification && (
                      <th className="px-4 py-2 font-medium">
                        {t("database.classification.self")}
                      </th>
                    )}
                    <th className="px-4 py-2 font-medium">
                      {t("common.type")}
                    </th>
                    <th className="px-4 py-2 font-medium">
                      {t("common.default")}
                    </th>
                    <th className="px-4 py-2 font-medium">
                      {t("database.nullable")}
                    </th>
                    {showCharacterSetColumn && (
                      <th className="px-4 py-2 font-medium">
                        {t("db.character-set")}
                      </th>
                    )}
                    {showColumnCollation && (
                      <th className="px-4 py-2 font-medium">
                        {t("db.collation")}
                      </th>
                    )}
                    <th className="px-4 py-2 font-medium">
                      {t("common.comment")}
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-block-border bg-white">
                  {filteredColumns.map((column) => (
                    <tr key={column.name}>
                      <td className="px-4 py-3 text-sm text-main">
                        {column.name}
                      </td>
                      {showSemanticTypeColumn && (
                        <td className="px-4 py-3 text-sm text-control">
                          {column.semanticType || "-"}
                        </td>
                      )}
                      {showColumnClassification && (
                        <td className="px-4 py-3 text-sm text-control">
                          {getClassificationTitle(
                            column.classification,
                            table.classificationConfig
                          )}
                        </td>
                      )}
                      <td className="px-4 py-3 text-sm text-control">
                        {column.type}
                      </td>
                      <td className="px-4 py-3 text-sm text-control">
                        {column.defaultValue}
                      </td>
                      <td className="px-4 py-3 text-sm text-control">
                        <input
                          checked={column.nullable}
                          disabled
                          readOnly
                          type="checkbox"
                        />
                      </td>
                      {showCharacterSetColumn && (
                        <td className="px-4 py-3 text-sm text-control">
                          {column.characterSet || "-"}
                        </td>
                      )}
                      {showColumnCollation && (
                        <td className="px-4 py-3 text-sm text-control">
                          {column.collation || "-"}
                        </td>
                      )}
                      <td className="px-4 py-3 text-sm text-control">
                        {column.comment || "-"}
                      </td>
                    </tr>
                  ))}
                  {filteredColumns.length === 0 && (
                    <tr>
                      <td
                        className="px-4 py-6 text-center text-sm text-control-light"
                        colSpan={
                          5 +
                          (showSemanticTypeColumn ? 1 : 0) +
                          (showColumnClassification ? 1 : 0) +
                          (showCharacterSetColumn ? 1 : 0) +
                          (showColumnCollation ? 1 : 0)
                        }
                      >
                        {t("common.no-data")}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {table.showIndexes && table.indexes.length > 0 && (
          <div className="mt-6 flex flex-col gap-y-4">
            <div className="text-sm font-medium text-control-light">
              {t("database.indexes")}
            </div>
            {table.indexes.map((index) => (
              <div
                key={index.name}
                className="rounded-lg border border-block-border"
              >
                <div className="border-b border-block-border px-4 py-3 text-base font-medium text-main">
                  {index.name}
                </div>
                <table className="min-w-full divide-y divide-block-border text-sm">
                  <thead className="bg-control-bg">
                    <tr className="text-left text-sm text-control-light">
                      <th className="px-4 py-2 font-medium">
                        {t("database.expression")}
                      </th>
                      <th className="px-4 py-2 font-medium">
                        {t("database.unique")}
                      </th>
                      {showIndexVisibleColumn && (
                        <th className="px-4 py-2 font-medium">
                          {t("database.visible")}
                        </th>
                      )}
                      {showIndexCommentColumn && (
                        <th className="px-4 py-2 font-medium">
                          {t("common.comment")}
                        </th>
                      )}
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-block-border bg-white">
                    <tr>
                      <td className="px-4 py-3 text-sm text-control">
                        {index.expressions.join(", ") || "-"}
                      </td>
                      <td className="px-4 py-3 text-sm text-control">
                        {String(index.unique)}
                      </td>
                      {showIndexVisibleColumn && (
                        <td className="px-4 py-3 text-sm text-control">
                          {String(index.visible)}
                        </td>
                      )}
                      {showIndexCommentColumn && (
                        <td className="px-4 py-3 text-sm text-control">
                          {index.comment || "-"}
                        </td>
                      )}
                    </tr>
                  </tbody>
                </table>
              </div>
            ))}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
