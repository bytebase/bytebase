<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <BannersWrapper v-if="showBanners" />
    <!-- Suspense is experimental, be aware of the potential change -->
    <Suspense>
      <template #default>
        <ProvideSQLEditorContext>
          <router-view />
        </ProvideSQLEditorContext>
      </template>
    </Suspense>
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed } from "vue";
import BannersWrapper from "@/components/BannersWrapper.vue";
import ProvideSQLEditorContext from "@/components/ProvideSQLEditorContext.vue";
import { useActuatorV1Store } from "@/store";

const actuatorStore = useActuatorV1Store();
const { pageMode } = storeToRefs(actuatorStore);

const showBanners = computed(() => {
  return pageMode.value === "BUNDLED";
});
</script>
