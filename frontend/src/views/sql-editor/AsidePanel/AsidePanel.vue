<template>
  <div
    ref="containerRef"
    class="h-full flex flex-row overflow-hidden"
    :data-width="containerWidth"
  >
    <div class="h-full border-r shrink-0">
      <GutterBar size="medium" />
    </div>
    <div
      v-if="asidePanelTab === 'SCHEMA' && !isDisconnected"
      class="h-full border-r shrink-0"
    >
      <ActionBar />
    </div>
    <div class="h-full flex-1 flex flex-col overflow-hidden">
      <div class="flex flex-row items-center gap-x-1 px-1 py-1 border-b">
        <ProjectSelect
          style="width: 100%"
          class="project-select"
          :menu-props="{ class: 'project-select-menu' }"
          :value="project"
          :include-all="false"
          :clearable="false"
          :include-default-project="allowAccessDefaultProject"
          :loading="!projectContextReady"
          @update:value="handleSwitchProject($event as (string | undefined))"
        >
          <template #empty>
            <div class="text-sm text-control-placeholder flex flex-col gap-1">
              <p>
                {{ $t("sql-editor.no-project.not-member-of-any-projects") }}
                <RouterLink
                  v-if="allowCreateProject"
                  :to="{
                    name: PROJECT_V1_ROUTE_DASHBOARD,
                    hash: '#new',
                  }"
                >
                  {{ $t("sql-editor.no-project.go-to-create") }}
                </RouterLink>
              </p>
              <p v-if="!allowCreateProject">
                {{
                  $t("sql-editor.no-project.contact-the-admin-to-grant-access")
                }}
              </p>
            </div>
          </template>
        </ProjectSelect>
      </div>

      <div class="flex-1 flex flex-row overflow-hidden">
        <div class="h-full flex-1 flex flex-col pt-1 overflow-hidden">
          <WorksheetPane v-if="asidePanelTab === 'WORKSHEET'" />
          <SchemaPane v-if="asidePanelTab === 'SCHEMA'" />
          <HistoryPane v-if="asidePanelTab === 'HISTORY'" />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { storeToRefs } from "pinia";
import { computed, ref } from "vue";
import { ProjectSelect } from "@/components/v2";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { useSQLEditorStore, useSQLEditorTabStore } from "@/store";
import { defaultProject, isValidProjectName } from "@/types";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";
import { useSQLEditorContext } from "../context";
import ActionBar from "./ActionBar";
import GutterBar from "./GutterBar";
import HistoryPane from "./HistoryPane";
import SchemaPane from "./SchemaPane";
import WorksheetPane from "./WorksheetPane";

const editorStore = useSQLEditorStore();
const { asidePanelTab, maybeSwitchProject } = useSQLEditorContext();
const { isDisconnected } = storeToRefs(useSQLEditorTabStore());

const { project, projectContextReady } = storeToRefs(editorStore);
const containerRef = ref<HTMLDivElement>();
const { width: containerWidth } = useElementSize(containerRef);

const allowAccessDefaultProject = computed(() => {
  return hasProjectPermissionV2(defaultProject(), "bb.projects.get");
});
const allowCreateProject = computed(() => {
  return hasWorkspacePermissionV2("bb.projects.create");
});

const handleSwitchProject = (name: string | undefined) => {
  if (!name || !isValidProjectName(name)) {
    editorStore.setProject("");
  } else {
    maybeSwitchProject(name);
  }
};
</script>

<style lang="postcss" scoped>
.project-select :deep(.n-base-selection) {
  --n-height: 30px !important;
}
</style>
<style lang="postcss">
.project-select-menu .n-base-select-menu__empty {
  padding: 12px 16px !important;
}
</style>
