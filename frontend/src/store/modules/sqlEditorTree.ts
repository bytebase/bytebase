import { useLocalStorage } from "@vueuse/core";
import { orderBy } from "lodash-es";
import { defineStore } from "pinia";
import { v4 as uuidv4 } from "uuid";
import { ref, watch } from "vue";
import { t } from "@/plugins/i18n";
import {
  ComposedDatabase,
  ComposedInstance,
  ComposedProject,
  SQLEditorTreeFactor as Factor,
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeNodeTarget as NodeTarget,
  SQLEditorTreeNodeType as NodeType,
  SQLEditorTreeState as TreeState,
  isValidSQLEditorTreeFactor as isValidFactor,
  extractSQLEditorLabelFactor as extractLabelFactor,
  unknownEnvironment,
} from "@/types";
import {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { Environment } from "@/types/proto/v1/environment_service";
import { Policy } from "@/types/proto/v1/org_policy_service";
import { getSemanticLabelValue, groupBy } from "@/utils";
import {
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
} from "./v1";

export const ROOT_NODE_ID = "ROOT";
export const SQL_EDITOR_TREE_NODE_ID_DELIMITER = "->";

const defaultFactorList = (): Factor[] => ["project", "environment"];

const factorListInLocalStorage = useLocalStorage<Factor[]>(
  "bb.sql-editor.tree-factor-list",
  defaultFactorList(),
  {
    serializer: {
      read: (raw: string): Factor[] => {
        try {
          const array = JSON.parse(raw) as string[];
          if (!Array.isArray(array)) {
            throw new Error();
          }
          const factorList: Factor[] = [];
          array.forEach((str) => {
            if (isValidFactor(str)) {
              factorList.push(str);
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

export const useSQLEditorTreeStore = defineStore("SQL-Editor-Tree", () => {
  // states
  const accessControlPolicyList = ref<Policy[]>([]);
  const databaseList = ref<ComposedDatabase[]>([]);
  // const factorList = ref(factorListInLocalStorage.value);
  const factorList = ref<Factor[]>(["instance", "environment", "label:tenant"]);
  const state = ref<TreeState>("UNSET");
  // const flattenExpandedTreeNodeKeys = ref<string[]>([]); // mixed factor type

  const tree = ref<TreeNode[]>([]);
  const buildTree = () => {
    tree.value = buildTreeImpl(databaseList.value, factorList.value);
  };
  // const expandedTreeNodeKeysForCurrentFactors = computed(() => {
  //   const { tree, expandedTreeNodeKeys } = connectionTreeStore;
  //   const prefixList = factorList.value.map((factor) => {
  //     return `${factor}-`;
  //   });

  //   switch (tree.mode) {
  //     case ConnectionTreeMode.INSTANCE:
  //       return expandedTreeNodeKeys.filter(
  //         (key) => !key.startsWith("project-")
  //       );
  //     case ConnectionTreeMode.PROJECT:
  //       return expandedTreeNodeKeys.filter(
  //         (key) => !key.startsWith("instance-")
  //       );
  //   }
  //   // Fallback to make TypeScript compiler happy
  //   return [];
  // });

  // // actions
  // const fetchConnectionByInstanceIdAndDatabaseId = async (
  //   instanceId: string,
  //   databaseId: string
  // ): Promise<Connection> => {
  //   try {
  //     const [db, _] = await Promise.all([
  //       useDatabaseV1Store().getOrFetchDatabaseByUID(databaseId),
  //       useInstanceV1Store().getOrFetchInstanceByUID(instanceId),
  //     ]);
  //     await useDBSchemaV1Store().getOrFetchTableList(db.name);

  //     return {
  //       instanceId,
  //       databaseId,
  //     };
  //   } catch {
  //     // Fallback to disconnected if error occurs such as 404.
  //     return {
  //       instanceId: String(UNKNOWN_ID),
  //       databaseId: String(UNKNOWN_ID),
  //     };
  //   }
  // };
  // const fetchConnectionByInstanceId = async (
  //   instanceId: string
  // ): Promise<Connection> => {
  //   try {
  //     await useInstanceV1Store().getOrFetchInstanceByUID(instanceId);

  //     return {
  //       instanceId,
  //       databaseId: String(UNKNOWN_ID),
  //     };
  //   } catch {
  //     // Fallback to disconnected if error occurs such as 404.
  //     return {
  //       instanceId: String(UNKNOWN_ID),
  //       databaseId: String(UNKNOWN_ID),
  //     };
  //   }
  // };
  // // utilities
  // const mapAtom = (
  //   item:
  //     | Project
  //     | ComposedInstance
  //     | ComposedDatabase
  //     | SchemaMetadata
  //     | TableMetadata,
  //   type: ConnectionAtomType,
  //   parent: ConnectionAtom | undefined,
  //   children?: ConnectionAtom[]
  // ) => {
  //   const id = idForConnectionAtomItem(type, item, parent);
  //   const key = `${type}-${id}`;
  //   const connectionAtom: ConnectionAtom = {
  //     parentId: parent?.id ?? ROOT_NODE_ID,
  //     id,
  //     key,
  //     label:
  //       type === "project"
  //         ? (item as Project).title
  //         : type === "instance"
  //         ? (item as ComposedInstance).title
  //         : type === "database"
  //         ? (item as ComposedDatabase).databaseName
  //         : type === "schema"
  //         ? (item as SchemaMetadata).name
  //         : type === "table"
  //         ? (item as TableMetadata).name
  //         : "",
  //     type,
  //     isLeaf: type === "table",
  //     children,
  //   };
  //   return connectionAtom;
  // };

  watch(
    factorList,
    (factorList) => {
      factorListInLocalStorage.value = factorList;
    },
    { immediate: true, deep: true }
  );

  return {
    accessControlPolicyList,
    databaseList,
    factorList,
    state,
    tree,
    buildTree,
  };
});

// export const searchConnectionByName = (
//   instanceId: string,
//   databaseId: string,
//   instanceName: string,
//   databaseName: string
// ): Connection => {
//   const connection = emptyConnection();
//   const store = useConnectionTreeStore();

//   if (instanceId !== String(UNKNOWN_ID)) {
//     // If we found instanceId and/or databaseId, use the IDs first.
//     connection.instanceId = instanceId;
//     if (databaseId !== String(UNKNOWN_ID)) {
//       connection.databaseId = databaseId;
//     }

//     return connection;
//   }

//   // Search the instance and database by name otherwise.
//   // Remain this part for legacy sheet support.
//   const rootNodes = store.tree.data;
//   for (let i = 0; i < rootNodes.length; i++) {
//     const maybeInstanceNode = rootNodes[i];
//     if (maybeInstanceNode.type !== "instance") {
//       // Skip if we met dirty data.
//       continue;
//     }
//     if (maybeInstanceNode.label === instanceName) {
//       connection.instanceId = maybeInstanceNode.id;
//       if (databaseName) {
//         const { children = [] } = maybeInstanceNode;
//         for (let j = 0; j < children.length; j++) {
//           const maybeDatabaseNode = children[j];
//           if (maybeDatabaseNode.type !== "database") {
//             // Skip if we met dirty data.
//             continue;
//           }
//           if (maybeDatabaseNode.label === databaseName) {
//             connection.databaseId = maybeDatabaseNode.id;
//             // Don't go further since we've found the databaseId
//             break;
//           }
//         }
//       }
//       // Don't go further since we've found the instanceId
//       break;
//     }
//   }

//   return connection;
// };

// export const isConnectableAtom = (atom: ConnectionAtom): boolean => {
//   if (atom.disabled) {
//     return false;
//   }
//   if (atom.type === "database") {
//     return true;
//   }
//   if (atom.type === "instance") {
//     const instance = useInstanceV1Store().getInstanceByUID(atom.id);
//     const { engine } = instance;
//     return engine === Engine.MYSQL || engine === Engine.TIDB;
//   }
//   return false;
// };

// export const idForConnectionAtomItem = (
//   type: ConnectionAtomType,
//   item:
//     | Project
//     | ComposedInstance
//     | ComposedDatabase
//     | SchemaMetadata
//     | TableMetadata,
//   parent: ConnectionAtom | undefined
// ) => {
//   if (type === "project" || type === "instance" || type === "database") {
//     const target = item as Project | ComposedInstance | ComposedDatabase;
//     return target.uid;
//   }
//   if (type === "schema") {
//     const target = item as SchemaMetadata;
//     return [parent?.id ?? ROOT_NODE_ID, target.name].join(
//       SQL_EDITOR_TREE_DELIMITER
//     );
//   }
//   if (type === "table") {
//     const target = item as TableMetadata;
//     return [parent?.id ?? ROOT_NODE_ID, target.name].join(
//       SQL_EDITOR_TREE_DELIMITER
//     );
//   }

//   console.error("should never reach this line", type, item);
//   return "";
// };

export const keyForSQLEditorTreeNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
): string => {
  let prefix: string = type;
  if (type === "label") {
    prefix = `label:${(target as NodeTarget<"label">).key}`;
  }
  return `${prefix}-${uuidv4()}`;
};

export const idForSQLEditorTreeNodeTarget = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>,
  parent: TreeNode | undefined = undefined
) => {
  if (
    type === "project" ||
    type === "instance" ||
    type === "environment" ||
    type === "database"
  ) {
    return (
      target as
        | ComposedProject
        | ComposedInstance
        | Environment
        | ComposedDatabase
    ).name;
  }
  if (type === "schema" || type === "table") {
    const name = (target as SchemaMetadata | TableMetadata).name;
    return [parent?.id ?? ROOT_NODE_ID, name].join(
      SQL_EDITOR_TREE_NODE_ID_DELIMITER
    );
  }
  if (type === "label") {
    const kv = target as NodeTarget<"label">;
    return [parent?.id ?? ROOT_NODE_ID, `${kv.key}:${kv.value}`].join(
      SQL_EDITOR_TREE_NODE_ID_DELIMITER
    );
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

  // group (project, instance, project, label) nodes
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
      ["asc"]
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
  if (factor === "project") {
    const project = useProjectV1Store().getProjectByName(value);
    return mapTreeNodeByType("project", project, parent);
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

// const mapTreeNode = (db: ComposedDatabase, factor: Factor) => {
//   if (factor === "environment") {
//     const environment = db.effectiveEnvironmentEntity;
//     return mapTreeNodeByType("environment", environment);
//   }
//   if (factor === "instance") {
//     const instance = db.instanceEntity;
//     return mapTreeNodeByType("instance", instance);
//   }
//   if (factor === "project") {
//     const project = db.projectEntity;
//     return mapTreeNodeByType("project", project);
//   }
//   const key = extractSQLEditorLabelFactor(factor);
//   if (key) {
//     const value = getSemanticLabelValue(db, key);
//     return mapTreeNodeByType("label", { key, value });
//   }
// };

export const mapTreeNodeByType = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>,
  parent: TreeNode | undefined
): TreeNode<T> => {
  const id = idForSQLEditorTreeNodeTarget(type, target, parent);
  const key = keyForSQLEditorTreeNodeTarget(type, target);
  const node: TreeNode<T> = {
    id,
    key,
    meta: { type, target },
    parent: parent?.id ?? ROOT_NODE_ID,
    label: readableTargetByType(type, target),
    isLeaf: type === "table",
  };

  return node;
};

const readableTargetByType = <T extends NodeType>(
  type: T,
  target: NodeTarget<T>
): string => {
  if (type === "environment") {
    return (target as Environment).title;
  }
  if (type === "project") {
    return (target as ComposedProject).title;
  }
  if (type === "instance") {
    return (target as ComposedInstance).title;
  }
  if (type === "database") {
    return (target as ComposedDatabase).databaseName;
  }
  if (type === "schema") {
    return (target as SchemaMetadata).name;
  }
  if (type === "table") {
    return (target as TableMetadata).name;
  }
  // return (target as NodeTarget<"label">).value;
  // for debugging below
  const value = (target as NodeTarget<"label">).value;
  if (!value) {
    return t("label.empty-label-value");
  }
  return value;
};

const getSemanticFactorValue = (db: ComposedDatabase, factor: Factor) => {
  switch (factor) {
    case "environment":
      return db.effectiveEnvironment;
    case "instance":
      return db.instance;
    case "project":
      return db.project;
  }

  const key = extractLabelFactor(factor);
  if (key) {
    return getSemanticLabelValue(db, key);
  }
  console.error("should never reach this line", db, factor);
  return "";
};
