<template>
  <!-- eslint-disable vue/no-mutating-props -->

  <div class="flex items-center py-1">
    <label for="project" class="textlabel mr-4">
      {{ $t("common.projects") }}
    </label>
    <div class="w-64">
      <ProjectSelect
        id="project"
        class="mt-1"
        name="project"
        :mode="ProjectMode.Tenant"
        :selected-id="props.state.tenantProjectId"
        @select-project-id="(id: number) => props.state.tenantProjectId = id"
      />
    </div>
  </div>
  <div v-if="state.tenantProjectId">
    <ProjectTenantView
      :state="state"
      :database-list="filteredDatabaseList"
      :environment-list="environmentList"
      :project="project"
      @dismiss="$emit('dismiss')"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { Database, Environment, Project, ProjectId } from "../../types";
import ProjectSelect, { Mode as ProjectMode } from "../ProjectSelect.vue";
import ProjectTenantView, {
  State as ProjectTenantState,
} from "./ProjectTenantView.vue";
import { useProjectStore } from "@/store";

export type State = ProjectTenantState & {
  tenantProjectId: ProjectId | undefined;
};

const props = defineProps<{
  state: State;
  databaseList: Database[];
  environmentList: Environment[];
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();

const projectStore = useProjectStore();

const project = computed(() => {
  return projectStore.getProjectById(
    props.state.tenantProjectId as number
  ) as Project;
});

const filteredDatabaseList = computed(() => {
  if (!props.state.tenantProjectId) return [];
  return props.databaseList.filter(
    (db) => db.project.id === props.state.tenantProjectId
  );
});
</script>
