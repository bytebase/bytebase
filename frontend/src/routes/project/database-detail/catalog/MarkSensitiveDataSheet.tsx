import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { FormField, FormFieldGroup, FormLabel } from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useAppDatabaseMetadata } from "@/hooks/useAppDatabaseMetadata";
import { updateColumnCatalog } from "@/lib/column-data-table/utils";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { SemanticTypeSetting_SemanticType } from "@/types/proto-es/v1/setting_service_pb";
import { getSemanticTypeListWithBuiltins } from "@/types/semanticTypes";

interface MarkSensitiveDataSheetProps {
  database: Database;
  open: boolean;
  semanticTypeList: SemanticTypeSetting_SemanticType[];
  onOpenChange: (open: boolean) => void;
}

export function MarkSensitiveDataSheet({
  database,
  open,
  semanticTypeList,
  onOpenChange,
}: MarkSensitiveDataSheetProps) {
  const { t } = useTranslation();
  const metadata = useAppDatabaseMetadata(database.name, { autoFetch: open });
  const [schemaIndex, setSchemaIndex] = useState("");
  const [tableIndex, setTableIndex] = useState("");
  const [columnName, setColumnName] = useState("");
  const [semanticTypeId, setSemanticTypeId] = useState("bb.default");
  const [saving, setSaving] = useState(false);
  const availableSemanticTypeList = useMemo(
    () => getSemanticTypeListWithBuiltins(semanticTypeList),
    [semanticTypeList]
  );

  const selectedSchema =
    schemaIndex === "" ? undefined : metadata.schemas[Number(schemaIndex)];
  const selectedTable =
    tableIndex === "" ? undefined : selectedSchema?.tables[Number(tableIndex)];
  const canSave =
    !!selectedSchema &&
    !!selectedTable &&
    !!columnName &&
    !!semanticTypeId &&
    !saving;

  useEffect(() => {
    if (open) {
      return;
    }
    setSchemaIndex("");
    setTableIndex("");
    setColumnName("");
    setSemanticTypeId("bb.default");
    setSaving(false);
  }, [database.name, open]);

  const handleSave = async () => {
    if (!canSave || !selectedSchema || !selectedTable) {
      return;
    }

    setSaving(true);
    try {
      await updateColumnCatalog({
        database: database.name,
        schema: selectedSchema.name,
        table: selectedTable.name,
        column: columnName,
        columnCatalog: {
          semanticType: semanticTypeId,
        },
        notification: "common.updated",
      });
      onOpenChange(false);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent width="standard">
        <SheetHeader>
          <SheetTitle>
            {t("settings.sensitive-data.mark-sensitive-data")}
          </SheetTitle>
          <SheetDescription>
            {t("settings.sensitive-data.semantic-types.label")}
          </SheetDescription>
        </SheetHeader>
        <SheetBody>
          <FormFieldGroup>
            <FormField>
              <FormLabel htmlFor="mark-sensitive-data-schema">
                {t("common.schema")}
              </FormLabel>
              <Select
                value={schemaIndex}
                onValueChange={(value) => {
                  setSchemaIndex(value ?? "");
                  setTableIndex("");
                  setColumnName("");
                }}
              >
                <SelectTrigger
                  id="mark-sensitive-data-schema"
                  className="w-full"
                >
                  <SelectValue>
                    {selectedSchema
                      ? selectedSchema.name || t("database.schema.unspecified")
                      : t("common.select")}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {metadata.schemas.map((schema, index) => (
                    <SelectItem
                      key={`${schema.name}-${index}`}
                      value={`${index}`}
                    >
                      {schema.name || t("database.schema.unspecified")}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <FormField>
              <FormLabel htmlFor="mark-sensitive-data-table">
                {t("common.table")}
              </FormLabel>
              <Select
                value={tableIndex}
                disabled={!selectedSchema}
                onValueChange={(value) => {
                  setTableIndex(value ?? "");
                  setColumnName("");
                }}
              >
                <SelectTrigger
                  id="mark-sensitive-data-table"
                  className="w-full"
                >
                  <SelectValue>
                    {selectedTable?.name || t("common.select")}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {selectedSchema?.tables.map((table, index) => (
                    <SelectItem
                      key={`${table.name}-${index}`}
                      value={`${index}`}
                    >
                      {table.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <FormField>
              <FormLabel htmlFor="mark-sensitive-data-column">
                {t("common.column")}
              </FormLabel>
              <Select
                value={columnName}
                disabled={!selectedTable}
                onValueChange={(value) => setColumnName(value ?? "")}
              >
                <SelectTrigger
                  id="mark-sensitive-data-column"
                  className="w-full"
                >
                  <SelectValue placeholder={t("common.select")} />
                </SelectTrigger>
                <SelectContent>
                  {selectedTable?.columns.map((column) => (
                    <SelectItem key={column.name} value={column.name}>
                      {column.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>

            <FormField>
              <FormLabel htmlFor="mark-sensitive-data-semantic-type">
                {t(
                  "settings.sensitive-data.semantic-types.table.semantic-type"
                )}
              </FormLabel>
              <Select
                value={semanticTypeId}
                onValueChange={(value) => setSemanticTypeId(value ?? "")}
              >
                <SelectTrigger
                  id="mark-sensitive-data-semantic-type"
                  className="w-full"
                >
                  <SelectValue
                    placeholder={t(
                      "settings.sensitive-data.semantic-types.select"
                    )}
                  />
                </SelectTrigger>
                <SelectContent>
                  {availableSemanticTypeList.map((semanticType) => (
                    <SelectItem key={semanticType.id} value={semanticType.id}>
                      {semanticType.title || semanticType.id}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
          </FormFieldGroup>
        </SheetBody>
        <SheetFooter>
          <Button
            type="button"
            appearance="outline"
            onClick={() => onOpenChange(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button type="button" disabled={!canSave} onClick={handleSave}>
            {t("common.save")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
