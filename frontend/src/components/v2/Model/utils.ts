import { type DataTableColumn, type DataTableSortState } from "naive-ui";

export const mapSorterStatus = <T>(
  columns: DataTableColumn<T>[],
  sorters?: DataTableSortState[]
): DataTableColumn<T>[] => {
  if (!sorters) {
    return columns;
  }

  return columns.map((column) => {
    if (column.type) {
      return column;
    }
    const sorterIndex = sorters.findIndex(
      (s) => s.columnKey === column.key.toString()
    );
    if (sorterIndex < 0) {
      return column;
    }
    return {
      ...column,
      sorter: {
        multiple: sorterIndex,
      },
      sortOrder: sorters[sorterIndex].order,
    };
  });
};
