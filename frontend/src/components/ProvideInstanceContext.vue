<template>
  <slot />
</template>

<script lang="ts">
import { watchEffect } from "vue";
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

    const prepareInstanceContext = async function () {
      await Promise.all([
        store.dispatch(
          "database/fetchDatabaseListByInstanceId",
          idFromSlug(props.instanceSlug)
        ),
        store.dispatch(
          "instance/fetchInstanceUserListById",
          idFromSlug(props.instanceSlug)
        ),
      ]);
    };

    watchEffect(prepareInstanceContext);
  },
};
</script>
