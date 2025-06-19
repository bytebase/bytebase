import { fromJson, toJson } from "@bufbuild/protobuf";
import type { 
  DatabaseCatalog as OldDatabaseCatalog,
  TableCatalog as OldTableCatalog,
  ColumnCatalog as OldColumnCatalog,
  TableCatalog_Columns as OldTableCatalog_Columns
} from "@/types/proto/v1/database_catalog_service";
import { 
  DatabaseCatalog as OldDatabaseCatalogProto,
  TableCatalog as OldTableCatalogProto,
  ColumnCatalog as OldColumnCatalogProto,
  TableCatalog_Columns as OldTableCatalog_ColumnsProto
} from "@/types/proto/v1/database_catalog_service";
import type { 
  DatabaseCatalog as NewDatabaseCatalog,
  TableCatalog as NewTableCatalog,
  ColumnCatalog as NewColumnCatalog,
  TableCatalog_Columns as NewTableCatalog_Columns
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { 
  DatabaseCatalogSchema,
  TableCatalogSchema,
  ColumnCatalogSchema,
  TableCatalog_ColumnsSchema
} from "@/types/proto-es/v1/database_catalog_service_pb";

// Convert old proto to proto-es
export const convertOldDatabaseCatalogToNew = (oldCatalog: OldDatabaseCatalog): NewDatabaseCatalog => {
  const json = OldDatabaseCatalogProto.toJSON(oldCatalog) as any;
  return fromJson(DatabaseCatalogSchema, json);
};

// Convert proto-es to old proto
export const convertNewDatabaseCatalogToOld = (newCatalog: NewDatabaseCatalog): OldDatabaseCatalog => {
  const json = toJson(DatabaseCatalogSchema, newCatalog);
  return OldDatabaseCatalogProto.fromJSON(json);
};

export const convertOldTableCatalogToNew = (oldTable: OldTableCatalog): NewTableCatalog => {
  const json = OldTableCatalogProto.toJSON(oldTable) as any;
  return fromJson(TableCatalogSchema, json);
};

export const convertNewTableCatalogToOld = (newTable: NewTableCatalog): OldTableCatalog => {
  const json = toJson(TableCatalogSchema, newTable);
  return OldTableCatalogProto.fromJSON(json);
};

export const convertOldColumnCatalogToNew = (oldColumn: OldColumnCatalog): NewColumnCatalog => {
  const json = OldColumnCatalogProto.toJSON(oldColumn) as any;
  return fromJson(ColumnCatalogSchema, json);
};

export const convertNewColumnCatalogToOld = (newColumn: NewColumnCatalog): OldColumnCatalog => {
  const json = toJson(ColumnCatalogSchema, newColumn);
  return OldColumnCatalogProto.fromJSON(json);
};

export const convertOldTableCatalogColumnsToNew = (oldColumns: OldTableCatalog_Columns): NewTableCatalog_Columns => {
  const json = OldTableCatalog_ColumnsProto.toJSON(oldColumns) as any;
  return fromJson(TableCatalog_ColumnsSchema, json);
};

export const convertNewTableCatalogColumnsToOld = (newColumns: NewTableCatalog_Columns): OldTableCatalog_Columns => {
  const json = toJson(TableCatalog_ColumnsSchema, newColumns);
  return OldTableCatalog_ColumnsProto.fromJSON(json);
};