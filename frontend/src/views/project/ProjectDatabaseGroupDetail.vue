<template>
  <div
    v-if="databaseGroup"
    v-bind="$attrs"
    class="min-h-full flex-1 relative flex flex-col gap-y-4 pt-4"
  >
    <FeatureAttention
      :feature="PlanFeature.FEATURE_DATABASE_GROUPS"
      class="mb-4"
    />

    <div
      v-if="hasDatabaseGroupFeature && !state.editing"
      class="flex flex-row justify-end items-center flex-wrap shrink gap-x-2 gap-y-2"
    >
      <PermissionGuardWrapper
        v-slot="slotProps"
        :project="project"
        :permissions="[
          'bb.issues.create',
          'bb.plans.create',
          'bb.rollouts.create',
        ]"
      >
        <NTooltip
          :disabled="slotProps.disabled || hasMatchedDatabases"
        >
          <template #trigger>
            <NButton
              :disabled="slotProps.disabled || !hasMatchedDatabases"
              @click="() => {
                preCreateIssue(project.name, [databaseGroupResourceName])
              }"
            >
              {{ $t("database.change-database") }}
            </NButton>
          </template>
          {{ $t("database-group.no-matched-databases") }}
        </NTooltip>
      </PermissionGuardWrapper>

      <PermissionGuardWrapper
        v-slot="slotProps"
        :project="project"
        :permissions="[
          'bb.databaseGroups.update',
        ]"
      >
        <NButton type="primary" :disabled="slotProps.disabled" @click="state.editing = true">
          <template #icon>
            <EditIcon class="w-4 h-4" />
          </template>
          {{ $t("common.configure") }}
        </NButton>
      </PermissionGuardWrapper>
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
import DatabaseGroupForm from "@/components/DatabaseGroup/DatabaseGroupForm.vue";
import FeatureAttention from "@/components/FeatureGuard/FeatureAttention.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { preCreateIssue } from "@/components/Plan/logic/issue";
import { useBodyLayoutContext } from "@/layouts/common";
import { featureToRef, useDBGroupStore, useProjectByName } from "@/store";
import {
  databaseGroupNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

interface LocalState {
  editing: boolean;
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
});

const databaseGroupResourceName = computed(() => {
  return `${project.value.name}/${databaseGroupNamePrefix}${props.databaseGroupName}`;
});

const databaseGroup = computed(() => {
  return dbGroupStore.getDBGroupByName(databaseGroupResourceName.value);
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

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("py-0!");
</script>
