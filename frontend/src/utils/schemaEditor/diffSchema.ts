import { isEqual, isUndefined } from "lodash-es";
import { useSchemaEditorStore } from "@/store";
import {
  AlterTableContext,
  CreateSchemaContext,
  CreateTableContext,
  DropSchemaContext,
  DropTableContext,
  RenameSchemaContext,
  RenameTableContext,
} from "@/types";
import { Schema } from "@/types/schemaEditor/atomType";
import { diffColumnList } from "./diffColumn";
import { transformTableToCreateTableContext } from "./transform";

interface DiffResult {
  createSchemaList: CreateSchemaContext[];
  renameSchemaList: RenameSchemaContext[];
  dropSchemaList: DropSchemaContext[];
  createTableList: CreateTableContext[];
  alterTableList: AlterTableContext[];
  renameTableList: RenameTableContext[];
  dropTableList: DropTableContext[];
}

export const diffSchema = (
  databaseId: string,
  originSchema: Schema | undefined,
  schema: Schema
): DiffResult => {
  const editorStore = useSchemaEditorStore();
  const createSchemaContextList: CreateSchemaContext[] = [];
  const renameSchemaContextList: RenameSchemaContext[] = [];
  const dropSchemaContextList: DropSchemaContext[] = [];
  if (originSchema && originSchema.name !== schema.name) {
    renameSchemaContextList.push({
      oldName: originSchema.name,
      newName: schema.name,
    });
  }
  if (schema.status === "created") {
    createSchemaContextList.push({
      schema: schema.name,
    });
  } else if (schema.status === "dropped") {
    dropSchemaContextList.push({
      schema: schema.name,
    });
  }

  const createTableContextList: CreateTableContext[] = [];
  const createdTableList = schema.tableList.filter(
    (table) => table.status === "created"
  );
  for (const table of createdTableList) {
    const createTableContext = transformTableToCreateTableContext(
      schema.name,
      table
    );
    const diffColumnListResult = diffColumnList([], table.columnList);
    createTableContext.addColumnList = diffColumnListResult.addColumnList;
    for (const columnId of table.primaryKey.columnIdList) {
      const column = table.columnList.find((column) => column.id === columnId);
      if (column) {
        createTableContext.primaryKeyList.push(column.name);
      }
    }
    const foreignKeyList = schema.foreignKeyList.filter(
      (fk) => fk.tableId === table.id
    );
    for (const foreignKey of foreignKeyList) {
      const referencedSchema = editorStore.getSchema(
        databaseId,
        foreignKey.referencedSchemaId
      );
      const referencedTable = editorStore.getTable(
        databaseId,
        foreignKey.referencedSchemaId,
        foreignKey.referencedTableId
      );
      if (referencedSchema && referencedTable) {
        const columnNameList: string[] = [];
        const referencedColumnNameList: string[] = [];
        for (const columnId of foreignKey.columnIdList) {
          const column = table.columnList.find(
            (column) => column.id === columnId
          );
          if (column) {
            columnNameList.push(column.name);
          }
        }
        for (const referencedColumnId of foreignKey.referencedColumnIdList) {
          const column = referencedTable.columnList.find(
            (column) => column.id === referencedColumnId
          );
          if (column) {
            referencedColumnNameList.push(column.name);
          }
        }

        createTableContext.addForeignKeyList.push({
          columnList: columnNameList,
          referencedSchema: referencedSchema.name,
          referencedTable: referencedTable.name,
          referencedColumnList: referencedColumnNameList,
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
    const originTable = originSchema?.tableList.find(
      (originTable) => originTable.id === table.id
    );
    if (!originTable) {
      continue;
    }

    const originPrimaryKey = originTable.primaryKey;
    const primaryKey = table.primaryKey;
    const originForeignKeyList = originSchema?.foreignKeyList.filter(
      (fk) => fk.tableId === table.id
    );
    const foreignKeyList = schema.foreignKeyList.filter(
      (fk) => fk.tableId === table.id
    );

    if (
      !isEqual(originTable, table) ||
      !isEqual(originPrimaryKey, primaryKey) ||
      !isEqual(originForeignKeyList, foreignKeyList)
    ) {
      if (originTable.name !== table.name) {
        renameTableContextList.push({
          schema: schema.name,
          oldName: originTable.name,
          newName: table.name,
        });
      }

      const columnListDiffResult = diffColumnList(
        originTable.columnList,
        table.columnList
      );
      if (
        !isEqual(originPrimaryKey, primaryKey) ||
        !isEqual(originForeignKeyList, foreignKeyList) ||
        columnListDiffResult.addColumnList.length > 0 ||
        columnListDiffResult.changeColumnList.length > 0 ||
        columnListDiffResult.dropColumnList.length > 0
      ) {
        const alterTableContext: AlterTableContext = {
          schema: schema.name,
          name: table.name,
          ...columnListDiffResult,
          dropPrimaryKey: false,
          dropPrimaryKeyList: [],
          dropForeignKeyList: [],
          addForeignKeyList: [],
        };

        // Compose primary key changes.
        if (!isEqual(originPrimaryKey, primaryKey)) {
          const droppedColumnNameList = alterTableContext.dropColumnList.map(
            (column) => column.name
          );
          const filterOriginPrimaryKeyColumnNameList: string[] = [];
          for (const columnId of originPrimaryKey.columnIdList) {
            const column = originTable.columnList.find(
              (column) => column.id === columnId
            );
            if (column && !droppedColumnNameList.includes(column.name)) {
              filterOriginPrimaryKeyColumnNameList.push(column.name);
            }
          }
          const primaryKeyColumnNameList: string[] = [];
          for (const columnId of primaryKey.columnIdList) {
            const column = table.columnList.find(
              (column) => column.id === columnId
            );
            if (column) {
              primaryKeyColumnNameList.push(column.name);
            }
          }

          if (
            !isEqual(
              filterOriginPrimaryKeyColumnNameList,
              primaryKeyColumnNameList
            )
          ) {
            if (originPrimaryKey.columnIdList.length !== 0) {
              alterTableContext.dropPrimaryKey = true;
              alterTableContext.dropPrimaryKeyList.push(originPrimaryKey.name);
            }
            alterTableContext.primaryKeyList = primaryKeyColumnNameList;
          }
        }

        // Compose foreign key changes.
        if (!isEqual(originForeignKeyList, foreignKeyList)) {
          for (const foreignKey of foreignKeyList) {
            const originForeignKey = originForeignKeyList?.find(
              (fk) => fk.name === foreignKey.name
            );

            if (!isEqual(originForeignKey, foreignKey)) {
              if (!isUndefined(originForeignKey)) {
                alterTableContext.dropForeignKeyList.push(
                  originForeignKey.name
                );
              }
              if (
                foreignKey.columnIdList.length > 0 &&
                foreignKey.columnIdList.length ===
                  foreignKey.referencedColumnIdList.length
              ) {
                const referencedSchema = editorStore.getSchema(
                  databaseId,
                  foreignKey.referencedSchemaId
                );
                const referencedTable = referencedSchema?.tableList.find(
                  (table) => table.id === foreignKey.referencedTableId
                );
                if (!referencedSchema || !referencedTable) {
                  continue;
                }

                const columnList: string[] = [];
                const referencedColumnList: string[] = [];
                for (const columnId of foreignKey.columnIdList) {
                  const column = table.columnList.find(
                    (column) => column.id === columnId
                  );
                  const columnIndex = foreignKey.columnIdList.findIndex(
                    (item) => item === column?.id
                  );
                  const referencedColumn = referencedTable?.columnList.find(
                    (column) =>
                      column.id ===
                      foreignKey.referencedColumnIdList[columnIndex]
                  );
                  if (
                    column &&
                    referencedColumn &&
                    column.status !== "dropped"
                  ) {
                    columnList.push(column.name);
                    referencedColumnList.push(referencedColumn.name);
                  }
                }

                alterTableContext.addForeignKeyList.push({
                  columnList,
                  referencedSchema: referencedSchema.name,
                  referencedTable: referencedTable.name,
                  referencedColumnList,
                });
              }
            }
          }
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
      schema: schema.name,
      name: table.name,
    });
  }

  return {
    createSchemaList: createSchemaContextList,
    renameSchemaList: renameSchemaContextList,
    dropSchemaList: dropSchemaContextList,
    createTableList: createTableContextList,
    alterTableList: alterTableContextList,
    renameTableList: renameTableContextList,
    dropTableList: dropTableContextList,
  };
};

export const checkHasSchemaChanges = (diffResult: DiffResult): boolean => {
  return (
    diffResult.createSchemaList.length > 0 ||
    diffResult.renameSchemaList.length > 0 ||
    diffResult.dropSchemaList.length > 0 ||
    diffResult.createTableList.length > 0 ||
    diffResult.alterTableList.length > 0 ||
    diffResult.renameTableList.length > 0 ||
    diffResult.dropTableList.length > 0
  );
};

export const mergeDiffResults = (diffResults: DiffResult[]): DiffResult => {
  return {
    createSchemaList: diffResults.map((item) => item.createSchemaList).flat(),
    renameSchemaList: diffResults.map((item) => item.renameSchemaList).flat(),
    dropSchemaList: diffResults.map((item) => item.dropSchemaList).flat(),
    createTableList: diffResults.map((item) => item.createTableList).flat(),
    alterTableList: diffResults.map((item) => item.alterTableList).flat(),
    renameTableList: diffResults.map((item) => item.renameTableList).flat(),
    dropTableList: diffResults.map((item) => item.dropTableList).flat(),
  };
};
