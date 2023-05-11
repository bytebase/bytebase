<template>
  <div v-if="ready" class="mb-2 space-y-2">
    <div
      v-if="
        filterTypes.includes('project') || filterTypes.includes('environment')
      "
      class="flex items-center gap-x-4"
    >
      <div class="flex-1 flex items-center gap-x-4">
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
        <ProjectSelect
          v-if="filterTypes.includes('project')"
          :project="params.project?.id ?? UNKNOWN_ID"
          :include-default-project="canVisitDefaultProject"
          :include-all="true"
          :disabled="loading"
          @update:project="changeProjectId"
        />
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
          :value="params.fromTime"
          :disabled="loading"
          :is-date-disabled="isDateDisabled"
          :placeholder="$t('slow-query.filter.from-date')"
          type="date"
          clearable
          format="yyyy-MM-dd z"
          style="width: 12rem"
          @update:value="changeFromTime($event)"
        />
        <NDatePicker
          v-if="filterTypes.includes('time-range')"
          :value="params.toTime"
          :disabled="loading"
          :is-date-disabled="isDateDisabled"
          :placeholder="$t('slow-query.filter.to-date')"
          type="date"
          clearable
          format="yyyy-MM-dd z"
          style="width: 12rem"
          @update:value="changeToTime($event)"
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
const { list: policyList, ready } = useSlowQueryPolicyList();

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
const changeTime = (
  fromTime: number | undefined,
  toTime: number | undefined
) => {
  if (fromTime && toTime && fromTime > toTime) {
    // Swap if from > to
    changeTime(toTime, fromTime);
    return;
  }
  if (fromTime) {
    // fromTime is the start of the day
    fromTime = dayjs(fromTime).startOf("day").valueOf();
  }
  if (toTime) {
    // toTime is the end of the day
    toTime = dayjs(toTime).endOf("day").valueOf();
  }
  update({ fromTime, toTime });
};
const changeFromTime = (fromTime: number | undefined) => {
  const { toTime } = props.params;
  changeTime(fromTime, toTime);
};
const changeToTime = (toTime: number | undefined) => {
  const { fromTime } = props.params;
  changeTime(fromTime, toTime);
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
    (policy) => policy.resourceUid == instance.id
  );
  if (!policy) {
    return false;
  }
  return policy.slowQueryPolicy?.active;
};

const isDateDisabled = (date: number) => {
  return date > dayjs().endOf("day").valueOf();
};
</script>
