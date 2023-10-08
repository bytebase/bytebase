export interface AffectedTable {
  schema: string;
  table: string;
  dropped: boolean;
}

export const EmptyAffectedTable: AffectedTable = {
  schema: "",
  table: "",
  dropped: false,
};
