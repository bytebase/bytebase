<template>
  <Suspense>
    <template #default>
      <div
        v-if="!isValidInstanceName(instance.name)"
        class="flex items-center gap-x-2 m-4"
      >
        <BBSpin :size="20" />
        Loading instance...
      </div>
      <router-view v-else :instance-id="instanceId" />
    </template>
    <template #fallback>
      <span>Loading instance...</span>
    </template>
  </Suspense>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { BBSpin } from "@/bbkit";
import { useInstanceV1Store } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { isValidInstanceName } from "@/types";

const props = defineProps<{
  instanceId: string;
}>();

const instanceStore = useInstanceV1Store();

watchEffect(async () => {
  await instanceStore.getOrFetchInstanceByName(
    `${instanceNamePrefix}${props.instanceId}`
  );
});

const instance = computed(() =>
  instanceStore.getInstanceByName(`${instanceNamePrefix}${props.instanceId}`)
);
</script>
