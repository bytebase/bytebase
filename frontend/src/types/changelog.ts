export interface Table {
  schema: string;
  table: string;
}

export interface AffectedTable extends Table {
  dropped: boolean;
}

export const EmptyAffectedTable: AffectedTable = {
  schema: "",
  table: "",
  dropped: false,
};

export interface SearchChangeLogParams {
  tables?: Table[];
  types?: string[];
}
