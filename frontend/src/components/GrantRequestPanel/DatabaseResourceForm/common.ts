import { flatten, isUndefined } from "lodash-es";
import type { TransferOption, TreeOption } from "naive-ui";
import { useDBSchemaV1Store } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import type { ComposedDatabase, DatabaseResource } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { TableMetadata } from "@/types/proto-es/v1/database_service_pb";
import { hasSchemaProperty } from "@/utils";

export type DatabaseResourceType =
  | "databases"
  | "schemas"
  | "tables"
  | "columns";

export interface DatabaseTreeOption<L = DatabaseResourceType>
  extends TreeOption,
    TransferOption {
  label: string;
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
  const databaseNodes: DatabaseTreeOption<"databases">[] = [];
  const filteredDatabaseList = filterValueList
    ? databaseList.filter((database) =>
        filterValueList.some(
          (value) => value.split("/schemas/")[0] === database.name
        )
      )
    : databaseList;
  for (const database of filteredDatabaseList) {
    const databaseNode: DatabaseTreeOption<"databases"> = {
      level: "databases",
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
}): DatabaseTreeOption<"tables">[] => {
  const tableNodes = tableList.map((table): DatabaseTreeOption<"tables"> => {
    const option: DatabaseTreeOption<"tables"> = {
      level: "tables",
      value: `${prefix}/tables/${table.name}`,
      label: table.name,
      isLeaf: true,
    };
    if (includeCloumn) {
      option.children = table.columns.map(
        (column): DatabaseTreeOption<"columns"> => ({
          level: "columns",
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
  if (database.instanceResource.engine === Engine.MONGODB) {
    // do not support table level select for MongoDB.
    return [];
  }
  const dbSchemaStore = useDBSchemaV1Store();
  const databaseMetadata = dbSchemaStore.getDatabaseMetadataWithoutDefault(
    database.name
  );
  if (!databaseMetadata) {
    return undefined;
  }
  if (hasSchemaProperty(database.instanceResource.engine)) {
    const schemaNodes = databaseMetadata.schemas.map(
      (schema): DatabaseTreeOption<"schemas"> => {
        const value = `${database.name}/schemas/${schema.name}`;
        const schemaNode: DatabaseTreeOption<"schemas"> = {
          level: "schemas",
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
): DatabaseTreeOption[] => {
  return options.flatMap((option) => {
    return [option, ...flattenTreeOptions(option.children ?? [])];
  });
};

export const parseStringToResource = (
  key: string
): DatabaseResource | undefined => {
  // The key should in instances/{instance resource id}/databases/{database resource id}/schemas/{schema}/tables/{table}/columns/{column}
  const sections = key.split("/");
  const resource: DatabaseResource = {
    databaseFullName: "",
  };

  while (sections.length > 0) {
    const keyword = sections.shift() as DatabaseResourceType | "instances";
    const data = sections.shift() || "";

    switch (keyword) {
      case "instances": {
        resource.instanceResourceId = data;
        if (resource.databaseResourceId) {
          resource.databaseFullName = `${instanceNamePrefix}${resource.instanceResourceId}/${databaseNamePrefix}${resource.databaseResourceId}`;
        }
        break;
      }
      case "databases": {
        resource.databaseResourceId = data;
        if (resource.instanceResourceId) {
          resource.databaseFullName = `${instanceNamePrefix}${resource.instanceResourceId}/${databaseNamePrefix}${resource.databaseResourceId}`;
        }
        break;
      }
      case "schemas":
        resource.schema = data;
        break;
      case "tables":
        resource.table = data;
        break;
      case "columns":
        if (data) {
          resource.columns = [data];
        }
        break;
      default:
        return;
    }
  }

  if (!resource.databaseFullName) {
    return;
  }

  return resource;
};
