<template>
  <!-- Suspense is experimental, be aware of the potential change -->
  <Suspense>
    <template #default>
      <ProvideInstanceContext :instance-slug="instanceSlug">
        <router-view :instance-slug="instanceSlug" />
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
import { idFromSlug } from "@/utils";

const props = defineProps<{
  instanceSlug: string;
}>();

watchEffect(async () => {
  await useInstanceV1Store().getOrFetchInstanceByUID(
    idFromSlug(props.instanceSlug)
  );
});
</script>
