<template>
  <div
    v-if="databaseGroup"
    v-bind="$attrs"
    class="h-full flex-1 relative flex flex-col gap-y-4"
  >
    <FeatureAttention
      :feature="PlanFeature.FEATURE_DATABASE_GROUPS"
      class="mb-4"
    />

    <div
      v-if="hasDatabaseGroupFeature && !state.editing"
      class="flex flex-row justify-end items-center flex-wrap shrink gap-x-2 gap-y-2"
    >
      <NTooltip
        v-if="hasPermissionToCreateIssue"
        :disabled="hasMatchedDatabases"
      >
        <template #trigger>
          <NButton
            :disabled="!hasMatchedDatabases"
            @click="state.showChangeDatabaseDrawer = true"
          >
            {{ $t("database.change-database") }}
          </NButton>
        </template>
        {{ $t("database-group.no-matched-databases") }}
      </NTooltip>
      <NButton v-if="allowEdit" type="primary" @click="state.editing = true">
        <template #icon>
          <EditIcon class="w-4 h-4" />
        </template>
        {{ $t("common.configure") }}
      </NButton>
    </div>

    <DatabaseGroupForm
      :readonly="!state.editing"
      :project="project"
      :database-group="databaseGroup"
      @dismiss="state.editing = false"
    />

    <AddSpecDrawer
      v-if="databaseGroup"
      v-model:show="state.showChangeDatabaseDrawer"
      :title="$t('database.change-database')"
      :project-name="project.name"
      :pre-selected-database-group="databaseGroupResourceName"
      :use-legacy-issue-flow="true"
    />
  </div>
</template>

<script lang="ts" setup>
import { EditIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import DatabaseGroupForm from "@/components/DatabaseGroup/DatabaseGroupForm.vue";
import FeatureAttention from "@/components/FeatureGuard/FeatureAttention.vue";
import { AddSpecDrawer } from "@/components/Plan";
import { featureToRef, useDBGroupStore, useProjectByName } from "@/store";
import {
  databaseGroupNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasPermissionToCreateChangeDatabaseIssueInProject } from "@/utils";

interface LocalState {
  editing: boolean;
  showChangeDatabaseDrawer: boolean;
}

const props = defineProps<{
  projectId: string;
  databaseGroupName: string;
  allowEdit: boolean;
}>();

const dbGroupStore = useDBGroupStore();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
const state = reactive<LocalState>({
  editing: false,
  showChangeDatabaseDrawer: false,
});

const databaseGroupResourceName = computed(() => {
  return `${project.value.name}/${databaseGroupNamePrefix}${props.databaseGroupName}`;
});

const databaseGroup = computed(() => {
  return dbGroupStore.getDBGroupByName(databaseGroupResourceName.value);
});

const hasPermissionToCreateIssue = computed(() => {
  return hasPermissionToCreateChangeDatabaseIssueInProject(project.value);
});

const hasMatchedDatabases = computed(
  () => (databaseGroup.value?.matchedDatabases.length ?? 0) > 0
);

const hasDatabaseGroupFeature = featureToRef(
  PlanFeature.FEATURE_DATABASE_GROUPS
);

watchEffect(async () => {
  await dbGroupStore.getOrFetchDBGroupByName(databaseGroupResourceName.value, {
    skipCache: true,
    view: DatabaseGroupView.FULL,
  });
});
</script>
