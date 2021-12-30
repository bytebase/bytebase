<template>
  <slot />
</template>

<script lang="ts" setup>
import { watchEffect } from "vue";
import { useStore } from "vuex";
import {
  Instance,
  Database,
  Table,
  ConnectionAtom,
  ConnectionAtomType,
} from "../types";

const store = useStore();

const prepareSqlEdtiorContext = async function () {
  store.dispatch("sqlEditor/setConnectionContext", {
    isLoadingTree: true,
  });
  let connectionTree = [];

  const mapConnectionAtom =
    (type: ConnectionAtomType, parentId: number) =>
    (item: Instance | Database | Table) => {
      const connectionAtom: ConnectionAtom = {
        parentId,
        id: item.id,
        key: `${type}-${item.id}`,
        label: item.name,
        type,
      };

      return connectionAtom;
    };

  const instanceList = await store.dispatch("instance/fetchInstanceList");
  connectionTree = instanceList.map(mapConnectionAtom("instance", 0));

  for (const instance of instanceList) {
    const databaseList = await store.dispatch(
      "database/fetchDatabaseListByInstanceId",
      instance.id
    );

    const instanceItem = connectionTree.find(
      (item: ConnectionAtom) => item.id === instance.id
    );
    instanceItem.children = databaseList.map(
      mapConnectionAtom("database", instance.id)
    );

    for (const db of databaseList) {
      const tableList = await store.dispatch(
        "table/fetchTableListByDatabaseId",
        db.id
      );

      const databaseItem = instanceItem.children.find(
        (item: ConnectionAtom) => item.id === db.id
      );

      databaseItem.children = tableList.map(mapConnectionAtom("table", db.id));
    }
  }

  store.dispatch("sqlEditor/setConnectionTree", connectionTree);
  store.dispatch("sqlEditor/setConnectionContext", {
    isLoadingTree: false,
  });
};

watchEffect(prepareSqlEdtiorContext);
</script>
