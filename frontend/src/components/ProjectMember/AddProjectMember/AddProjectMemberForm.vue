<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-4">
    <div class="w-full">
      <div class="flex items-center justify-between">
        {{ $t("project.members.select-users") }}

        <NButton v-if="allowRemove" text @click="$emit('remove')">
          <template #icon>
            <heroicons:trash class="w-4 h-4" />
          </template>
        </NButton>
      </div>
      <UserSelect
        v-model:users="state.userUidList"
        class="mt-2"
        style="width: 100%"
        :multiple="true"
        :include-all="false"
      />
    </div>
    <div class="w-full">
      <span>{{ $t("project.members.assign-role") }}</span>
      <ProjectMemberRoleSelect v-model:role="state.role" class="mt-2" />
    </div>

    <div v-if="state.role === 'roles/QUERIER'" class="w-full">
      <span class="block mb-2">{{ $t("common.databases") }}</span>
      <QuerierDatabaseResourceForm
        :project-id="project.uid"
        :database-resources="state.databaseResources"
        @update:condition="state.databaseResourceCondition = $event"
        @update:database-resources="state.databaseResources = $event"
      />
    </div>
    <template v-if="state.role === 'roles/EXPORTER'">
      <div class="w-full">
        <span class="block mb-2">{{ $t("common.database") }}</span>
        <DatabaseSelect
          class="!w-full"
          :project="project.uid"
          :database="state.databaseId"
          @update:database="state.databaseId = $event"
        />
      </div>
      <div class="w-full">
        <span class="block mb-2">{{
          $t("issue.grant-request.export-method")
        }}</span>
        <ExporterDatabaseResourceForm
          class="w-full"
          :project-id="project.uid"
          :database-id="state.databaseId"
          :database-resources="state.databaseResources"
          @update:condition="state.databaseResourceCondition = $event"
          @update:database-resources="state.databaseResources = $event"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <span class="mb-2">
          {{ $t("issue.grant-request.export-rows") }}
        </span>
        <input
          v-model="state.maxRowCount"
          required
          type="number"
          class="textfield"
          placeholder="Max row count"
        />
      </div>
    </template>

    <div class="w-full">
      <span>{{ $t("common.expiration") }}</span>
      <ExpirationSelector
        class="mt-2"
        :options="expireDaysOptions"
        :value="state.expireDays"
        @update="state.expireDays = $event"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */

import dayjs from "dayjs";
import { computed, nextTick, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useUserStore } from "@/store";
import { ComposedProject, DatabaseResource } from "@/types";
import { Binding } from "@/types/proto/v1/iam_policy";
import { Expr } from "@/types/proto/google/type/expr";
import ProjectMemberRoleSelect from "@/components/v2/Select/ProjectMemberRoleSelect.vue";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import QuerierDatabaseResourceForm from "@/components/Issue/panel/RequestQueryPanel/DatabaseResourceForm/index.vue";
import ExporterDatabaseResourceForm from "@/components/Issue/panel/RequestExportPanel/ExportResourceForm/index.vue";
import { DatabaseSelect } from "@/components/v2";

const props = defineProps<{
  project: ComposedProject;
  binding: Binding;
  allowRemove: boolean;
}>();

defineEmits<{
  (event: "remove"): void;
}>();

interface LocalState {
  userUidList: string[];
  role?: string;
  expireDays: number;
  // Querier and exporter options.
  databaseResourceCondition?: string;
  databaseResources?: DatabaseResource[];
  // Exporter options.
  maxRowCount: number;
  databaseId?: string;
}

const { t } = useI18n();
const userStore = useUserStore();
const state = reactive<LocalState>({
  userUidList: [],
  expireDays: 7,
  // Exporter options.
  maxRowCount: 1000,
});

const expireDaysOptions = computed(() => {
  if (state.role === "roles/EXPORTER") {
    return [
      {
        value: 1,
        label: t("common.date.days", { days: 1 }),
      },
      {
        value: 3,
        label: t("common.date.days", { days: 3 }),
      },
      {
        value: 7,
        label: t("common.date.days", { days: 7 }),
      },
      {
        value: 15,
        label: t("common.date.days", { days: 15 }),
      },
    ];
  }
  return [
    {
      value: 7,
      label: t("common.date.days", { days: 7 }),
    },
    {
      value: 30,
      label: t("common.date.days", { days: 30 }),
    },
    {
      value: 60,
      label: t("common.date.days", { days: 60 }),
    },
    {
      value: 90,
      label: t("common.date.days", { days: 90 }),
    },
    {
      value: 180,
      label: t("common.date.months", { months: 6 }),
    },
    {
      value: 365,
      label: t("common.date.years", { years: 1 }),
    },
    {
      value: 0,
      label: t("project.members.never-expires"),
    },
  ];
});

watch(
  () => state.role,
  () => {
    state.databaseResourceCondition = undefined;
    state.databaseResources = undefined;
    nextTick(() => {
      state.expireDays = expireDaysOptions.value[0].value;
    });
  },
  {
    immediate: true,
  }
);

watch(
  () => state,
  () => {
    if (state.userUidList) {
      props.binding.members = state.userUidList.map((uid) => {
        const user = userStore.getUserById(uid);
        return `user:${user!.email}`;
      });
    }
    if (state.role) {
      props.binding.role = state.role;
    }
    const expression: string[] = [];
    if (state.expireDays > 0) {
      expression.push(
        `request.time < timestamp("${dayjs()
          .add(state.expireDays, "days")
          .toISOString()}")`
      );
    }
    if (state.role === "roles/QUERIER") {
      if (state.databaseResourceCondition) {
        expression.push(state.databaseResourceCondition);
      }
    }
    if (state.role === "roles/EXPORTER") {
      if (state.databaseResourceCondition) {
        expression.push(state.databaseResourceCondition);
      }
      if (state.maxRowCount) {
        expression.push(`request.rows_limit <= ${state.maxRowCount}`);
      }
    }
    if (expression.length > 0) {
      props.binding.condition = Expr.create({
        expression: expression.join(" && "),
      });
    }
  },
  {
    deep: true,
  }
);
</script>
