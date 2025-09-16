<template>
  <div
    v-if="databaseGroup"
    v-bind="$attrs"
    class="h-full flex-1 relative space-y-4"
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
            @click="
              previewDatabaseGroupIssue('bb.issue.database.schema.update')
            "
          >
            {{ $t("database.edit-schema") }}
          </NButton>
        </template>
        {{ $t("database-group.no-matched-databases") }}
      </NTooltip>
      <NTooltip
        v-if="hasPermissionToCreateIssue"
        :disabled="hasMatchedDatabases"
      >
        <template #trigger>
          <NButton
            :disabled="!hasMatchedDatabases"
            @click="previewDatabaseGroupIssue('bb.issue.database.data.update')"
          >
            {{ $t("database.change-data") }}
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
  </div>
</template>

<script lang="ts" setup>
import { EditIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useRouter } from "vue-router";
import DatabaseGroupForm from "@/components/DatabaseGroup/DatabaseGroupForm.vue";
import FeatureAttention from "@/components/FeatureGuard/FeatureAttention.vue";
import { useDBGroupStore, useProjectByName, featureToRef } from "@/store";
import {
  databaseGroupNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasPermissionToCreateChangeDatabaseIssueInProject } from "@/utils";
import { generateDatabaseGroupIssueRoute } from "@/utils/databaseGroup/issue";

interface LocalState {
  editing: boolean;
}

const props = defineProps<{
  projectId: string;
  databaseGroupName: string;
  allowEdit: boolean;
}>();

const router = useRouter();
const dbGroupStore = useDBGroupStore();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
const state = reactive<LocalState>({
  editing: false,
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
    view: DatabaseGroupView.BASIC,
  });
});

const previewDatabaseGroupIssue = (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update"
) => {
  if (!databaseGroup.value) {
    return;
  }
  const issueRoute = generateDatabaseGroupIssueRoute(type, databaseGroup.value);
  router.push(issueRoute);
};
</script>
