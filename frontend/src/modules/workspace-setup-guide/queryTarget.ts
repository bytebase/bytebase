import type {
  DatabaseCatalog,
  ObjectSchema,
  TableCatalog,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { GuideQueryTarget } from "./types";

const objectSchemaHasSemanticType = (
  schema: ObjectSchema | undefined
): boolean => {
  if (!schema) return false;
  if (schema.semanticType) return true;
  if (schema.kind.case === "structKind") {
    return Object.values(schema.kind.value.properties).some(
      objectSchemaHasSemanticType
    );
  }
  if (schema.kind.case === "arrayKind") {
    return objectSchemaHasSemanticType(schema.kind.value.kind);
  }
  return false;
};

export const tableHasMarkedSensitiveData = (
  table: TableCatalog | undefined
): boolean => {
  if (table?.kind.case === "columns") {
    return table.kind.value.columns.some((column) => !!column.semanticType);
  }
  if (table?.kind.case === "objectSchema") {
    return objectSchemaHasSemanticType(table.kind.value);
  }
  return false;
};

export const findGuideQueryTarget = (
  metadata: Pick<DatabaseMetadata, "schemas"> | undefined,
  sensitiveCatalog?: Pick<DatabaseCatalog, "schemas">
): GuideQueryTarget | undefined => {
  for (const schema of metadata?.schemas ?? []) {
    for (const table of schema.tables) {
      if (sensitiveCatalog) {
        const catalogTable = sensitiveCatalog.schemas
          .find(({ name }) => name === schema.name)
          ?.tables.find(({ name }) => name === table.name);
        if (!tableHasMarkedSensitiveData(catalogTable)) continue;
      }
      return { schema: schema.name, table: table.name };
    }
  }
  return undefined;
};
