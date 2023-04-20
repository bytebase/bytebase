<template>
  <div class="w-full">
    <div class="w-full flex flex-row justify-start items-start gap-4">
      <span>Source schema</span>
      <div>
        <div>
          <span>Project - </span>
          <span>{{ project.name }}</span>
        </div>
        <div>
          <span>Environment - </span>
          <span>{{ environment.name }}</span>
        </div>
      </div>
      <div>
        <div>
          <span>Database - </span>
          <span>{{ database.name }}</span>
        </div>
        <div>
          <span>Schema version - </span>
          <span>{{ sourceSchema.migrationHistory.version }}</span>
        </div>
      </div>
    </div>

    <Splitpanes
      class="default-theme border rounded-lg w-full h-128 flex flex-row overflow-hidden mt-4"
    >
      <Pane min-size="10" size="20">
        <div
          class="w-full border-b p-2 px-3 flex flex-row justify-between items-center"
        >
          <span>Target databases</span>
          <button @click="state.showSelectDatabasePanel = true">select</button>
        </div>
        <div>
          <span v-for="id of state.selectedDatabaseIdList" :key="id">
            {{ id }}
          </span>
        </div>
      </Pane>
      <Pane min-size="60" size="80">
        <main class="col-span-8 shrink p-2 px-3">
          12222222 \n
          <p>12312</p>
          2222
        </main>
      </Pane>
    </Splitpanes>
  </div>

  <TargetDatabasesSelectPanel
    v-if="state.showSelectDatabasePanel"
    :project-id="projectId"
    :selected-database-id-list="state.selectedDatabaseIdList"
    @close="state.showSelectDatabasePanel = false"
    @update="handleSelectedDatabaseIdListChanged"
  />
</template>

<script lang="ts" setup>
import { PropType, computed, reactive } from "vue";
import { Splitpanes, Pane } from "splitpanes";
import {
  useDatabaseStore,
  useEnvironmentStore,
  useProjectStore,
} from "@/store";
import {
  DatabaseId,
  EnvironmentId,
  MigrationHistory,
  ProjectId,
} from "@/types";
import TargetDatabasesSelectPanel from "./TargetDatabasesSelectPanel.vue";

interface SourceSchema {
  environmentId: EnvironmentId;
  databaseId: DatabaseId;
  migrationHistory: MigrationHistory;
}

interface LocalState {
  showSelectDatabasePanel: boolean;
  selectedDatabaseIdList: DatabaseId[];
}

const props = defineProps({
  projectId: {
    type: Number as PropType<ProjectId>,
    required: true,
  },
  sourceSchema: {
    type: Object as PropType<SourceSchema>,
    required: true,
  },
});

const projectStore = useProjectStore();
const environmentStore = useEnvironmentStore();
const databaseStore = useDatabaseStore();
const state = reactive<LocalState>({
  showSelectDatabasePanel: false,
  selectedDatabaseIdList: [],
});

const project = computed(() => {
  return projectStore.getProjectById(props.projectId);
});
const environment = computed(() => {
  return environmentStore.getEnvironmentById(props.sourceSchema.environmentId);
});
const database = computed(() => {
  return databaseStore.getDatabaseById(props.sourceSchema.databaseId);
});

const handleSelectedDatabaseIdListChanged = (databaseIdList: DatabaseId[]) => {
  state.selectedDatabaseIdList = databaseIdList;
  state.showSelectDatabasePanel = false;
};
</script>
