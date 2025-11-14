import { flatten, orderBy, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref } from "vue";
import type {
  ComposedDatabase,
  SQLEditorTreeFactor as Factor,
  SQLEditorTreeNodeTarget as NodeTarget,
  SQLEditorTreeNodeType as NodeType,
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeState as TreeState,
} from "@/types";
import {
  extractSQLEditorLabelFactor as extractLabelFactor,
  formatEnvironmentName,
  LeafTreeNodeTypes,
  unknownEnvironment,
} from "@/types";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Environment } from "@/types/v1/environment";
import { getSemanticLabelValue, groupBy } from "@/utils";
import { useEnvironmentV1Store, useInstanceResourceByName } from "../v1";

export const useSQLEditorTreeStore = defineStore("sqlEditorTree", () => {
  const nodeListMapById = reactive(
    new Map<
      string /* node id by type and target */,
      string[] /* node key list */
    >()
  );
  // states
  const allNodeKeys = computed(() => {
    return uniq(flatten([...nodeListMapById.values()]));
  });

  const state = ref<TreeState>("UNSET");

  const collectNode = <T extends NodeType>(node: TreeNode<T>) => {
    const { type, target } = node.meta;
    const id = idForSQLEditorTreeNodeTarget(type, target);
    const nodeList = nodeListMapById.get(id) ?? [];
    nodeList.push(node.key);
    nodeListMapById.set(id, nodeList);
  };

  const nodeKeysByTarget = <T extends NodeType>(
    type: T,
    target: NodeTarget<T>
  ) => {
    const id = idForSQLEditorTreeNodeTarget(type, target);
    return nodeListMapById.get(id) ?? [];
  };

  return {
    state,
    collectNode,
    nodeKeysByTarget,
    allNodeKeys,
  };
});

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

export const idForSQLEditorTreeNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
) => {
  if (type === "instance" || type === "database") {
    return (target as Project | InstanceResource | ComposedDatabase).name;
  }
  if (type === "environment") {
    return formatEnvironmentName((target as Environment).id);
  }
  if (type === "label") {
    const kv = target as NodeTarget<"label">;
    return `labels/${kv.key}:${kv.value}`;
  }

  throw new Error(
    `should never reach this line, type=${type}, target=${target}, parent=${parent}`
  );
};

const buildSubTree = (
  databaseList: ComposedDatabase[],
  parent: TreeNode | undefined,
  factorList: Factor[],
  factorIndex: number
): TreeNode[] => {
  if (factorIndex === factorList.length) {
    // the last dimension is database nodes
    return databaseList.map((db) => {
      const node = mapTreeNodeByType("database", db, parent);
      return node;
    });
  }

  // group (project, instance, environment, label) nodes
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
    // Sort by environment order DESC. Put production envs to the first.
    return orderBy(
      nodes as TreeNode<"environment">[],
      [(node) => node.meta.target.order],
      ["desc"]
    );
  }
  if (factor.startsWith("label:")) {
    // Sort in lexicographical order, and put <empty value> to the last
    return orderBy(
      nodes as TreeNode<"label">[],
      [
        (node) => (node.meta.target.value ? -1 : 1), // Empty value to the last,
        (node) => node.meta.target.value, // lexicographical order then
      ],
      ["asc", "asc"]
    );
  }

  return nodes;
};

export const buildTreeImpl = (
  databaseList: ComposedDatabase[],
  factorList: Factor[]
): TreeNode[] => {
  return buildSubTree(
    databaseList,
    undefined /* parent */,
    factorList,
    0 /* factorIndex */
  );
};

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
  // factor is label
  const key = extractLabelFactor(factor);
  if (key) {
    return mapTreeNodeByType(
      "label",
      {
        key,
        value,
      },
      parent
    );
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

  useSQLEditorTreeStore().collectNode(node);

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
    return (target as ComposedDatabase).databaseName;
  }
  return (target as NodeTarget<"label">).value;
};

const isLeafNodeType = (type: NodeType) => {
  return LeafTreeNodeTypes.includes(type);
};

const getSemanticFactorValue = (db: ComposedDatabase, factor: Factor) => {
  switch (factor) {
    case "environment":
      return db.effectiveEnvironment || unknownEnvironment().name;
    case "instance":
      return db.instance;
  }

  const key = extractLabelFactor(factor);
  if (key) {
    return getSemanticLabelValue(db, key);
  }
  console.error("should never reach this line", db, factor);
  return "";
};
