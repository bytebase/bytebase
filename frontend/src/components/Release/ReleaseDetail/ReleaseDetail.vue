<template>
  <div class="flex flex-col items-start gap-y-4 relative">
    <NavBar />
    <BasicInfo />
    <ReleaseFileTable
      :files="release.files"
      :release-type="release.type"
      :show-selection="false"
      @row-click="(_, file) => (state.selectedReleaseFile = file)"
    />
    <div class="pl-2 opacity-80">
      <ReleaseArchiveRestoreButton />
    </div>
  </div>

  <Drawer
    :show="!!state.selectedReleaseFile"
    @close="state.selectedReleaseFile = undefined"
  >
    <DrawerContent
      style="width: 75vw; max-width: calc(100vw - 8rem)"
      :title="'Release File'"
    >
      <ReleaseFileDetailPanel
        v-if="state.selectedReleaseFile"
        :release="release"
        :release-file="state.selectedReleaseFile"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useRoute } from "vue-router";
import { Drawer, DrawerContent } from "@/components/v2";
import { PROJECT_V1_ROUTE_RELEASE_DETAIL } from "@/router/dashboard/projectV1";
import type { Release_File } from "@/types/proto-es/v1/release_service_pb";
import BasicInfo from "./BasicInfo.vue";
import { provideReleaseDetailContext } from "./context";
import NavBar from "./NavBar";
import ReleaseArchiveRestoreButton from "./ReleaseArchiveRestoreButton.vue";
import ReleaseFileTable from "./ReleaseFileTable";
import ReleaseFileDetailPanel from "./ReleaseFileTable/ReleaseFileDetailPanel.vue";

interface LocalState {
  selectedReleaseFile?: Release_File;
}

const route = useRoute();
const { release } = provideReleaseDetailContext();
const state = reactive<LocalState>({});

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
