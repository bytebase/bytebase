import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import type {
  DatabaseCatalog,
  SchemaCatalog,
  TableCatalog,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { TableCatalog_ColumnsSchema } from "@/types/proto-es/v1/database_catalog_service_pb";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
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
    if (tableCatalog.kind?.case !== "columns") {
      tableCatalog.kind = {
        case: "columns",
        value: create(TableCatalog_ColumnsSchema, {}),
      };
    }
    tableCatalog.kind.value.columns = tableCatalog.kind.value.columns.filter(
      (cc) => columnMap.has(cc.name)
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
  const response = cloneDeep(catalog);
  // Remove unused schema catalog
  response.schemas = response.schemas.filter((sc) => schemaMap.has(sc.name));
  // Recursively cleanup table catalog
  response.schemas.forEach((sc) => {
    const schema = schemaMap.get(sc.name)!;
    cleanupTableCatalog(schema, sc);
  });
  // Cleanup empty schema catalog
  response.schemas = response.schemas.filter((sc) => sc.tables.length > 0);
  return response;
};
