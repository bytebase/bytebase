import Long from "long";
import { v1 as uuidv1 } from "uuid";
import { markRaw } from "vue";
import {
  SchemaMetadata,
  TableMetadata,
  ColumnMetadata,
  ColumnConfig,
  TableConfig,
  SchemaConfig,
} from "@/types/proto/v1/database_service";
import { isGhostTable } from "@/utils";

type Status = "normal" | "created" | "dropped";

export interface ColumnDefaultValue {
  hasDefault: boolean;
  defaultNull?: boolean;
  defaultString?: string;
  defaultExpression?: string;
}
export interface Column extends ColumnDefaultValue {
  id: string;
  name: string;
  type: string;
  nullable: boolean;
  comment: string;
  userComment: string;
  status: Status;
  classification?: string;
  config: ColumnConfig;
}

export interface PrimaryKey {
  name: string;
  columnIdList: string[];
}

export interface ForeignKey {
  // Should be an unique name.
  name: string;
  tableId: string;
  columnIdList: string[];
  referencedSchemaId: string;
  referencedTableId: string;
  referencedColumnIdList: string[];
}

export interface Table {
  id: string;
  name: string;
  engine: string;
  collation: string;
  rowCount: Long;
  dataSize: Long;
  comment: string;
  userComment: string;
  columnList: Column[];
  primaryKey: PrimaryKey;
  foreignKeyList: ForeignKey[];
  status: Status;
  classification?: string;
}

export interface Schema {
  id: string;
  // It should be an empty string for MySQL/TiDB.
  name: string;
  tableList: Table[];
  status: Status;
}

export const convertColumnMetadataToColumn = (
  columnMetadata: ColumnMetadata,
  status: Status,
  config: ColumnConfig | undefined = undefined
): Column => {
  return {
    id: uuidv1(),
    name: columnMetadata.name,
    type: columnMetadata.type,
    nullable: columnMetadata.nullable,
    comment: columnMetadata.comment,
    userComment: columnMetadata.userComment,
    hasDefault: columnMetadata.hasDefault,
    defaultNull: columnMetadata.defaultNull,
    defaultString: columnMetadata.defaultString,
    defaultExpression: columnMetadata.defaultExpression,
    classification: columnMetadata.classification,
    status,
    config: ColumnConfig.fromPartial({
      ...(config ?? {}),
      name: columnMetadata.name,
    }),
  };
};

export const convertTableMetadataToTable = (
  tableMetadata: TableMetadata,
  status: Status,
  config: TableConfig = TableConfig.fromPartial({})
): Table => {
  const table: Table = {
    id: uuidv1(),
    name: tableMetadata.name,
    engine: tableMetadata.engine,
    collation: tableMetadata.collation,
    rowCount: tableMetadata.rowCount,
    dataSize: tableMetadata.dataSize,
    comment: tableMetadata.comment,
    userComment: tableMetadata.userComment,
    classification: tableMetadata.classification,
    columnList: tableMetadata.columns.map((column) =>
      convertColumnMetadataToColumn(
        column,
        status,
        config.columnConfigs.find(
          (columnConfig) => columnConfig.name === column.name
        )
      )
    ),
    primaryKey: {
      name: "",
      columnIdList: [],
    },
    foreignKeyList: [],
    status,
  };

  for (const indexMetadata of tableMetadata.indexes) {
    if (indexMetadata.primary === true) {
      table.primaryKey.name = indexMetadata.name;
      for (const columnName of indexMetadata.expressions) {
        const column = table.columnList.find(
          (column) => column.name === columnName
        );
        if (column) {
          table.primaryKey.columnIdList.push(column.id);
        }
      }
      break;
    }
  }

  return table;
};

export const convertSchemaMetadataToSchema = (
  schemaMetadata: SchemaMetadata,
  status: Status,
  config: SchemaConfig = SchemaConfig.fromPartial({})
): Schema => {
  const tableList: Table[] = [];

  for (const tableMetadata of schemaMetadata.tables) {
    // Don't display ghost table in Schema Editor.
    if (isGhostTable(tableMetadata)) {
      continue;
    }

    const tableConfig = config.tableConfigs.find(
      (tableConfig) => tableConfig.name === tableMetadata.name
    );

    const table = convertTableMetadataToTable(
      tableMetadata,
      status,
      tableConfig
    );
    tableList.push(table);
  }

  return {
    id: uuidv1(),
    name: schemaMetadata.name,
    tableList: tableList,
    status,
  };
};

export const convertSchemaMetadataList = (
  schemaMetadataList: SchemaMetadata[],
  schemaConfigList: SchemaConfig[]
): Schema[] => {
  // Compose all tables of each schema.
  const schemaList: Schema[] = schemaMetadataList.map((schemaMetadata) =>
    convertSchemaMetadataToSchema(
      schemaMetadata,
      "normal",
      schemaConfigList.find(
        (schemaConfig) => schemaConfig.name === schemaMetadata.name
      )
    )
  );

  // Build foreign keys for schema and referenced schema.
  for (const schemaMetadata of schemaMetadataList) {
    const schema = schemaList.find(
      (schema) => schema.name === schemaMetadata.name
    );
    if (!schema) {
      continue;
    }

    const tableList = schema.tableList;
    for (const tableMetadata of schemaMetadata.tables) {
      const table = tableList.find(
        (table) => table.name === tableMetadata.name
      );
      if (!table) {
        continue;
      }

      const foreignKeyList: ForeignKey[] = [];
      for (const foreignKeyMetadata of tableMetadata.foreignKeys) {
        const referencedSchema = schemaList.find(
          (schema) => schema.name === foreignKeyMetadata.referencedSchema
        );
        const referencedTable = referencedSchema?.tableList.find(
          (table) => table.name === foreignKeyMetadata.referencedTable
        );
        if (!referencedSchema || !referencedTable) {
          continue;
        }

        const fk: ForeignKey = {
          name: foreignKeyMetadata.name,
          tableId: table.id,
          columnIdList: [],
          referencedSchemaId: referencedSchema.id,
          referencedTableId: referencedTable.id,
          referencedColumnIdList: [],
        };
        for (const columnName of foreignKeyMetadata.columns) {
          const column = table.columnList.find(
            (column) => column.name === columnName
          );
          if (column) {
            fk.columnIdList.push(column.id);
          }
        }
        for (const referencedColumnName of foreignKeyMetadata.referencedColumns) {
          const referencedColumn = referencedTable.columnList.find(
            (column) => column.name === referencedColumnName
          );
          if (referencedColumn) {
            fk.referencedColumnIdList.push(referencedColumn.id);
          }
        }
        foreignKeyList.push(fk);
      }
      table.foreignKeyList = foreignKeyList;
    }
  }

  return markRaw(schemaList);
};
