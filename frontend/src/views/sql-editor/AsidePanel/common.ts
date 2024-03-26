import type { RenderFunction } from "vue";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useCurrentUserV1,
  useDBSchemaV1Store,
  mapTreeNodeByType,
} from "@/store";
import type {
  ComposedDatabase,
  SQLEditorTreeNode,
  SQLEditorTreeNodeType,
  TextTarget,
} from "@/types";
import type {
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
} from "@/types/proto/v1/database_service";
import { hasProjectPermissionV2 } from "@/utils";

const createDummyNode = (
  type: "table" | "view",
  parent: SQLEditorTreeNode,
  error: unknown | undefined = undefined
) => {
  return mapTreeNodeByType(
    "dummy",
    {
      id: parent.key,
      type,
      error,
    },
    parent,
    {
      disabled: true,
    }
  );
};
const createExpandableTextNode = (
  type: SQLEditorTreeNodeType,
  parent: SQLEditorTreeNode,
  text: TextTarget<true>["text"],
  render?: RenderFunction
) => {
  return mapTreeNodeByType(
    "expandable-text",
    {
      id: parent.key,
      type,
      expandable: true,
      text,
      render,
    },
    parent
  );
};

const mapTableNodes = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  parent: SQLEditorTreeNode
) => {
  const children = schema.tables.map((table) => {
    const node = mapTreeNodeByType(
      "table",
      { database, schema, table },
      parent
    );

    // Map table-level partitions.
    if (table.partitions.length > 0) {
      node.isLeaf = false;
      node.children = [];
      for (const partition of table.partitions) {
        const subnode = mapTreeNodeByType(
          "partition-table",
          { database, schema, table, partition },
          node
        );
        if (partition.subpartitions.length > 0) {
          subnode.isLeaf = false;
          subnode.children = mapPartitionTableNodes(
            database,
            schema,
            table,
            partition,
            subnode
          );
        } else {
          subnode.isLeaf = true;
        }
        node.children.push(subnode);
      }
    } else {
      node.isLeaf = true;
    }
    return node;
  });
  if (children.length === 0) {
    return [createDummyNode("table", parent)];
  }
  return children;
};

const mapExternalTableNodes = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  parent: SQLEditorTreeNode
) => {
  const children = schema.externalTables.map((externalTable) => {
    const node = mapTreeNodeByType(
      "external-table",
      { database, schema, externalTable },
      parent
    );

    node.isLeaf = true;
    return node;
  });
  return children;
};

// Map partition-table-level partitions.
const mapPartitionTableNodes = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  table: TableMetadata,
  parentPartition: TablePartitionMetadata,
  parent: SQLEditorTreeNode
) => {
  const children = parentPartition.subpartitions.map((partition) => {
    const node = mapTreeNodeByType(
      "partition-table",
      {
        database,
        schema,
        table,
        parentPartition,
        partition: partition,
      },
      parent
    );
    if (partition.subpartitions.length > 0) {
      node.isLeaf = false;
      node.children = mapPartitionTableNodes(
        database,
        schema,
        table,
        partition,
        node
      );
    } else {
      node.isLeaf = true;
    }
    return node;
  });
  return children;
};

const mapViewNodes = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  parent: SQLEditorTreeNode
) => {
  const children = schema.views.map((view) =>
    mapTreeNodeByType(
      "view",
      {
        database,
        schema,
        view,
      },
      parent
    )
  );
  if (children.length === 0) {
    return [createDummyNode("view", parent)];
  }
  return children;
};

const buildSchemaNodeChildren = (
  database: ComposedDatabase,
  schema: SchemaMetadata,
  parent: SQLEditorTreeNode
) => {
  if (schema.tables.length === 0 && schema.views.length === 0) {
    return [createDummyNode("table", parent)];
  }

  const children: SQLEditorTreeNode[] = [];

  // Always show "Tables" node
  // If no tables, show "<Empty>"
  const tablesNode = createExpandableTextNode("table", parent, () =>
    t("db.tables")
  );
  tablesNode.children = mapTableNodes(database, schema, tablesNode);
  children.push(tablesNode);

  // Only show "External Tables" node if the schema do have external tables.
  if (schema.externalTables.length > 0) {
    const externalTablesNode = createExpandableTextNode(
      "external-table",
      parent,
      () => t("db.external-tables")
    );
    externalTablesNode.children = mapExternalTableNodes(
      database,
      schema,
      externalTablesNode
    );
    children.push(externalTablesNode);
  }

  // Only show "Views" node if the schema do have views.
  if (schema.views.length > 0) {
    const viewsNode = createExpandableTextNode("view", parent, () =>
      t("db.views")
    );
    viewsNode.children = mapViewNodes(database, schema, viewsNode);
    children.push(viewsNode);
  }
  return children;
};

export const fetchDatabaseSubTree = async (
  node: SQLEditorTreeNode<"database">
) => {
  const me = useCurrentUserV1();
  if (
    !hasProjectPermissionV2(
      node.meta.target.projectEntity,
      me.value,
      "bb.databases.getSchema"
    )
  ) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("sql-editor.missing-permission-to-load-db-metadata", {
        db: node.meta.target.databaseName,
      }),
    });
    node.children = [];
    node.isLeaf = true;
    return;
  }

  try {
    const database = node.meta.target;
    const databaseMetadata =
      await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
        database: database.name,
        skipCache: false,
      });

    const { schemas } = databaseMetadata;
    if (schemas.length === 0) {
      // Empty database, show "<Empty>"
      node.children = [createDummyNode("table", node)];
      return;
    }

    if (schemas.length === 1 && schemas[0].name === "") {
      const schema = schemas[0];
      // A single schema database, should render tables as views directly as a database
      // node's children
      node.children = buildSchemaNodeChildren(database, schema, node);
      return;
    } else {
      // Multiple schema database
      node.children = schemas.map((schema) => {
        const schemaNode = mapTreeNodeByType(
          "schema",
          { database, schema },
          node
        );

        schemaNode.children = buildSchemaNodeChildren(database, schema, node);
        return schemaNode;
      });
      return;
    }
  } catch (error) {
    console.warn("[fetchDatabaseSubTree]", error);
    node.children = [createDummyNode("table", node, error)];
  }
};
