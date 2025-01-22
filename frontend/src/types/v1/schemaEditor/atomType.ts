import type Long from "long";

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
}

export interface Schema {
  id: string;
  // It should be an empty string for MySQL/TiDB.
  name: string;
  tableList: Table[];
  status: Status;
}
