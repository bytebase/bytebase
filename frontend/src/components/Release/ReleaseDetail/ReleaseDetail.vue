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
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { Drawer, DrawerContent } from "@/components/v2";
import { PROJECT_V1_ROUTE_RELEASE_DETAIL } from "@/router/dashboard/projectV1";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { Release_File } from "@/types/proto-es/v1/release_service_pb";
import { setDocumentTitle } from "@/utils";
import BasicInfo from "./BasicInfo.vue";
import { provideReleaseDetailContext } from "./context";
import NavBar from "./NavBar";
import ReleaseFileTable from "./ReleaseFileTable";
import ReleaseFileDetailPanel from "./ReleaseFileTable/ReleaseFileDetailPanel.vue";

interface LocalState {
  selectedReleaseFile?: Release_File;
}

const { t } = useI18n();
const route = useRoute();
const { release } = provideReleaseDetailContext();
const state = reactive<LocalState>({});

const projectId = computed(() => {
  const parts = release.value.name.split("/");
  const idx = parts.indexOf("projects");
  return idx >= 0 ? parts[idx + 1] : "";
});
const projectResourceName = computed(
  () => `${projectNamePrefix}${projectId.value}`
);
const { project } = useProjectByName(projectResourceName);

watch(
  [() => route.name, () => project.value.title],
  () => {
    if (route.name === PROJECT_V1_ROUTE_RELEASE_DETAIL) {
      setDocumentTitle(t("release.releases"), project.value.title);
    }
  },
  { immediate: true }
);
</script>
