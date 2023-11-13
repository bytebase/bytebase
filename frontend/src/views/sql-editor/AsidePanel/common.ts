import { useDBSchemaV1Store } from "@/store";
import { mapTreeNodeByType } from "@/store/modules/sqlEditorTree";
import { ComposedDatabase, SQLEditorTreeNode } from "@/types";
import {
  SchemaMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";

const createDummyTableNode = (
  parent: SQLEditorTreeNode,
  error: unknown | undefined = undefined
) => {
  return mapTreeNodeByType(
    "dummy",
    {
      type: "table",
      error,
    },
    parent,
    {
      disabled: true,
    }
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
    return [createDummyTableNode(parent)];
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
        view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
      });

    const { schemas } = databaseMetadata;
    if (schemas.length === 0) {
      // Empty database
      node.children = [createDummyTableNode(node)];
      return;
    }

    if (schemas.length === 1 && schemas[0].name === "") {
      const schema = schemas[0];
      // A single schema database, should render tables directly as a database
      // node's children
      node.children = mapTableNodes(database, schema, node);
      return;
    } else {
      // Multiple schema database
      node.children = schemas.map((schema) => {
        const schemaNode = mapTreeNodeByType(
          "schema",
          { database, schema },
          node
        );
        schemaNode.children = mapTableNodes(database, schema, schemaNode);
        return schemaNode;
      });
      return;
    }
  } catch (error) {
    node.children = [createDummyTableNode(node, error)];
  }
};
