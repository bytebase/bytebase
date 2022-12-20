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
  originColumnList: Column[];

  status: TableOrColumnStatus;
}
