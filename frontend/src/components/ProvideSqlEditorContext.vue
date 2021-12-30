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
    (type: ConnectionAtomType) => (item: Instance | Database | Table) => {
      const connectionAtom: ConnectionAtom = {
        id: item.id,
        key: item.id,
        label: item.name,
        type,
      };

      return connectionAtom;
    };

  const instanceList = await store.dispatch("instance/fetchInstanceList");
  connectionTree = instanceList.map(mapConnectionAtom("instance"));

  for (const instance of instanceList) {
    const databaseList = await store.dispatch(
      "database/fetchDatabaseListByInstanceId",
      instance.id
    );

    const instanceItem = connectionTree.find(
      (item: ConnectionAtom) => item.id === instance.id
    );
    instanceItem.children = databaseList.map(mapConnectionAtom("database"));

    for (const db of databaseList) {
      const tableList = await store.dispatch(
        "table/fetchTableListByDatabaseId",
        db.id
      );

      const databaseItem = instanceItem.children.find(
        (item: ConnectionAtom) => item.id === db.id
      );

      databaseItem.children = tableList.map(mapConnectionAtom("table"));
    }
  }

  store.dispatch("sqlEditor/setConnectionTree", connectionTree);
  store.dispatch("sqlEditor/setConnectionContext", {
    isLoadingTree: false,
  });
};

watchEffect(prepareSqlEdtiorContext);
</script>
