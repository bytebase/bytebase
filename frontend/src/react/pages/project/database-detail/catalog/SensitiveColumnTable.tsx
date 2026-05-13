import { Trash2 } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import type { MaskData } from "@/react/lib/sensitive-data/types";
import { router } from "@/router";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { autoDatabaseRoute } from "@/utils";

const EMPTY_SELECT_VALUE = "__EMPTY__";

type SelectOption = {
  label: string;
  value: string;
};

function itemKey(item: MaskData) {
  const parts = [];
  if (item.schema) {
    parts.push(item.schema);
  }
  parts.push(item.table, item.column);
  return parts.join("::");
}

function EditableSelect({
  value,
  options,
  disabled,
  placeholder,
  onValueChange,
}: {
  value: string;
  options: SelectOption[];
  disabled: boolean;
  placeholder: string;
  onValueChange: (value: string) => void;
}) {
  const normalizedValue = value || EMPTY_SELECT_VALUE;
  const selectedLabel =
    options.find((option) => option.value === value)?.label || placeholder;

  return (
    <Select
      disabled={disabled}
      value={normalizedValue}
      onValueChange={(nextValue) =>
        onValueChange(
          !nextValue || nextValue === EMPTY_SELECT_VALUE ? "" : nextValue
        )
      }
    >
      <SelectTrigger className="w-full">
        <SelectValue>{selectedLabel}</SelectValue>
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={EMPTY_SELECT_VALUE}>{placeholder}</SelectItem>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function SensitiveColumnTable({
  database,
  columnList,
  checkedColumnList,
  showSelection,
  canEdit,
  showOperation,
  semanticTypeOptions,
  classificationOptions,
  onCheckedColumnListChange,
  onSemanticTypeChange,
  onClassificationChange,
  onDelete,
}: {
  database: Database;
  columnList: MaskData[];
  checkedColumnList: MaskData[];
  showSelection: boolean;
  canEdit: boolean;
  showOperation: boolean;
  semanticTypeOptions: SelectOption[];
  classificationOptions: SelectOption[];
  onCheckedColumnListChange: (columnList: MaskData[]) => void;
  onSemanticTypeChange: (
    item: MaskData,
    semanticTypeId: string
  ) => void | Promise<void>;
  onClassificationChange: (
    item: MaskData,
    classificationId: string
  ) => void | Promise<void>;
  onDelete: (item: MaskData) => void;
}) {
  const { t } = useTranslation();
  const checkedKeySet = useMemo(
    () => new Set(checkedColumnList.map(itemKey)),
    [checkedColumnList]
  );
  const showSelectionColumn = showSelection && columnList.length > 0;
  const visibleSelectedCount = useMemo(
    () => columnList.filter((item) => checkedKeySet.has(itemKey(item))).length,
    [columnList, checkedKeySet]
  );
  const allSelected =
    columnList.length > 0 && visibleSelectedCount === columnList.length;
  const someSelected =
    visibleSelectedCount > 0 && visibleSelectedCount < columnList.length;

  const toggleSelection = (item: MaskData) => {
    const key = itemKey(item);
    if (checkedKeySet.has(key)) {
      onCheckedColumnListChange(
        checkedColumnList.filter(
          (selectedItem) => itemKey(selectedItem) !== key
        )
      );
      return;
    }
    onCheckedColumnListChange([...checkedColumnList, item]);
  };

  const toggleSelectAll = () => {
    if (allSelected) {
      const visibleKeySet = new Set(columnList.map(itemKey));
      onCheckedColumnListChange(
        checkedColumnList.filter((item) => !visibleKeySet.has(itemKey(item)))
      );
      return;
    }
    const next = [...checkedColumnList];
    for (const item of columnList) {
      if (!checkedKeySet.has(itemKey(item))) {
        next.push(item);
      }
    }
    onCheckedColumnListChange(next);
  };

  return (
    <div className="overflow-x-auto rounded border border-block-border">
      <Table className="min-w-full">
        <TableHeader className="bg-control-bg">
          <TableRow className="hover:bg-control-bg">
            {showSelectionColumn && (
              <TableHead className="w-12">
                <Checkbox
                  checked={someSelected ? "indeterminate" : allSelected}
                  onCheckedChange={toggleSelectAll}
                  onClick={(event) => event.stopPropagation()}
                />
              </TableHead>
            )}
            <TableHead>{t("common.table")}</TableHead>
            <TableHead>{t("database.column")}</TableHead>
            <TableHead>
              {t("settings.sensitive-data.semantic-types.table.semantic-type")}
            </TableHead>
            <TableHead>{t("database.classification.self")}</TableHead>
            {showOperation && (
              <TableHead className="w-16">{t("common.operation")}</TableHead>
            )}
          </TableRow>
        </TableHeader>
        <TableBody className="bg-background">
          {columnList.length === 0 ? (
            <TableRow>
              <TableCell
                className="py-6 text-center text-control-light"
                colSpan={
                  (showSelectionColumn ? 1 : 0) + 4 + (showOperation ? 1 : 0)
                }
              >
                {t("common.no-data")}
              </TableCell>
            </TableRow>
          ) : (
            columnList.map((item) => {
              const key = itemKey(item);
              const isChecked = checkedKeySet.has(key);
              const semanticTypeDisabled =
                !canEdit || !!item.disableSemanticType;
              const classificationDisabled =
                !canEdit || !!item.disableClassification;

              return (
                <TableRow
                  key={key}
                  data-state={isChecked ? "selected" : undefined}
                >
                  {showSelectionColumn && (
                    <TableCell className="w-12">
                      <Checkbox
                        checked={isChecked}
                        onCheckedChange={() => toggleSelection(item)}
                        onClick={(event) => event.stopPropagation()}
                      />
                    </TableCell>
                  )}
                  <TableCell className="text-main">
                    <a
                      className="normal-link"
                      href={
                        router.resolve({
                          ...autoDatabaseRoute(database),
                          query: {
                            schema: item.schema,
                            table: item.table,
                          },
                          hash: "overview",
                        }).fullPath
                      }
                    >
                      {item.schema
                        ? `${item.schema}.${item.table}`
                        : item.table}
                    </a>
                  </TableCell>
                  <TableCell className="text-main">
                    {item.column || t("common.empty")}
                  </TableCell>
                  <TableCell>
                    <EditableSelect
                      value={item.semanticTypeId}
                      options={semanticTypeOptions}
                      disabled={semanticTypeDisabled}
                      placeholder={t("common.empty")}
                      onValueChange={(semanticTypeId) =>
                        void onSemanticTypeChange(item, semanticTypeId)
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <EditableSelect
                      value={item.classificationId}
                      options={classificationOptions}
                      disabled={classificationDisabled}
                      placeholder={t("common.empty")}
                      onValueChange={(classificationId) =>
                        void onClassificationChange(item, classificationId)
                      }
                    />
                  </TableCell>
                  {showOperation && (
                    <TableCell className="w-16">
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => onDelete(item)}
                        title={t(
                          "settings.sensitive-data.remove-sensitive-column-tips"
                        )}
                        aria-label={t(
                          "settings.sensitive-data.remove-sensitive-column-tips"
                        )}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  )}
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>
    </div>
  );
}
