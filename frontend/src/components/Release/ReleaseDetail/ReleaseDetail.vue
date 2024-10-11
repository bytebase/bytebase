<template>
  <div class="flex flex-col items-stretch gap-y-4 relative">
    <NavBar />
    <ReleaseFileTable />
  </div>
</template>

<script lang="ts" setup>
import { computed, watch } from "vue";
import { useRoute } from "vue-router";
import { PROJECT_V1_ROUTE_RELEASE_DETAIL } from "@/router/dashboard/projectV1";
import NavBar from "./NavBar";
import ReleaseFileTable from "./ReleaseFileTable";
import { provideReleaseDetailContext } from "./context";

const route = useRoute();
const { release } = provideReleaseDetailContext();

const documentTitle = computed(() => {
  if (route.name !== PROJECT_V1_ROUTE_RELEASE_DETAIL) {
    return undefined;
  }
  return release.value.title;
});

watch(
  documentTitle,
  (title) => {
    if (title) {
      document.title = title;
    }
  },
  { immediate: true }
);
</script>
