import { useDBSchemaV1Store } from "@/store";
import { mapTreeNodeByType } from "@/store/modules/sqlEditorTree";
import { SQLEditorTreeNode } from "@/types";

export const fetchDatabaseSubTree = async (
  node: SQLEditorTreeNode<"database">
) => {
  const db = node.meta.target;
  const databaseMetadata =
    await useDBSchemaV1Store().getOrFetchDatabaseMetadata(db.name);
  const { schemas } = databaseMetadata;
  if (schemas.length === 0) {
    // Empty database
    node.children = [];
    return;
  }

  if (schemas.length === 1 && schemas[0].name === "") {
    // A single schema database, should render tables directly as a database
    // node's children
    node.children = schemas[0].tables.map((table) =>
      mapTreeNodeByType("table", table, node)
    );
    return;
  } else {
    // Multiple schema database
    node.children = schemas.map((schema) => {
      const schemaNode = mapTreeNodeByType("schema", schema, node);
      schemaNode.children = schema.tables.map((table) =>
        mapTreeNodeByType("table", table, schemaNode)
      );
      return schemaNode;
    });
    return;
  }
};
