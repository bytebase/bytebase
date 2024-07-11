import { flatten } from "lodash-es";
import type { TransferOption, TreeOption } from "naive-ui";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { hasSchemaProperty } from "@/utils";

export interface DatabaseTreeOption<L = "database" | "schema" | "table">
  extends TreeOption {
  level: L;
  value: string;
}

export const mapTreeOptions = (
  databaseList: ComposedDatabase[],
  filterValueList?: string[]
) => {
  const databaseNodes: DatabaseTreeOption<"database">[] = [];
  const filteredDatabaseList = filterValueList
    ? databaseList.filter((database) =>
        filterValueList.some(
          (value) => value.split("/schemas/")[0] === database.name
        )
      )
    : databaseList;
  for (const database of filteredDatabaseList) {
    const databaseNode: DatabaseTreeOption<"database"> = {
      level: "database",
      value: database.name,
      label: database.databaseName,
      isLeaf: false,
    };
    databaseNodes.push(databaseNode);

    const childrenNodes = getSchemaOrTableTreeOptions(
      database,
      filterValueList
    );
    if (!childrenNodes) {
      continue;
    }

    if (childrenNodes.length > 0) {
      databaseNode.children = childrenNodes;
      databaseNode.isLeaf = false;
    } else {
      databaseNode.isLeaf = true;
    }
  }
  return databaseNodes;
};

export const getSchemaOrTableTreeOptions = (
  database: ComposedDatabase,
  filterValueList?: string[]
) => {
  const dbSchemaStore = useDBSchemaV1Store();
  const databaseMetadata = dbSchemaStore.getDatabaseMetadataWithoutDefault(
    database.name
  );
  if (!databaseMetadata) {
    return undefined;
  }
  if (hasSchemaProperty(database.instanceResource.engine)) {
    const schemaNodes = databaseMetadata.schemas.map(
      (schema): DatabaseTreeOption<"schema"> => {
        const schemaNode: DatabaseTreeOption<"schema"> = {
          level: "schema",
          value: `${database.name}/schemas/${schema.name}`,
          label: schema.name,
        };
        const tableNodes = schema.tables.map(
          (table): DatabaseTreeOption<"table"> => {
            return {
              level: "table",
              value: `${schemaNode.value}/tables/${table.name}`,
              label: table.name,
            };
          }
        );
        if (tableNodes.length > 0) {
          schemaNode.children = filterValueList
            ? tableNodes.filter((node) => filterValueList.includes(node.value))
            : tableNodes;
        }
        return schemaNode;
      }
    );
    return filterValueList
      ? schemaNodes.filter((node) =>
          filterValueList.some(
            (value) => value.split("/tables/")[0] === node.value
          )
        )
      : schemaNodes;
  } else {
    const tableNodes = flatten(
      databaseMetadata.schemas.map((schema) => schema.tables)
    ).map((table): DatabaseTreeOption<"table"> => {
      return {
        level: "table",
        value: `${database.name}/schemas//tables/${table.name}`,
        label: table.name,
      };
    });
    return filterValueList
      ? tableNodes.filter((node) => filterValueList.includes(node.value))
      : tableNodes;
  }
};

export const flattenTreeOptions = (
  options: DatabaseTreeOption[]
): TransferOption[] => {
  return options.flatMap((option) => {
    return [
      option as TransferOption,
      ...flattenTreeOptions(
        (option.children as DatabaseTreeOption[] | undefined) ?? []
      ),
    ];
  });
};
