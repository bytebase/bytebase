<template>
  <div class="py-4 flex flex-col">
    <div class="flex items-center justify-between px-4 pb-2 gap-x-2">
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
        :show-actions="true"
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
import { computed, onMounted, reactive, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useRoute, useRouter } from "vue-router";
import AdvancedSearch, {
  useCommonSearchScopeOptions,
} from "@/components/AdvancedSearch";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import { Drawer, PagedProjectTable } from "@/components/v2";
import ProjectOperations from "@/components/v2/Model/Project/ProjectOperations.vue";
import { useProjectV1Store, useUIStateStore } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  buildSearchParamsBySearchText,
  buildSearchTextBySearchParams,
  getValueFromSearchParams,
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

const route = useRoute();
const router = useRouter();
const projectStore = useProjectV1Store();

const pagedProjectTableRef = ref<ComponentExposed<typeof PagedProjectTable>>();

// Add label and state to the available scopes for filtering projects
// Filter state options based on permissions
const scopeOptions = computed(() => {
  const baseOptions = useCommonSearchScopeOptions(["label", "state"]).value;

  // If user doesn't have undelete permission, remove DELETED and ALL from state scope
  if (!hasWorkspacePermissionV2("bb.projects.undelete")) {
    return baseOptions.map((scope) => {
      if (scope.id === "state" && scope.options) {
        return {
          ...scope,
          options: scope.options.filter((opt) => opt.value === "ACTIVE"),
        };
      }
      return scope;
    });
  }

  return baseOptions;
});

// Extract labels from the search scopes
const selectedLabels = computed(() => {
  return getValuesFromSearchParams(state.params, "label");
});

const selectedState = computed(() => {
  const stateValue = getValueFromSearchParams(state.params, "state");
  if (stateValue === "DELETED") return State.DELETED;
  if (stateValue === "ALL") return undefined; // undefined = show all
  return State.ACTIVE; // default
});

const filter = computed(() => ({
  query: state.params.query,
  excludeDefault: true,
  labels: selectedLabels.value,
  state: selectedState.value,
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
  // Migrate old URL format (?state=archived) to new format (?q=state:archived)
  const queryState = router.currentRoute.value.query.state as string;
  if (queryState === "archived" || queryState === "all") {
    const stateValue = queryState === "archived" ? "DELETED" : "ALL";
    router.replace({
      query: { q: `state:${stateValue}` },
    });
  }

  // Initialize params from URL query
  const queryString = route.query.q as string;
  if (queryString) {
    state.params = buildSearchParamsBySearchText(queryString);
  }

  const uiStateStore = useUIStateStore();
  if (!uiStateStore.getIntroStateByKey("project.visit")) {
    uiStateStore.saveIntroStateByKey({
      key: "project.visit",
      newState: true,
    });
  }
});

// Sync params to URL query when params change
watch(
  () => state.params,
  (params) => {
    const queryString = buildSearchTextBySearchParams(params);
    const currentQuery = route.query.q as string;

    // Only update URL if query string has actually changed
    if (queryString !== currentQuery) {
      router.replace({
        query: queryString ? { q: queryString } : {},
      });
    }
  },
  { deep: true }
);

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
