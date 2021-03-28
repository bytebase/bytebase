<template>
  <slot />
</template>

<script lang="ts">
import { watchEffect } from "vue";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";

export default {
  name: "ProvideDatabaseContext",
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
  },
  async setup(props) {
    const store = useStore();

    const prepareDatabaseContext = async function () {
      await Promise.all([
        store.dispatch(
          "dataSource/fetchDataSourceListByDatabaseId",
          idFromSlug(props.databaseSlug)
        ),
      ]);
    };

    watchEffect(prepareDatabaseContext);
  },
};
</script>
