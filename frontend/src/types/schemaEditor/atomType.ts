import { isGhostTable } from "@/utils";
import { v1 as uuidv1 } from "uuid";
import {
  SchemaMetadata,
  TableMetadata,
  ColumnMetadata,
} from "../proto/v1/database_service";

type AtomResourceStatus = "normal" | "created" | "dropped";

export interface Column {
  id: string;
  name: string;
  type: string;
  nullable: boolean;
  comment: string;
  userComment: string;
  default?: string;
  status: AtomResourceStatus;
}

export interface PrimaryKey {
  name: string;
  columnIdList: string[];
}

export interface Table {
  id: string;
  name: string;
  engine: string;
  collation: string;
  rowCount: number;
  dataSize: number;
  comment: string;
  columnList: Column[];
  // Including column id list.
  primaryKey: PrimaryKey;
  status: AtomResourceStatus;
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

export interface Schema {
  id: string;
  // It should be an empty string for MySQL/TiDB.
  name: string;
  tableList: Table[];
  foreignKeyList: ForeignKey[];
  status: AtomResourceStatus;
}

export const convertColumnMetadataToColumn = (
  columnMetadata: ColumnMetadata
): Column => {
  return {
    id: uuidv1(),
    name: columnMetadata.name,
    type: columnMetadata.type,
    nullable: columnMetadata.nullable,
    comment: columnMetadata.comment,
    userComment: columnMetadata.userComment,
    default: columnMetadata.default,
    status: "normal",
  };
};

export const convertTableMetadataToTable = (
  tableMetadata: TableMetadata
): Table => {
  const table: Table = {
    id: uuidv1(),
    name: tableMetadata.name,
    engine: tableMetadata.engine,
    collation: tableMetadata.collation,
    rowCount: tableMetadata.rowCount,
    dataSize: tableMetadata.dataSize,
    comment: tableMetadata.comment,
    columnList: tableMetadata.columns.map((column) =>
      convertColumnMetadataToColumn(column)
    ),
    primaryKey: {
      name: "",
      columnIdList: [],
    },
    status: "normal",
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
  schemaMetadata: SchemaMetadata
): Schema => {
  const tableList: Table[] = [];

  for (const tableMetadata of schemaMetadata.tables) {
    // Don't display ghost table in Schema Editor.
    if (isGhostTable(tableMetadata)) {
      continue;
    }

    const table = convertTableMetadataToTable(tableMetadata);
    tableList.push(table);
  }

  return {
    id: uuidv1(),
    name: schemaMetadata.name,
    tableList: tableList,
    foreignKeyList: [],
    status: "normal",
  };
};

export const convertSchemaMetadataList = (
  schemaMetadataList: SchemaMetadata[]
) => {
  // Compose all tables of each schema.
  const schemaList: Schema[] = schemaMetadataList.map((schemaMetadata) =>
    convertSchemaMetadataToSchema(schemaMetadata)
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
    const foreignKeyList: ForeignKey[] = [];
    for (const tableMetadata of schemaMetadata.tables) {
      const table = tableList.find(
        (table) => table.name === tableMetadata.name
      );
      if (!table) {
        continue;
      }

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
    }
    schema.foreignKeyList = foreignKeyList;
  }

  return schemaList;
};
