<template>
  <slot />
</template>

<script lang="ts">
import {
  useDatabaseStore,
  useInstanceStore,
  useInstanceV1Store,
} from "@/store";
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
      const uid = String(idFromSlug(props.instanceSlug));
      await Promise.all([
        useDatabaseStore().fetchDatabaseListByInstanceId(
          idFromSlug(props.instanceSlug)
        ),
        useInstanceV1Store()
          .getOrFetchInstanceByUID(uid)
          .then((instance) => {
            return useInstanceV1Store().fetchInstanceRoleListByName(
              instance.name
            );
          }),
      ]);
    };

    watchEffect(prepareInstanceContext);
  },
});
</script>
