<template>
  <div class="mb-2 space-y-2">
    <div class="flex items-center gap-x-4 h-[34px]">
      <NRadioGroup
        v-if="allowFilterByEnvironment"
        v-model:value="mode"
        :disabled="loading"
        class="ml-1"
      >
        <NRadio value="environment">{{ $t("common.environment") }}</NRadio>
        <NRadio value="project">{{ $t("common.project") }}</NRadio>
      </NRadioGroup>

      <NInputGroup style="width: auto">
        <ProjectSelect
          v-if="mode === 'project' && filterTypes.includes('project')"
          :project="params.project?.id ?? UNKNOWN_ID"
          :include-default-project="false"
          :include-all="true"
          :disabled="loading"
          @update:project="changeProjectId"
        />
        <InstanceSelect
          v-if="mode === 'environment' && filterTypes.includes('instance')"
          :instance="params.instance?.id ?? UNKNOWN_ID"
          :environment="params.environment?.id"
          :include-all="true"
          :filter="instanceSupportSlowQuery"
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
          :filter="(db) => instanceSupportSlowQuery(db.instance)"
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

      <div class="flex-1 flex items-center justify-end">
        <slot name="suffix" />
      </div>
    </div>

    <div v-if="filterTypes.includes('environment')">
      <EnvironmentTabFilter
        :environment="params.environment?.id ?? UNKNOWN_ID"
        :include-all="true"
        :disabled="loading"
        @update:environment="changeEnvironmentId"
      />
    </div>
  </div>
</template>
<script lang="ts" setup>
import { computed, shallowRef, watch } from "vue";
import { NDatePicker, NInputGroup, NRadio, NRadioGroup } from "naive-ui";
import dayjs from "dayjs";

import {
  type DatabaseId,
  type EnvironmentId,
  type InstanceId,
  type ProjectId,
  UNKNOWN_ID,
} from "@/types";
import {
  useCurrentUser,
  useDatabaseStore,
  useEnvironmentStore,
  useInstanceStore,
  useProjectStore,
} from "@/store";
import { hasWorkspacePermission, instanceSupportSlowQuery } from "@/utils";
import type { FilterType, SlowQueryFilterParams } from "./types";
import {
  ProjectSelect,
  InstanceSelect,
  EnvironmentTabFilter,
  DatabaseSelect,
} from "@/components/v2";

type FilterMode = "environment" | "project";

const props = defineProps<{
  params: SlowQueryFilterParams;
  filterTypes: readonly FilterType[];
  loading?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SlowQueryFilterParams): void;
}>();

const currentUser = useCurrentUser();
const mode = shallowRef<FilterMode>("environment");

const allowFilterByEnvironment = computed(() => {
  if (!props.filterTypes.includes("mode")) {
    return false;
  }
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

// Clear unused filter params when mode changed
watch(mode, (mode) => {
  if (!props.filterTypes.includes("mode")) {
    return;
  }

  if (mode === "environment") {
    if (props.params.project) {
      update({ project: undefined, instance: undefined, database: undefined });
    }
  } else if (mode === "project") {
    if (props.params.environment) {
      update({
        environment: undefined,
        instance: undefined,
        database: undefined,
      });
    }
  }
});

watch(
  allowFilterByEnvironment,
  (allowed) => {
    if (!allowed) {
      mode.value = "project";
    }
  },
  { immediate: true }
);

const isDateDisabled = (date: number) => {
  return date > dayjs().endOf("day").valueOf();
};
</script>
