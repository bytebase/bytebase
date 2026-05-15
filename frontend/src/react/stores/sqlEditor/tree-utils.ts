import { orderBy } from "lodash-es";
import {
  useEnvironmentV1Store,
  useInstanceResourceByName,
} from "@/store/modules/v1";
import type {
  SQLEditorTreeFactor as Factor,
  SQLEditorTreeNodeTarget as NodeTarget,
  SQLEditorTreeNodeType as NodeType,
  SQLEditorTreeNode as TreeNode,
} from "@/types";
import {
  extractSQLEditorLabelFactor as extractLabelFactor,
  LeafTreeNodeTypes,
  unknownEnvironment,
} from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import type { Environment } from "@/types/v1/environment";
import {
  extractDatabaseResourceName,
  getSemanticLabelValue,
  groupBy,
} from "@/utils";
import { useSQLEditorStore } from "./index";
import { idForSQLEditorTreeNodeTarget } from "./tree";

const keyForSQLEditorTreeNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>,
  parent?: TreeNode
): string => {
  const id = idForSQLEditorTreeNodeTarget(type, target);
  const parts = [id];
  if (parent) {
    parts.unshift(parent.key);
  }
  return parts.join("/");
};

const buildSubTree = (
  databaseList: Database[],
  parent: TreeNode | undefined,
  factorList: Factor[],
  factorIndex: number
): TreeNode[] => {
  if (factorIndex === factorList.length) {
    return databaseList.map((db) => mapTreeNodeByType("database", db, parent));
  }

  const nodes: TreeNode[] = [];
  const factor = factorList[factorIndex];

  const groups = groupBy(databaseList, (db) =>
    getSemanticFactorValue(db, factor)
  );
  for (const [value, childrenDatabaseList] of groups) {
    const groupNode = mapGroupNode(factor, value, parent);
    groupNode.children = buildSubTree(
      childrenDatabaseList,
      groupNode,
      factorList,
      factorIndex + 1
    );
    nodes.push(groupNode);
  }
  return sortNodesIfNeeded(nodes, factor);
};

const sortNodesIfNeeded = (nodes: TreeNode[], factor: Factor) => {
  if (factor === "environment") {
    return orderBy(
      nodes as TreeNode<"environment">[],
      [(node) => node.meta.target.order],
      ["desc"]
    );
  }
  if (factor.startsWith("label:")) {
    return orderBy(
      nodes as TreeNode<"label">[],
      [
        (node) => (node.meta.target.value ? -1 : 1),
        (node) => node.meta.target.value,
      ],
      ["asc", "asc"]
    );
  }
  return nodes;
};

export const buildTreeImpl = (
  databaseList: Database[],
  factorList: Factor[]
): TreeNode[] => buildSubTree(databaseList, undefined, factorList, 0);

const mapGroupNode = (
  factor: Factor,
  value: string,
  parent: TreeNode | undefined
) => {
  if (factor === "environment") {
    const environment = useEnvironmentV1Store().getEnvironmentByName(
      value || ""
    );
    return mapTreeNodeByType("environment", environment, parent);
  }
  if (factor === "instance") {
    const { instance } = useInstanceResourceByName(value);
    return mapTreeNodeByType("instance", instance.value, parent);
  }
  const key = extractLabelFactor(factor);
  if (key) {
    return mapTreeNodeByType("label", { key, value }, parent);
  }
  throw new Error(
    `mapGroupNode: should never reach this line. factor=${factor}, factorValue=${value}`
  );
};

export const mapTreeNodeByType = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>,
  parent: TreeNode | undefined,
  overrides: Partial<TreeNode<T>> | undefined = undefined
): TreeNode<T> => {
  const key = keyForSQLEditorTreeNodeTarget(type, target, parent);
  const node: TreeNode<T> = {
    key,
    meta: { type, target },
    label: readableTargetByType(type, target),
    isLeaf: isLeafNodeType(type),
    ...overrides,
  };

  useSQLEditorStore.getState().collectTreeNode(node);
  return node;
};

const readableTargetByType = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
): string => {
  if (type === "environment") {
    return (target as Environment).title;
  }
  if (type === "instance") {
    return (target as InstanceResource).title;
  }
  if (type === "database") {
    return extractDatabaseResourceName((target as Database).name).databaseName;
  }
  return (target as NodeTarget<"label">).value;
};

const isLeafNodeType = (type: NodeType) => LeafTreeNodeTypes.includes(type);

const getSemanticFactorValue = (db: Database, factor: Factor) => {
  switch (factor) {
    case "environment":
      return db.effectiveEnvironment || unknownEnvironment().name;
    case "instance":
      return extractDatabaseResourceName(db.name).instance;
  }
  const key = extractLabelFactor(factor);
  if (key) {
    return getSemanticLabelValue(db, key);
  }
  console.error("should never reach this line", db, factor);
  return "";
};
