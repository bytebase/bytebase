import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import { reactive, ref, watch } from "vue";
import {
  Connection,
  ConnectionAtom,
  ConnectionTreeMode,
  ConnectionAtomType,
  ComposedInstance,
  ComposedDatabase,
} from "@/types";
import { ConnectionTreeState, UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { Policy } from "@/types/proto/v1/org_policy_service";
import { Project } from "@/types/proto/v1/project_service";
import { emptyConnection } from "@/utils";
import {
  useDatabaseV1Store,
  useInstanceV1Store,
  useDBSchemaV1Store,
} from "./v1";

export const ROOT_NODE_ID = "ROOT";
export const CONNECTION_TREE_DELIMITER = "->";

// Normalize value, fallback to ConnectionTreeMode.PROJECT
const normalizeConnectionTreeMode = (raw: string) => {
  if (raw === ConnectionTreeMode.INSTANCE) {
    return raw;
  }
  return ConnectionTreeMode.PROJECT;
};

export const useConnectionTreeStore = defineStore("connectionTree", () => {
  const treeModeInLocalStorage = useLocalStorage<ConnectionTreeMode>(
    "bb.sql-editor.default-connection-tree-mode",
    ConnectionTreeMode.PROJECT,
    {
      serializer: {
        read: normalizeConnectionTreeMode,
        write: normalizeConnectionTreeMode,
      },
    }
  );

  // states
  const accessControlPolicyList = ref<Policy[]>([]);
  const tree = reactive({
    databaseList: [] as ComposedDatabase[],
    data: [] as ConnectionAtom[],
    mode: treeModeInLocalStorage.value,
    state: ConnectionTreeState.UNSET,
  });
  const expandedTreeNodeKeys = ref<string[]>([]);

  // actions
  const fetchConnectionByInstanceIdAndDatabaseId = async (
    instanceId: string,
    databaseId: string
  ): Promise<Connection> => {
    try {
      const [db, _] = await Promise.all([
        useDatabaseV1Store().getOrFetchDatabaseByUID(databaseId),
        useInstanceV1Store().getOrFetchInstanceByUID(instanceId),
      ]);
      await useDBSchemaV1Store().getOrFetchTableList(db.name);

      return {
        instanceId,
        databaseId,
      };
    } catch {
      // Fallback to disconnected if error occurs such as 404.
      return { instanceId: String(UNKNOWN_ID), databaseId: String(UNKNOWN_ID) };
    }
  };
  const fetchConnectionByInstanceId = async (
    instanceId: string
  ): Promise<Connection> => {
    try {
      await useInstanceV1Store().getOrFetchInstanceByUID(instanceId);

      return {
        instanceId,
        databaseId: String(UNKNOWN_ID),
      };
    } catch {
      // Fallback to disconnected if error occurs such as 404.
      return { instanceId: String(UNKNOWN_ID), databaseId: String(UNKNOWN_ID) };
    }
  };
  // utilities
  const mapAtom = (
    item:
      | Project
      | ComposedInstance
      | ComposedDatabase
      | SchemaMetadata
      | TableMetadata,
    type: ConnectionAtomType,
    parent: ConnectionAtom | undefined,
    children?: ConnectionAtom[]
  ) => {
    const id = idForConnectionAtomItem(type, item, parent);
    const key = `${type}-${id}`;
    const connectionAtom: ConnectionAtom = {
      parentId: parent?.id ?? ROOT_NODE_ID,
      id,
      key,
      label:
        type === "project"
          ? (item as Project).title
          : type === "instance"
          ? (item as ComposedInstance).title
          : type === "database"
          ? (item as ComposedDatabase).databaseName
          : type === "schema"
          ? (item as SchemaMetadata).name
          : type === "table"
          ? (item as TableMetadata).name
          : "",
      type,
      isLeaf: type === "table",
      children,
    };
    return connectionAtom;
  };

  watch(
    () => tree.mode,
    (mode) => {
      treeModeInLocalStorage.value = mode;
    },
    { immediate: true }
  );

  return {
    accessControlPolicyList,
    tree,
    expandedTreeNodeKeys,
    fetchConnectionByInstanceIdAndDatabaseId,
    fetchConnectionByInstanceId,
    mapAtom,
  };
});

export const searchConnectionByName = (
  instanceId: string,
  databaseId: string,
  instanceName: string,
  databaseName: string
): Connection => {
  const connection = emptyConnection();
  const store = useConnectionTreeStore();

  if (instanceId !== String(UNKNOWN_ID)) {
    // If we found instanceId and/or databaseId, use the IDs first.
    connection.instanceId = instanceId;
    if (databaseId !== String(UNKNOWN_ID)) {
      connection.databaseId = databaseId;
    }

    return connection;
  }

  // Search the instance and database by name otherwise.
  // Remain this part for legacy sheet support.
  const rootNodes = store.tree.data;
  for (let i = 0; i < rootNodes.length; i++) {
    const maybeInstanceNode = rootNodes[i];
    if (maybeInstanceNode.type !== "instance") {
      // Skip if we met dirty data.
      continue;
    }
    if (maybeInstanceNode.label === instanceName) {
      connection.instanceId = maybeInstanceNode.id;
      if (databaseName) {
        const { children = [] } = maybeInstanceNode;
        for (let j = 0; j < children.length; j++) {
          const maybeDatabaseNode = children[j];
          if (maybeDatabaseNode.type !== "database") {
            // Skip if we met dirty data.
            continue;
          }
          if (maybeDatabaseNode.label === databaseName) {
            connection.databaseId = maybeDatabaseNode.id;
            // Don't go further since we've found the databaseId
            break;
          }
        }
      }
      // Don't go further since we've found the instanceId
      break;
    }
  }

  return connection;
};

export const isConnectableAtom = (atom: ConnectionAtom): boolean => {
  if (atom.disabled) {
    return false;
  }
  if (atom.type === "database") {
    return true;
  }
  if (atom.type === "instance") {
    const instance = useInstanceV1Store().getInstanceByUID(atom.id);
    const { engine } = instance;
    return engine === Engine.MYSQL || engine === Engine.TIDB;
  }
  return false;
};

export const idForConnectionAtomItem = (
  type: ConnectionAtomType,
  item:
    | Project
    | ComposedInstance
    | ComposedDatabase
    | SchemaMetadata
    | TableMetadata,
  parent: ConnectionAtom | undefined
) => {
  if (type === "project" || type === "instance" || type === "database") {
    const target = item as Project | ComposedInstance | ComposedDatabase;
    return target.uid;
  }
  if (type === "schema") {
    const target = item as SchemaMetadata;
    return [parent?.id ?? ROOT_NODE_ID, target.name].join(
      CONNECTION_TREE_DELIMITER
    );
  }
  if (type === "table") {
    const target = item as TableMetadata;
    return [parent?.id ?? ROOT_NODE_ID, target.name].join(
      CONNECTION_TREE_DELIMITER
    );
  }

  console.error("should never reach this line", type, item);
  return "";
};
