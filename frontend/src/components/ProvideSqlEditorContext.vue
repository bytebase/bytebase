<template>
  <slot />
</template>

<script lang="ts" setup>
import { watchEffect } from "vue";
import { useStore } from "vuex";
import { Instance, Database, Table, ConnectionAtom } from "../types";
// import { idFromSlug } from "../utils";

const store = useStore();

const prepareSqlEdtiorContext = async function () {
  let connectionTree = [];

  const mapConnectionAtom = (item: Instance | Database | Table) => {
    const connectionAtom: ConnectionAtom = {
      id: item.id,
      key: item.id,
      label: item.name,
    };

    return connectionAtom;
  };

  const instanceList = await store.dispatch("instance/fetchInstanceList");
  connectionTree = instanceList.map(mapConnectionAtom);

  for (const instance of instanceList) {
    const databaseList = await store.dispatch(
      "database/fetchDatabaseListByInstanceId",
      instance.id
    );

    const instanceItem = connectionTree.find(
      (item: ConnectionAtom) => item.id === instance.id
    );
    instanceItem.children = databaseList.map(mapConnectionAtom);

    for (const db of databaseList) {
      const tableList = await store.dispatch(
        "table/fetchTableListByDatabaseId",
        db.id
      );

      const databaseItem = instanceItem.children.find(
        (item: ConnectionAtom) => item.id === db.id
      );

      databaseItem.children = tableList.map(mapConnectionAtom);
    }
  }

  store.dispatch("sqlEditor/setConnectionTree", connectionTree);
};

watchEffect(prepareSqlEdtiorContext);
</script>
