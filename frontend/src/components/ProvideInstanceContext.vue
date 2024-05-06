<template>
  <slot />
</template>

<script lang="ts" setup>
import { watchEffect, computed } from "vue";
import { useInstanceV1Store } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_INSTANCE_NAME } from "@/types";

const props = defineProps<{
  instanceId: string;
}>();

const instanceStore = useInstanceV1Store();

const instanceName = computed(() => `${instanceNamePrefix}${props.instanceId}`);
const instance = computed(() =>
  instanceStore.getInstanceByName(instanceName.value)
);

const prepareInstanceContext = async function () {
  const ins = await prepareInstance();

  await instanceStore.fetchInstanceRoleListByName(ins.name);
};

const prepareInstance = async () => {
  if (instance.value.name !== UNKNOWN_INSTANCE_NAME) {
    return instance.value;
  }

  const ins = await useInstanceV1Store().getOrFetchInstanceByName(
    instanceName.value
  );
  return ins;
};

watchEffect(prepareInstanceContext);
</script>
