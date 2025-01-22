import {
  TableCatalog,
  SchemaCatalog,
  TableCatalog_Columns,
  DatabaseCatalog,
} from "@/types/proto/v1/database_catalog_service";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { keyBy } from "@/utils";

export const cleanupUnusedCatalog = (
  metadata: DatabaseMetadata,
  catalog: DatabaseCatalog
) => {
  const cleanupColumnCatalog = (
    table: TableMetadata,
    tableCatalog: TableCatalog
  ) => {
    const columnMap = keyBy(table.columns, (column) => column.name);
    // Remove unused column catalog
    if (!tableCatalog.columns) {
      tableCatalog.columns = TableCatalog_Columns.fromPartial({});
    }
    tableCatalog.columns.columns = tableCatalog.columns?.columns.filter((cc) =>
      columnMap.has(cc.name)
    );
  };
  const cleanupTableCatalog = (
    schema: SchemaMetadata,
    schemaCatalog: SchemaCatalog
  ) => {
    const tableMap = keyBy(schema.tables, (table) => table.name);
    // Remove unused table catalog
    schemaCatalog.tables = schemaCatalog.tables.filter((tc) =>
      tableMap.has(tc.name)
    );
    // Recursively cleanup column catalog
    schemaCatalog.tables.forEach((tc) => {
      cleanupColumnCatalog(tableMap.get(tc.name)!, tc);
    });
  };

  const schemaMap = keyBy(metadata.schemas, (schema) => schema.name);
  // Remove unused schema catalog
  catalog.schemas = catalog.schemas.filter((sc) => schemaMap.has(sc.name));
  // Recursively cleanup table catalog
  catalog.schemas.forEach((sc) => {
    const schema = schemaMap.get(sc.name)!;
    cleanupTableCatalog(schema, sc);
  });
  // Cleanup empty schema catalog
  catalog.schemas = catalog.schemas.filter((sc) => sc.tables.length > 0);
};
