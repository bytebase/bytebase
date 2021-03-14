<template>
  <slot />
</template>

<script lang="ts">
import { useStore } from "vuex";
import { idFromSlug } from "../utils";

export default {
  name: "ProvideInstanceContext",
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
  },
  async setup(props) {
    const store = useStore();

    if (props.instanceSlug.toLowerCase() != "new") {
      await store.dispatch(
        "database/fetchDatabaseListByInstanceId",
        idFromSlug(props.instanceSlug)
      );
      // This depends on database/fetchDatabaseListByInstanceId to convert its database id to database object.
      await store.dispatch(
        "dataSource/fetchDataSourceListByInstanceId",
        idFromSlug(props.instanceSlug)
      );
    }
  },
};
</script>
