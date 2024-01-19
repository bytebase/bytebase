<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <ProvideInstanceContext :instance-id="instanceId">
        <router-view :instance-id="instanceId" />
      </ProvideInstanceContext>
    </template>
    <template #fallback>
      <span>Loading instance...</span>
    </template>
  </Suspense>
</template>

<script lang="ts" setup>
import { watchEffect } from "vue";
import ProvideInstanceContext from "@/components/ProvideInstanceContext.vue";
import { useInstanceV1Store } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  instanceId: string;
}>();

watchEffect(async () => {
  await useInstanceV1Store().getOrFetchInstanceByName(
    `${instanceNamePrefix}${props.instanceId}`
  );
});
</script>
