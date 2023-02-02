import { defineStore } from "pinia";
import { reactive, ref, watch } from "vue";
import {
  DatabaseId,
  InstanceId,
  Connection,
  Policy,
  ConnectionAtom,
  ConnectionTreeMode,
  ConnectionAtomType,
  Project,
  Instance,
  Database,
} from "@/types";
import { ConnectionTreeState, UNKNOWN_ID } from "@/types";
import { useDatabaseStore } from "./database";
import { useInstanceStore } from "./instance";
import { emptyConnection } from "@/utils";
import { useDBSchemaStore } from "./dbSchema";
import { useLocalStorage } from "@vueuse/core";

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
    data: [] as ConnectionAtom[],
    mode: treeModeInLocalStorage.value,
    state: ConnectionTreeState.UNSET,
  });
  const expandedTreeNodeKeys = ref<string[]>([]);
  const selectedTableAtom = ref<ConnectionAtom>();

  // actions
  const fetchConnectionByInstanceIdAndDatabaseId = async (
    instanceId: InstanceId,
    databaseId: DatabaseId
  ): Promise<Connection> => {
    try {
      await Promise.all([
        useDatabaseStore().getOrFetchDatabaseById(databaseId),
        useInstanceStore().getOrFetchInstanceById(instanceId),
        useDBSchemaStore().getOrFetchTableListByDatabaseId(databaseId),
      ]);

      return {
        instanceId,
        databaseId,
      };
    } catch {
      // Fallback to disconnected if error occurs such as 404.
      return { instanceId: UNKNOWN_ID, databaseId: UNKNOWN_ID };
    }
  };
  const fetchConnectionByInstanceId = async (
    instanceId: InstanceId
  ): Promise<Connection> => {
    try {
      const [databaseList] = await Promise.all([
        useDatabaseStore().getDatabaseListByInstanceId(instanceId),
        useInstanceStore().getOrFetchInstanceById(instanceId),
      ]);
      const dbSchemaStore = useDBSchemaStore();
      await Promise.all(
        databaseList.map((db) =>
          dbSchemaStore.getOrFetchTableListByDatabaseId(db.id)
        )
      );

      return {
        instanceId,
        databaseId: UNKNOWN_ID,
      };
    } catch {
      // Fallback to disconnected if error occurs such as 404.
      return { instanceId: UNKNOWN_ID, databaseId: UNKNOWN_ID };
    }
  };
  // utilities
  const mapAtom = (
    item: Project | Instance | Database,
    type: ConnectionAtomType,
    parentId: number
  ) => {
    const id = item.id;
    const key = `${type}-${id}`;
    const connectionAtom: ConnectionAtom = {
      parentId,
      id,
      key,
      label: item.name,
      type,
      isLeaf: type === "database",
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
    selectedTableAtom,
    fetchConnectionByInstanceIdAndDatabaseId,
    fetchConnectionByInstanceId,
    mapAtom,
  };
});

export const searchConnectionByName = (
  instanceId: InstanceId,
  databaseId: DatabaseId,
  instanceName: string,
  databaseName: string
): Connection => {
  const connection = emptyConnection();
  const store = useConnectionTreeStore();

  if (instanceId !== UNKNOWN_ID) {
    // If we found instanceId and/or databaseId, use the IDs first.
    connection.instanceId = instanceId;
    if (databaseId !== UNKNOWN_ID) {
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
