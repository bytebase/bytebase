<template>
  <slot />
</template>

<script lang="ts" setup>
import { watchEffect } from "vue";
import { useInstanceV1Store } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  instanceId: string;
}>();

const prepareInstanceContext = async function () {
  await Promise.all([
    useInstanceV1Store()
      .getOrFetchInstanceByName(`${instanceNamePrefix}${props.instanceId}`)
      .then((instance) => {
        return useInstanceV1Store().fetchInstanceRoleListByName(instance.name);
      }),
  ]);
};

watchEffect(prepareInstanceContext);
</script>
