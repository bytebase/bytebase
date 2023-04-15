<template>
  <div class="mb-2 space-y-2">
    <div
      v-if="
        filterTypes.includes('project') || filterTypes.includes('environment')
      "
      class="flex items-center gap-x-4"
    >
      <div class="flex-1 flex items-center gap-x-4">
        <ProjectSelect
          v-if="filterTypes.includes('project')"
          :project="params.project?.id ?? UNKNOWN_ID"
          :include-default-project="canVisitDefaultProject"
          :include-all="true"
          :disabled="loading"
          @update:project="changeProjectId"
        />

        <EnvironmentTabFilter
          v-if="filterTypes.includes('environment')"
          class="flex-1"
          :environment="params.environment?.id ?? UNKNOWN_ID"
          :include-all="true"
          :disabled="loading"
          @update:environment="changeEnvironmentId"
        />
      </div>

      <div class="flex items-center justify-end">
        <slot name="suffix" />
      </div>
    </div>

    <div class="flex items-center gap-x-4">
      <NInputGroup class="flex-1">
        <InstanceSelect
          v-if="filterTypes.includes('instance')"
          :instance="params.instance?.id ?? UNKNOWN_ID"
          :environment="params.environment?.id"
          :include-all="true"
          :filter="instanceFilter"
          :disabled="loading"
          @update:instance="changeInstanceId"
        />
        <DatabaseSelect
          v-if="filterTypes.includes('database')"
          :database="params.database?.id ?? UNKNOWN_ID"
          :environment="params.environment?.id"
          :instance="params.instance?.id"
          :project="params.project?.id"
          :include-all="true"
          :filter="(db) => instanceFilter(db.instance)"
          :disabled="loading"
          @update:database="changeDatabaseId"
        />
        <NDatePicker
          v-if="filterTypes.includes('time-range')"
          :value="params.timeRange"
          :disabled="loading"
          :is-date-disabled="isDateDisabled"
          type="daterange"
          clearable
          style="width: 16rem"
          @update:value="changeTimeRange"
        />
      </NInputGroup>

      <div
        v-if="
          !filterTypes.includes('project') &&
          !filterTypes.includes('environment')
        "
        class="flex items-center justify-end"
      >
        <slot name="suffix" />
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { computed } from "vue";
import { NDatePicker, NInputGroup } from "naive-ui";
import dayjs from "dayjs";

import {
  type DatabaseId,
  type EnvironmentId,
  type InstanceId,
  type ProjectId,
  UNKNOWN_ID,
  Instance,
  SlowQueryPolicyPayload,
} from "@/types";
import {
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentStore,
  useInstanceStore,
  useProjectStore,
  useSlowQueryPolicyList,
} from "@/store";
import { hasWorkspacePermission, instanceSupportSlowQuery } from "@/utils";
import type { FilterType, SlowQueryFilterParams } from "./types";
import {
  ProjectSelect,
  InstanceSelect,
  EnvironmentTabFilter,
  DatabaseSelect,
} from "@/components/v2";

const props = defineProps<{
  params: SlowQueryFilterParams;
  filterTypes: readonly FilterType[];
  loading?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SlowQueryFilterParams): void;
}>();

const currentUser = useCurrentUser();
const policyList = useSlowQueryPolicyList();

const canVisitDefaultProject = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-database",
    currentUser.value.role
  );
});

const changeEnvironmentId = (id: EnvironmentId) => {
  const environment = useEnvironmentStore().getEnvironmentById(id);
  update({ environment });
};
const changeInstanceId = (id: InstanceId | undefined) => {
  if (id && id !== UNKNOWN_ID) {
    const instance = useInstanceStore().getInstanceById(id ?? UNKNOWN_ID);
    update({ instance });
    return;
  }
  update({ instance: undefined });
};
const changeDatabaseId = (id: DatabaseId | undefined) => {
  if (id && id !== UNKNOWN_ID) {
    const database = useDatabaseStore().getDatabaseById(id ?? UNKNOWN_ID);
    update({ database });
    return;
  }
  update({ database: undefined });
};
const changeProjectId = (id: ProjectId | undefined) => {
  if (id && id !== UNKNOWN_ID) {
    const project = useProjectStore().getProjectById(id);
    update({ project });
    return;
  }
  update({ project: undefined });
};
const changeTimeRange = (timeRange: [number, number] | null) => {
  if (!timeRange) {
    update({ timeRange: undefined });
    return;
  }
  update({ timeRange });
};

const update = (params: Partial<SlowQueryFilterParams>) => {
  emit("update:params", {
    ...props.params,
    ...params,
  });
};

const instanceFilter = (instance: Instance) => {
  if (!instanceSupportSlowQuery(instance)) {
    return false;
  }
  const policy = policyList.value.find(
    (policy) => policy.resourceId === instance.id
  );
  if (!policy) {
    return false;
  }
  const payload = policy.payload as SlowQueryPolicyPayload;
  return payload.active;
};

const isDateDisabled = (date: number) => {
  return date > dayjs().endOf("day").valueOf();
};
</script>
