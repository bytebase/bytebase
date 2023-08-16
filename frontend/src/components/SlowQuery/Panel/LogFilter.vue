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
          :environment="params.environment?.uid ?? String(UNKNOWN_ID)"
          :include-all="true"
          :disabled="loading"
          @update:environment="changeEnvironmentId"
        />
      </div>

      <div class="flex items-center justify-end gap-x-3">
        <slot name="suffix" />
      </div>
    </div>

    <div class="flex items-center gap-x-4">
      <NInputGroup class="flex-1">
        <ProjectSelect
          v-if="filterTypes.includes('project')"
          :project="params.project?.uid ?? String(UNKNOWN_ID)"
          :include-default-project="canVisitDefaultProject"
          :include-all="true"
          :disabled="loading"
          @update:project="changeProjectId"
        />
        <InstanceSelect
          v-if="filterTypes.includes('instance')"
          :instance="params.instance?.uid ?? String(UNKNOWN_ID)"
          :environment="params.environment?.uid"
          :include-all="true"
          :filter="instanceFilter"
          :disabled="loading"
          @update:instance="changeInstanceId"
        />
        <DatabaseSelect
          v-if="filterTypes.includes('database')"
          :database="params.database?.uid ?? String(UNKNOWN_ID)"
          :environment="params.environment?.uid"
          :instance="params.instance?.uid"
          :project="params.project?.uid"
          :include-all="true"
          :filter="(db) => instanceFilter(db.instanceEntity)"
          :disabled="loading"
          :consistent-menu-width="false"
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
import dayjs from "dayjs";
import { NDatePicker, NInputGroup } from "naive-ui";
import { computed } from "vue";
import {
  ProjectSelect,
  InstanceSelect,
  EnvironmentTabFilter,
  DatabaseSelect,
} from "@/components/v2";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
  useSlowQueryPolicyList,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { Instance } from "@/types/proto/v1/instance_service";
import { hasWorkspacePermissionV1, instanceV1SupportSlowQuery } from "@/utils";
import type { FilterType, SlowQueryFilterParams } from "./types";

const props = defineProps<{
  params: SlowQueryFilterParams;
  filterTypes: readonly FilterType[];
  loading?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SlowQueryFilterParams): void;
}>();

const currentUserV1 = useCurrentUserV1();
const { list: policyList, ready } = useSlowQueryPolicyList();

const canVisitDefaultProject = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-database",
    currentUserV1.value.userRole
  );
});

const changeEnvironmentId = (id: string) => {
  const environment = useEnvironmentV1Store().getEnvironmentByUID(id);
  update({ environment });
};
const changeInstanceId = (uid: string | undefined) => {
  if (uid && uid !== String(UNKNOWN_ID)) {
    const instance = useInstanceV1Store().getInstanceByUID(uid);
    update({ instance });
    return;
  }
  update({ instance: undefined });
};
const changeDatabaseId = (uid: string | undefined) => {
  if (uid && uid !== String(UNKNOWN_ID)) {
    const database = useDatabaseV1Store().getDatabaseByUID(uid);
    update({ database });
    return;
  }
  update({ database: undefined });
};
const changeProjectId = (id: string | undefined) => {
  if (id && id !== String(UNKNOWN_ID)) {
    const project = useProjectV1Store().getProjectByUID(id);
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
  if (!instanceV1SupportSlowQuery(instance)) {
    return false;
  }
  const policy = policyList.value.find(
    (policy) => policy.resourceUid === instance.uid
  );
  if (!policy) {
    return false;
  }
  return !!policy.slowQueryPolicy?.active;
};

const isDateDisabled = (date: number) => {
  return date > dayjs().endOf("day").valueOf();
};
</script>
