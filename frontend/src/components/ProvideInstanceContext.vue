<template>
  <slot />
</template>

<script lang="ts">
import { useDatabaseStore, useInstanceStore } from "@/store";
import { defineComponent, watchEffect } from "vue";
import { idFromSlug } from "../utils";

export default defineComponent({
  name: "ProvideInstanceContext",
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
  },
  async setup(props) {
    const prepareInstanceContext = async function () {
      await Promise.all([
        useDatabaseStore().fetchDatabaseListByInstanceId(
          idFromSlug(props.instanceSlug)
        ),
        useInstanceStore().fetchInstanceUserListById(
          idFromSlug(props.instanceSlug)
        ),
      ]);
    };

    watchEffect(prepareInstanceContext);
  },
});
</script>
