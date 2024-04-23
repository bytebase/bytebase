<template>
  <div v-if="loading" class="flex items-center gap-x-2 m-4">
    <BBSpin :size="20" />
    Loading instance...
  </div>
  <slot v-else />
</template>

<script lang="ts" setup>
import { watchEffect, ref, computed } from "vue";
import { useInstanceV1Store } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_INSTANCE_NAME } from "@/types";

const props = defineProps<{
  instanceId: string;
}>();

const instanceStore = useInstanceV1Store();
const loading = ref(false);

const instanceName = computed(() => `${instanceNamePrefix}${props.instanceId}`);
const instance = computed(() =>
  instanceStore.getInstanceByName(instanceName.value)
);

const prepareInstanceContext = async function () {
  const ins = await prepareInstance();
  await useInstanceV1Store().fetchInstanceRoleListByName(ins.name);
};

const prepareInstance = async () => {
  if (instance.value.name !== UNKNOWN_INSTANCE_NAME) {
    return instance.value;
  }

  loading.value = true;
  try {
    const ins = await useInstanceV1Store().getOrFetchInstanceByName(
      instanceName.value
    );
    return ins;
  } finally {
    loading.value = false;
  }
};

watchEffect(prepareInstanceContext);
</script>
