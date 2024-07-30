<template>
  <slot />
</template>

<script lang="ts" setup>
import { watchEffect, computed } from "vue";
import { useInstanceV1Store } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { isValidInstanceName } from "@/types";

const props = defineProps<{
  instanceId: string;
}>();

const instanceStore = useInstanceV1Store();

const instanceName = computed(() => `${instanceNamePrefix}${props.instanceId}`);
const instance = computed(() =>
  instanceStore.getInstanceByName(instanceName.value)
);

const prepareInstanceContext = async function () {
  await prepareInstance();
};

const prepareInstance = async () => {
  if (isValidInstanceName(instance.value.name)) {
    return instance.value;
  }

  const ins = await useInstanceV1Store().getOrFetchInstanceByName(
    instanceName.value
  );
  return ins;
};

watchEffect(prepareInstanceContext);
</script>
