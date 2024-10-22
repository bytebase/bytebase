import { flatten, isUndefined } from "lodash-es";
import type { TransferOption, TreeOption } from "naive-ui";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import type { TableMetadata } from "@/types/proto/v1/database_service";
import { hasSchemaProperty } from "@/utils";

export interface DatabaseTreeOption<
  L = "database" | "schema" | "table" | "column",
> extends TreeOption {
  level: L;
  value: string;
  children?: DatabaseTreeOption[];
}

export const mapTreeOptions = ({
  databaseList,
  filterValueList,
  includeCloumn,
}: {
  databaseList: ComposedDatabase[];
  filterValueList?: string[];
  includeCloumn: boolean;
}) => {
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
      children: getSchemaOrTableTreeOptions({
        database,
        filterValueList,
        includeCloumn,
      }),
    };
    if (!isUndefined(databaseNode.children)) {
      databaseNode.isLeaf = databaseNode.children.length === 0;
    }

    databaseNodes.push(databaseNode);
  }
  return databaseNodes;
};

const getTableTreeOptions = ({
  prefix,
  tableList,
  filterValueList,
  includeCloumn,
}: {
  prefix: string;
  tableList: TableMetadata[];
  filterValueList?: string[];
  includeCloumn: boolean;
}): DatabaseTreeOption<"table">[] => {
  const tableNodes = tableList.map((table): DatabaseTreeOption<"table"> => {
    const option: DatabaseTreeOption<"table"> = {
      level: "table",
      value: `${prefix}/tables/${table.name}`,
      label: table.name,
      isLeaf: true,
    };
    if (includeCloumn) {
      option.children = table.columns.map(
        (column): DatabaseTreeOption<"column"> => ({
          level: "column",
          value: `${option.value}/columns/${column.name}`,
          label: column.name,
          isLeaf: true,
        })
      );
      if (filterValueList) {
        option.children = option.children.filter((node) =>
          filterValueList.includes(node.value)
        );
      }
    }
    if (option.children && option.children.length > 0) {
      option.isLeaf = false;
    }
    return option;
  });
  return filterValueList
    ? tableNodes.filter((node) =>
        filterValueList.some(
          (value) => value.split("/columns/")[0] === node.value
        )
      )
    : tableNodes;
};

export const getSchemaOrTableTreeOptions = ({
  database,
  filterValueList,
  includeCloumn,
}: {
  database: ComposedDatabase;
  filterValueList?: string[];
  includeCloumn: boolean;
}) => {
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
        const value = `${database.name}/schemas/${schema.name}`;
        const schemaNode: DatabaseTreeOption<"schema"> = {
          level: "schema",
          value,
          label: schema.name,
          children: getTableTreeOptions({
            prefix: value,
            tableList: schema.tables,
            filterValueList,
            includeCloumn,
          }),
          isLeaf: true,
        };
        if (schemaNode.children && schemaNode.children.length > 0) {
          schemaNode.isLeaf = false;
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
    return getTableTreeOptions({
      prefix: `${database.name}/schemas/`,
      tableList: flatten(
        databaseMetadata.schemas.map((schema) => schema.tables)
      ),
      filterValueList,
      includeCloumn,
    });
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
