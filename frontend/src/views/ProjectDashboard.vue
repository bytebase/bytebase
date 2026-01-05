<template>
  <div class="flex flex-col gap-y-4">
    <div class="flex items-center justify-between px-4 gap-x-2">
      <AdvancedSearch
        v-model:params="state.params"
        :scope-options="scopeOptions"
        :autofocus="false"
        :placeholder="$t('project.filter-projects')"
      />
      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="['bb.projects.create']"
      >
        <NButton
          type="primary"
          :disabled="slotProps.disabled"
          @click="state.showCreateDrawer = true"
        >
          <template #icon>
            <PlusIcon class="h-4 w-4" />
          </template>
          {{ $t("quick-action.new-project") }}
        </NButton>
      </PermissionGuardWrapper>
    </div>
    <div>
      <ProjectOperations
        v-if="hasWorkspacePermissionV2('bb.projects.delete')"
        :project-list="selectedProjectList"
        @update="handleBatchOperation"
      />
      <PagedProjectTable
        ref="pagedProjectTableRef"
        session-key="bb.project-table"
        :filter="filter"
        :bordered="false"
        :footer-class="'mx-4'"
        :prevent-default="!!onRowClick"
        :show-selection="true"
        :selected-project-names="selectedProjectNames"
        @update:selected-project-names="updateSelectedProjects"
        @row-click="onRowClick"
      />
    </div>
  </div>
  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="state.showCreateDrawer"
    @close="state.showCreateDrawer = false"
  >
    <ProjectCreatePanel
      :on-created="handleCreated"
      @dismiss="state.showCreateDrawer = false"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useRouter } from "vue-router";
import AdvancedSearch, {
  useCommonSearchScopeOptions,
} from "@/components/AdvancedSearch";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import { Drawer, PagedProjectTable } from "@/components/v2";
import ProjectOperations from "@/components/v2/Model/Project/ProjectOperations.vue";
import { useProjectV1Store, useUIStateStore } from "@/store";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  getValuesFromSearchParams,
  hasWorkspacePermissionV2,
  type SearchParams,
} from "@/utils";

interface LocalState {
  params: SearchParams;
  showCreateDrawer: boolean;
  selectedProjects: Set<string>;
}

const props = defineProps<{
  onRowClick?: (project: Project) => void;
}>();

const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [],
  },
  showCreateDrawer: false,
  selectedProjects: new Set(),
});

const router = useRouter();
const projectStore = useProjectV1Store();

const pagedProjectTableRef = ref<ComponentExposed<typeof PagedProjectTable>>();

// Add label to the available scopes for filtering projects
const scopeOptions = useCommonSearchScopeOptions(["label"]);

// Extract labels from the search scopes
const selectedLabels = computed(() => {
  return getValuesFromSearchParams(state.params, "label");
});

const filter = computed(() => ({
  query: state.params.query,
  excludeDefault: true,
  labels: selectedLabels.value,
}));

const selectedProjectNames = computed(() => {
  return Array.from(state.selectedProjects);
});

const selectedProjectList = computed(() => {
  if (state.selectedProjects.size === 0) {
    return [];
  }
  return Array.from(state.selectedProjects)
    .map((name) => projectStore.getProjectByName(name))
    .filter((p): p is Project => p !== undefined);
});

const updateSelectedProjects = (projectNames: string[]) => {
  state.selectedProjects = new Set(projectNames);
};

const handleBatchOperation = () => {
  state.selectedProjects.clear();
  pagedProjectTableRef.value?.refresh();
};

onMounted(() => {
  const uiStateStore = useUIStateStore();
  if (!uiStateStore.getIntroStateByKey("project.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "project.visit",
      newState: true,
    });
  }
});

const handleCreated = async (project: Project) => {
  if (props.onRowClick) {
    return props.onRowClick(project);
  }
  const url = {
    path: `/${project.name}`,
  };
  router.push(url);
  state.showCreateDrawer = false;
};
</script>
