export type Column = {
  name: string;
};

export type Table = {
  // If we can parse subquery, a subquery can be
  // recognized as a virtual table (database=undefined).
  database: string | undefined;
  name: string;
  columns: Column[];
};

export type Database = {
  name: string;
  tables: Table[];
  // Views may be supported in the future.
  // But we don't have "columns" info now.
};

export type Schema = {
  databases: Database[];
};
