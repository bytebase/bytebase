import type { TreeNode } from "../schemaTree";
import { CheckNode } from "./CheckNode";
import { ColumnNode } from "./ColumnNode";
import { DatabaseNode } from "./DatabaseNode";
import { DependencyColumnNode } from "./DependencyColumnNode";
import { DummyNode } from "./DummyNode";
import { ExternalTableNode } from "./ExternalTableNode";
import { ForeignKeyNode } from "./ForeignKeyNode";
import { FunctionNode } from "./FunctionNode";
import { IndexNode } from "./IndexNode";
import { PackageNode } from "./PackageNode";
import { PartitionTableNode } from "./PartitionTableNode";
import { ProcedureNode } from "./ProcedureNode";
import { SchemaNode } from "./SchemaNode";
import { SequenceNode } from "./SequenceNode";
import { TableNode } from "./TableNode";
import { TextNode } from "./TextNode";
import { TriggerNode } from "./TriggerNode";
import { ViewNode } from "./ViewNode";

type Props = {
  readonly node: TreeNode;
  readonly keyword: string;
};

/**
 * Replaces `TreeNode/Label.vue`. Renders the right per-type leaf
 * component for the given tree node. Mirrors the Vue dispatcher's order
 * exactly — the 18 branches map 1:1 to Vue's `<template v-if>` chain.
 */
export function Label({ node, keyword }: Props) {
  switch (node.meta.type) {
    case "database":
      return <DatabaseNode node={node} keyword={keyword} />;
    case "schema":
      return <SchemaNode node={node} keyword={keyword} />;
    case "table":
      return <TableNode node={node} keyword={keyword} />;
    case "external-table":
      return <ExternalTableNode node={node} keyword={keyword} />;
    case "column":
      return <ColumnNode node={node} keyword={keyword} />;
    case "index":
      return <IndexNode node={node} keyword={keyword} />;
    case "foreign-key":
      return <ForeignKeyNode node={node} keyword={keyword} />;
    case "check":
      return <CheckNode node={node} keyword={keyword} />;
    case "partition-table":
      return <PartitionTableNode node={node} keyword={keyword} />;
    case "view":
      return <ViewNode node={node} keyword={keyword} />;
    case "dependency-column":
      return <DependencyColumnNode node={node} keyword={keyword} />;
    case "procedure":
      return <ProcedureNode node={node} keyword={keyword} />;
    case "package":
      return <PackageNode node={node} keyword={keyword} />;
    case "function":
      return <FunctionNode node={node} keyword={keyword} />;
    case "sequence":
      return <SequenceNode node={node} keyword={keyword} />;
    case "trigger":
      return <TriggerNode node={node} keyword={keyword} />;
    case "expandable-text":
      return <TextNode node={node} keyword={keyword} />;
    case "error":
      return <DummyNode node={node} keyword={keyword} />;
    default:
      return null;
  }
}
