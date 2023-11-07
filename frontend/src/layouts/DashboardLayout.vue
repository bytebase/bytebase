<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <BannersWrapper />
    <!-- Suspense is experimental, be aware of the potential change -->
    <Suspense>
      <template #default>
        <ProvideDashboardContext>
          <router-view name="body" />
        </ProvideDashboardContext>
      </template>
    </Suspense>
  </div>
</template>

<script lang="ts" setup>
import BannersWrapper from "@/components/BannersWrapper.vue";
import ProvideDashboardContext from "@/components/ProvideDashboardContext.vue";
</script>
