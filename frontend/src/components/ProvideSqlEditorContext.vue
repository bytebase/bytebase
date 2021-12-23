<template>
  <slot />
</template>

<script lang="ts" setup>
import { watchEffect } from "vue";
import { useStore } from "vuex";
import { Instance, Database } from "../types";
// import { idFromSlug } from "../utils";

const store = useStore();

const prepareSqlEdtiorContext = async function () {
  let connectionTree = []

  const instances = await store.dispatch("instance/fetchInstanceList");
  connectionTree = instances.map((item: Instance) => ({
    id: item.id,
    key: item.id,
    label: item.name,
    children: []
  }))

  for (const instance of instances) {
    const databases = await store.dispatch(
      "database/fetchDatabaseListByInstanceId",
      instance.id
    );

    const InstanceItem = connectionTree.find(
      (item: any) => item.id === instance.id
    );
    InstanceItem.children = databases.map((item: Database) => ({
      id: item.id,
      key: item.id,
      label: item.name,
      children: []
    }));

    for (const db of databases) {
      const tables = await store.dispatch(
        "table/fetchTableListByDatabaseId",
        db.id
      );

      const DatabaseItem = InstanceItem.children.find(
        (item: any) => item.id === db.id
      );

      DatabaseItem.children = tables.map((item: any) => {
        // await store.dispatch("table/fetchTableByDatabaseIdAndTableName", {
        //   databaseId: db.id,
        //   tableName: item.name
        // });

        return {
          id: item.id,
          key: item.id,
          label: item.name
        }
      });
    }
  }

  store.dispatch("sqlEditor/setConnectionTree", connectionTree);
};

watchEffect(prepareSqlEdtiorContext);
</script>
