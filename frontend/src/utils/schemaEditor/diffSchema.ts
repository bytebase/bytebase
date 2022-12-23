import {
  AlterTableContext,
  CreateTableContext,
  DropTableContext,
  RenameTableContext,
} from "@/types";
import { Schema } from "@/types/schemaEditor/atomType";
import { isEqual } from "lodash-es";
import { diffColumnList } from "./diffColumn";
import { transformTableToCreateTableContext } from "./transform";

export const diffSchema = (originSchema: Schema, schema: Schema) => {
  const createTableContextList: CreateTableContext[] = [];
  const createdTableList = schema.tableList.filter(
    (table) => table.status === "created"
  );
  for (const table of createdTableList) {
    const createTableContext = transformTableToCreateTableContext(table);
    const diffColumnListResult = diffColumnList([], table.columnList);
    createTableContext.addColumnList = diffColumnListResult.addColumnList;
    createTableContext.primaryKeyList =
      schema.primaryKeyList
        .find((pk) => pk.table === createTableContext.name)
        ?.columnList.map((columnRef) => columnRef.value.newName) ?? [];
    createTableContextList.push(createTableContext);
  }

  const alterTableContextList: AlterTableContext[] = [];
  const renameTableContextList: RenameTableContext[] = [];
  const changedTableList = schema.tableList.filter(
    (table) => table.status === "normal"
  );
  for (const table of changedTableList) {
    const originTable = originSchema.tableList.find(
      (originTable) => originTable.oldName === table.oldName
    );
    const originPrimaryKey = originSchema?.primaryKeyList
      .find((pk) => pk.table === table.newName)
      ?.columnList.map((columnRef) => columnRef.value)
      .sort();
    const primaryKey = schema?.primaryKeyList
      .find((pk) => pk.table === table.newName)
      ?.columnList.map((columnRef) => columnRef.value)
      .sort();
    if (
      !isEqual(originTable, table) ||
      !isEqual(originPrimaryKey, primaryKey)
    ) {
      if (table.newName !== table.oldName) {
        renameTableContextList.push({
          oldName: table.oldName,
          newName: table.newName,
        });
      }

      const columnListDiffResult = diffColumnList(
        originTable?.columnList ?? [],
        table.columnList
      );
      if (
        !isEqual(originPrimaryKey, primaryKey) ||
        columnListDiffResult.addColumnList.length > 0 ||
        columnListDiffResult.changeColumnList.length > 0 ||
        columnListDiffResult.dropColumnList.length > 0
      ) {
        const alterTableContext: AlterTableContext = {
          name: table.newName,
          ...columnListDiffResult,
          dropPrimaryKey: false,
        };
        if (!isEqual(originPrimaryKey, primaryKey)) {
          if (originPrimaryKey?.length !== 0) {
            alterTableContext.dropPrimaryKey = true;
          }
          alterTableContext.primaryKeyList = primaryKey?.map(
            (column) => column.newName
          );
        }
        alterTableContextList.push(alterTableContext);
      }
    }
  }

  const dropTableContextList: DropTableContext[] = [];
  const droppedTableList = schema.tableList.filter(
    (table) => table.status === "dropped"
  );
  for (const table of droppedTableList) {
    dropTableContextList.push({
      name: table.oldName,
    });
  }

  return {
    createTableList: createTableContextList,
    alterTableList: alterTableContextList,
    renameTableList: renameTableContextList,
    dropTableList: dropTableContextList,
  };
};
