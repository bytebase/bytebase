import { flatten } from "lodash-es";
import { TransferOption, TreeOption } from "naive-ui";
import { useDBSchemaV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
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
  const dbSchemaStore = useDBSchemaV1Store();
  const databaseNodes: DatabaseTreeOption<"database">[] = [];
  for (const database of databaseList) {
    const databaseMetadata = dbSchemaStore.getDatabaseMetadata(database.name);
    const databaseNode: DatabaseTreeOption<"database"> = {
      level: "database",
      value: `d-${database.uid}`,
      label: database.databaseName,
      checkboxDisabled: true,
    };

    if (hasSchemaProperty(database.instanceEntity.engine)) {
      const schemaNodes = databaseMetadata.schemas.map(
        (schema): DatabaseTreeOption<"schema"> => {
          const schemaNode: DatabaseTreeOption<"schema"> = {
            level: "schema",
            value: `s-${database.uid}-${schema.name}`,
            label: schema.name,
            checkboxDisabled: true,
          };
          const tableNodes = schema.tables.map(
            (table): DatabaseTreeOption<"table"> => {
              return {
                level: "table",
                value: `t-${database.uid}-${schema.name}-${table.name}`,
                label: table.name,
              };
            }
          );
          if (tableNodes.length > 0) {
            schemaNode.children = filterValueList
              ? tableNodes.filter(
                  (node) =>
                    (node.children && node.children?.length > 0) ||
                    filterValueList.includes(node.value)
                )
              : tableNodes;
          }
          return schemaNode;
        }
      );
      if (schemaNodes.length > 0) {
        databaseNode.children = filterValueList
          ? schemaNodes.filter(
              (node) =>
                (node.children && node.children?.length > 0) ||
                filterValueList.includes(node.value)
            )
          : schemaNodes;
      }
    } else {
      const tableNodes = flatten(
        databaseMetadata.schemas.map((schema) => schema.tables)
      ).map((table): DatabaseTreeOption<"table"> => {
        return {
          level: "table",
          value: `t-${database.uid}--${table.name}`,
          label: table.name,
        };
      });
      if (tableNodes.length > 0) {
        databaseNode.children = filterValueList
          ? tableNodes.filter(
              (node) =>
                (node.children && node.children?.length > 0) ||
                filterValueList.includes(node.value)
            )
          : tableNodes;
      }
    }
    databaseNodes.push(databaseNode);
  }
  return filterValueList
    ? databaseNodes.filter(
        (node) =>
          (node.children && node.children?.length > 0) ||
          filterValueList.includes(node.value)
      )
    : databaseNodes;
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
