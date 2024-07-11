import { useLocalStorage } from "@vueuse/core";
import { cloneDeep, orderBy, uniq } from "lodash-es";
import { defineStore, storeToRefs } from "pinia";
import { computed, reactive, ref, watch } from "vue";
import { useCurrentUserV1 } from "@/store";
import type {
  ComposedDatabase,
  ComposedInstance,
  ComposedProject,
  SQLEditorTreeFactor as Factor,
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeNodeTarget as NodeTarget,
  SQLEditorTreeNodeType as NodeType,
  SQLEditorTreeState as TreeState,
  StatefulSQLEditorTreeFactor as StatefulFactor,
} from "@/types";
import {
  isValidSQLEditorTreeFactor as isValidFactor,
  extractSQLEditorLabelFactor as extractLabelFactor,
  unknownEnvironment,
  LeafTreeNodeTypes,
} from "@/types";
import type { Environment } from "@/types/proto/v1/environment_service";
import { getSemanticLabelValue, groupBy, isDatabaseV1Queryable } from "@/utils";
import { useFilterStore } from "../filter";
import { useEnvironmentV1Store, useInstanceV1Store } from "../v1";
import { useSQLEditorStore } from "./editor";

export const ROOT_NODE_ID = "ROOT";

const defaultEnvironmentFactor: StatefulFactor = {
  factor: "environment",
  disabled: false,
};

export const useSQLEditorTreeStore = defineStore("sqlEditorTree", () => {
  const { filter } = useFilterStore();
  const me = useCurrentUserV1();

  const defaultFactorList = (): StatefulFactor[] => {
    return [defaultEnvironmentFactor];
  };

  const factorListInLocalStorage = useLocalStorage<StatefulFactor[]>(
    "bb.sql-editor.tree-factor-list",
    defaultFactorList(),
    {
      serializer: {
        read: (raw: string): StatefulFactor[] => {
          try {
            const array = JSON.parse(raw) as StatefulFactor[];
            if (!Array.isArray(array)) {
              throw new Error();
            }
            const factorList: StatefulFactor[] = [];
            array.forEach((sf) => {
              if (isValidFactor(sf.factor)) {
                factorList.push({
                  factor: sf.factor,
                  disabled: !!sf.disabled,
                });
              }
            });
            if (factorList.length === 0) {
              throw new Error();
            }
            return factorList;
          } catch {
            return defaultFactorList();
          }
        },
        write: (factorList) => JSON.stringify(factorList),
      },
    }
  );
  const nodeListMapById = reactive(new Map<string, TreeNode[]>());
  // states
  // re-expose `databaseList`, `project`, `currentProject` from sqlEditor store for shortcuts
  const { databaseList, project, currentProject } =
    storeToRefs(useSQLEditorStore());
  const factorList = ref<StatefulFactor[]>(
    cloneDeep(factorListInLocalStorage.value)
  );

  const sortedDatabaseList = computed(() => {
    if (!showMissingQueryDatabases.value) {
      return databaseList.value.filter((db) =>
        isDatabaseV1Queryable(db, me.value)
      );
    }
    return orderBy(
      databaseList.value,
      [(db) => (isDatabaseV1Queryable(db, me.value) ? 1 : 0)],
      ["desc"]
    );
  });

  const filteredDatabaseList = computed(() => {
    if (filter.database) {
      return sortedDatabaseList.value.filter((database) => {
        return database.name === filter.database;
      });
    }
    if (filter.project || currentProject.value) {
      const projectName = filter.project ?? currentProject.value?.name;
      return sortedDatabaseList.value.filter((database) => {
        return database.project === projectName;
      });
    }

    return sortedDatabaseList.value;
  });

  const filteredFactorList = computed(() => {
    return factorList.value.filter((sf) => !sf.disabled).map((sf) => sf.factor);
  });
  const availableFactorList = computed(() => {
    const PRESET_FACTORS: Factor[] = ["instance", "environment"];
    const labelFactors = orderBy(
      uniq(
        databaseList.value.flatMap((db) => Object.keys(db.labels))
      ).map<Factor>((key) => `label:${key}` as Factor),
      [(key) => key],
      ["asc"] // lexicographical order
    );

    return {
      preset: PRESET_FACTORS,
      label: labelFactors,
      all: [...PRESET_FACTORS, ...labelFactors],
    };
  });

  const state = ref<TreeState>("UNSET");
  const tree = ref<TreeNode[]>([]);
  const expandedKeys = ref<string[]>([]);
  const showMissingQueryDatabases = ref<boolean>(false);

  const collectNode = <T extends NodeType>(node: TreeNode<T>) => {
    const { type, target } = node.meta;
    const id = idForSQLEditorTreeNodeTarget(type, target);
    const nodeList = nodeListMapById.get(id) ?? [];
    nodeList.push(node);
    nodeListMapById.set(id, nodeList);
  };
  const nodesByTarget = <T extends NodeType>(
    type: T,
    target: NodeTarget<T>
  ) => {
    const id = idForSQLEditorTreeNodeTarget(type, target);
    return (nodeListMapById.get(id) ?? []) as TreeNode<T>[];
  };

  const buildTree = () => {
    nodeListMapById.clear();
    expandedKeys.value = [];
    tree.value = buildTreeImpl(
      filteredDatabaseList.value,
      filteredFactorList.value
    );
  };

  const cleanup = () => {
    tree.value = [];
    factorList.value = defaultFactorList();
    nodeListMapById.clear();
    expandedKeys.value = [];
    showMissingQueryDatabases.value = false;
    state.value = "UNSET";
  };

  watch(
    () => showMissingQueryDatabases.value,
    () => {
      tree.value = buildTreeImpl(
        filteredDatabaseList.value,
        filteredFactorList.value
      );
    }
  );

  watch(
    factorList,
    (factorList) => {
      factorListInLocalStorage.value = factorList;
    },
    { immediate: true, deep: true }
  );
  watch(
    project,
    (project) => {
      if (project) {
        const position = factorList.value.findIndex(
          (sf) => sf.factor === "project"
        );
        if (position > 0) {
          factorList.value.splice(position, 1);
          if (factorList.value.length === 0) {
            factorList.value = defaultFactorList();
          }
        }
      }
    },
    {
      immediate: true,
    }
  );

  return {
    availableFactorList,
    factorList,
    filteredFactorList,
    state,
    tree,
    collectNode,
    nodesByTarget,
    buildTree,
    expandedKeys,
    showMissingQueryDatabases,
    cleanup,
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
  return JSON.stringify(parts);
};

export const idForSQLEditorTreeNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
) => {
  if (type === "instance" || type === "environment" || type === "database") {
    return (
      target as
        | ComposedProject
        | ComposedInstance
        | Environment
        | ComposedDatabase
    ).name;
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

const buildTreeImpl = (
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
    const environment =
      useEnvironmentV1Store().getEnvironmentByName(value) ??
      unknownEnvironment();
    return mapTreeNodeByType("environment", environment, parent);
  }
  if (factor === "instance") {
    const instance = useInstanceV1Store().getInstanceByName(value);
    return mapTreeNodeByType("instance", instance, parent);
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
    parent,
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
    return (target as ComposedInstance).title;
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
      return db.effectiveEnvironment;
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
