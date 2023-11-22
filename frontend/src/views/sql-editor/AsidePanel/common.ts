import { RenderFunction } from "vue";
import { t } from "@/plugins/i18n";
import { useDBSchemaV1Store } from "@/store";
import { mapTreeNodeByType } from "@/store/modules/sqlEditorTree";
import {
  ComposedDatabase,
  SQLEditorTreeNode,
  SQLEditorTreeNodeType,
  TextTarget,
} from "@/types";
import {
  SchemaMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";

const createDummyNode = (
  type: "table" | "view",
  parent: SQLEditorTreeNode,
  error: unknown | undefined = undefined
) => {
  return mapTreeNodeByType(
    "dummy",
    {
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
  const children = schema.tables.map((table) =>
    mapTreeNodeByType("table", { database, schema, table }, parent)
  );
  if (children.length === 0) {
    return [createDummyNode("table", parent)];
  }
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

  // Only show "Views" node if the schema do have views
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
  try {
    const database = node.meta.target;
    const databaseMetadata =
      await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
        database: database.name,
        skipCache: false,
        view: DatabaseMetadataView.DATABASE_METADATA_VIEW_BASIC,
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
