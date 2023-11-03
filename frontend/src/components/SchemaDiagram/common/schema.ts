import {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

export const findPrimaryKey = (table: TableMetadata) => {
  return table.indexes.find((idx) => idx.primary);
};

export const isPrimaryKey = (table: TableMetadata, column: ColumnMetadata) => {
  const pk = findPrimaryKey(table);
  if (!pk) return false;
  return pk.expressions.includes(column.name);
};

export const findIndexes = (table: TableMetadata, column: ColumnMetadata) => {
  return table.indexes.filter(
    (idx) => !idx.primary && idx.expressions.includes(column.name)
  );
};

export const isIndex = (table: TableMetadata, column: ColumnMetadata) => {
  return findIndexes(table, column).length > 0;
};
