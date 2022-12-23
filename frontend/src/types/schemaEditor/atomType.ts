import { ref, Ref } from "vue";
import {
  ColumnMetadata,
  SchemaMetadata,
  TableMetadata,
} from "../proto/database";

type TableOrColumnStatus = "normal" | "created" | "dropped";

export interface Column {
  oldName: string;
  newName: string;
  type: string;
  nullable: boolean;
  comment: string;
  default?: string;
  status: TableOrColumnStatus;
}

export interface Table {
  oldName: string;
  newName: string;
  engine: string;
  collation: string;
  rowCount: number;
  dataSize: number;
  comment: string;
  columnList: Column[];
  status: TableOrColumnStatus;
}

export interface PrimaryKey {
  table: string;
  // column is a ref to Column.
  columnList: Ref<Column>[];
}

export interface ForeignKey {
  schema: string;
  table: string;
  columnList: string[];
  referencedSchema: string;
  referencedTable: string;
  referencedColumns: string[];
}

export interface Schema {
  // It should be an empty string for MySQL/TiDB.
  name: string;
  tableList: Table[];
  primaryKeyList: PrimaryKey[];
  foreignKeyList: ForeignKey[];
}

export const convertColumnMetadataToColumn = (
  columnMetadata: ColumnMetadata
): Column => {
  return {
    oldName: columnMetadata.name,
    newName: columnMetadata.name,
    type: columnMetadata.type,
    nullable: columnMetadata.nullable,
    comment: columnMetadata.comment,
    default: columnMetadata.default,
    status: "normal",
  };
};

export const convertTableMetadataToTable = (
  tableMetadata: TableMetadata
): Table => {
  return {
    oldName: tableMetadata.name,
    newName: tableMetadata.name,
    engine: tableMetadata.engine,
    collation: tableMetadata.collation,
    rowCount: tableMetadata.rowCount,
    dataSize: tableMetadata.dataSize,
    comment: tableMetadata.comment,
    columnList: tableMetadata.columns.map((column) =>
      convertColumnMetadataToColumn(column)
    ),
    status: "normal",
  };
};

export const convertSchemaMetadataToSchema = (
  schemaMetadata: SchemaMetadata
): Schema => {
  const tableList: Table[] = [];
  const primaryKeyList: PrimaryKey[] = [];
  const foreignKeyList: ForeignKey[] = [];

  for (const tableMetadata of schemaMetadata.tables) {
    const table = convertTableMetadataToTable(tableMetadata);
    tableList.push(table);

    const primaryKey: PrimaryKey = {
      table: tableMetadata.name,
      columnList: [],
    };
    for (const indexMetadata of tableMetadata.indexes) {
      if (indexMetadata.primary === true) {
        for (const columnName of indexMetadata.expressions) {
          const column = table.columnList.find(
            (column) => column.oldName === columnName
          );
          if (column) {
            primaryKey.columnList.push(ref(column));
          }
        }
        break;
      }
    }
    primaryKeyList.push(primaryKey);

    for (const foreignKeyMetadata of tableMetadata.foreignKeys) {
      foreignKeyList.push({
        schema: schemaMetadata.name,
        table: tableMetadata.name,
        columnList: foreignKeyMetadata.columns,
        referencedSchema: foreignKeyMetadata.referencedSchema,
        referencedTable: foreignKeyMetadata.referencedTable,
        referencedColumns: foreignKeyMetadata.referencedColumns,
      });
    }
  }

  return {
    name: schemaMetadata.name,
    tableList: tableList,
    primaryKeyList: primaryKeyList,
    foreignKeyList: foreignKeyList,
  };
};
