import { isEqual } from "lodash-es";
import { useSchemaDesignerContext } from "../common";

export const isTableChanged = (schemaId: string, tableId: string): boolean => {
  const { originalSchemas, editableSchemas } = useSchemaDesignerContext();
  const originSchema = originalSchemas.value.find(
    (schema) => schema.id === schemaId
  );
  const schema = editableSchemas.value.find((schema) => schema.id === schemaId);
  const originTable = originSchema?.tableList.find(
    (table) => table.id === tableId
  );
  const table = schema?.tableList.find((table) => table.id === tableId);
  const originForeignKeyList =
    originSchema?.foreignKeyList.filter((fk) => fk.tableId === table?.id) || [];
  const foreignKeyList =
    schema?.foreignKeyList.filter((fk) => fk.tableId === table?.id) || [];

  return (
    !isEqual(originTable, table) ||
    !isEqual(originForeignKeyList, foreignKeyList)
  );
};
