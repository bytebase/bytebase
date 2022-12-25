import {
  isEqualForeignKeyList,
  isEqualForeignKey,
} from "@/components/SchemaEditor/utils/table";
import {
  AlterTableContext,
  CreateTableContext,
  DropTableContext,
  RenameTableContext,
} from "@/types";
import { Column, ForeignKey, Schema } from "@/types/schemaEditor/atomType";
import { isEqual, isUndefined } from "lodash-es";
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
    const foreignKeyList = schema.foreignKeyList.filter(
      (fk) => fk.table === createTableContext.name
    );
    for (const foreignKey of foreignKeyList) {
      if (foreignKey.columnList.length > 0) {
        createTableContext.addForeignKeyList.push({
          columnList: foreignKey.columnList.map(
            (columnRef) => columnRef.value.newName
          ),
          referencedTable: foreignKey.referencedTable,
          referencedColumnList: foreignKey.referencedColumns,
        });
      }
    }
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
      .find((pk) => pk.table === table.oldName)
      ?.columnList.map((columnRef) => columnRef.value)
      .sort();
    const primaryKey = schema?.primaryKeyList
      .find((pk) => pk.table === table.newName)
      ?.columnList.filter((column) => column.value.status !== "dropped")
      .map((columnRef) => columnRef.value)
      .sort();
    const originForeignKeyList = originSchema.foreignKeyList.filter(
      (fk) => fk.table === table.newName
    );
    const foreignKeyList = schema.foreignKeyList.filter(
      (fk) => fk.table === table.newName
    );

    if (
      !isEqual(originTable, table) ||
      !isEqual(originPrimaryKey, primaryKey) ||
      !isEqualForeignKeyList(originForeignKeyList, foreignKeyList)
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
        !isEqualForeignKeyList(originForeignKeyList, foreignKeyList) ||
        columnListDiffResult.addColumnList.length > 0 ||
        columnListDiffResult.changeColumnList.length > 0 ||
        columnListDiffResult.dropColumnList.length > 0
      ) {
        const alterTableContext: AlterTableContext = {
          name: table.newName,
          ...columnListDiffResult,
          dropPrimaryKey: false,
          dropForeignKeyList: [],
          addForeignKeyList: [],
        };
        composePrimaryKeyWithAlterTableContext(
          originPrimaryKey || [],
          primaryKey || [],
          alterTableContext
        );
        composeForeignKeyWithAlterTableContext(
          originForeignKeyList || [],
          foreignKeyList || [],
          alterTableContext
        );
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

const composePrimaryKeyWithAlterTableContext = (
  originPrimaryKey: Column[],
  primaryKey: Column[],
  alterTableContext: AlterTableContext
) => {
  const droppedColumnNameList = alterTableContext.dropColumnList.map(
    (column) => column.name
  );
  const filterOriginPrimaryKey = originPrimaryKey.filter(
    (column) => !droppedColumnNameList.includes(column.oldName)
  );
  if (!isEqual(filterOriginPrimaryKey, primaryKey)) {
    if (originPrimaryKey.length !== 0) {
      alterTableContext.dropPrimaryKey = true;
    }
    alterTableContext.primaryKeyList = primaryKey.map(
      (column) => column.newName
    );
  }
};

const composeForeignKeyWithAlterTableContext = (
  originForeignKeyList: ForeignKey[],
  foreignKeyList: ForeignKey[],
  alterTableContext: AlterTableContext
) => {
  for (let i = 0; i < foreignKeyList.length; i++) {
    const originForeignKey = originForeignKeyList[i];
    const foreignKey = foreignKeyList[i];

    if (!isEqualForeignKey(originForeignKey, foreignKey)) {
      if (!isUndefined(originForeignKey)) {
        alterTableContext.dropForeignKeyList.push(originForeignKey.name);
      }
      if (foreignKey.columnList.length > 0) {
        alterTableContext.addForeignKeyList.push({
          columnList: foreignKey.columnList.map(
            (columnRef) => columnRef.value.newName
          ),
          referencedTable: foreignKey.referencedTable,
          referencedColumnList: foreignKey.referencedColumns,
        });
      }
    }
  }
};
