<template>
  <div class="w-full space-y-4">
    <FeatureAttention feature="bb.feature.database-grouping" />

    <DatabaseGroupDataTable
      :database-group-list="databaseGroupList"
      :custom-click="true"
      :show-selection="false"
      :show-project="false"
      :loading="isLoading"
      @row-click="handleDatabaseGroupClick"
      @edit="handleEditDatabaseGroup"
    />
  </div>

  <DatabaseGroupPanel
    :show="state.showDatabaseGroupPanel"
    :project="project"
    :database-group="state.editingDatabaseGroup"
    @close="state.showDatabaseGroupPanel = false"
  />

  <FeatureModal
    :open="state.showFeatureModal"
    feature="bb.feature.database-grouping"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import DatabaseGroupDataTable from "@/components/DatabaseGroup/DatabaseGroupDataTable.vue";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import { useDBGroupStore, hasFeature } from "@/store";
import type { ComposedDatabaseGroup, ComposedProject } from "@/types";
import { FeatureAttention, FeatureModal } from "../FeatureGuard";
import DatabaseGroupPanel from "./DatabaseGroupPanel.vue";

interface LocalState {
  showDatabaseGroupPanel: boolean;
  showFeatureModal: boolean;
  editingDatabaseGroup?: ComposedDatabaseGroup;
}

const props = defineProps<{
  project: ComposedProject;
}>();

const router = useRouter();
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  showFeatureModal: false,
  showDatabaseGroupPanel: false,
});
const isLoading = ref(true);

const hasDatabaseGroupFeature = computed(() => {
  return hasFeature("bb.feature.database-grouping");
});

const databaseGroupList = computed(() => {
  return dbGroupStore.getDBGroupListByProjectName(props.project.name);
});

onMounted(async () => {
  await dbGroupStore.getOrFetchDBGroupListByProjectName(props.project.name);
  isLoading.value = false;
});

const handleDatabaseGroupClick = (
  event: MouseEvent,
  databaseGroup: ComposedDatabaseGroup
) => {
  const url = router.resolve({
    name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
    params: {
      databaseGroupName: databaseGroup.databaseGroupName,
    },
  }).fullPath;
  if (event.ctrlKey || event.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};

const handleEditDatabaseGroup = (databaseGroup: ComposedDatabaseGroup) => {
  if (!hasDatabaseGroupFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  state.editingDatabaseGroup = cloneDeep(databaseGroup);
  state.showDatabaseGroupPanel = true;
};
</script>
