<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-4 my-8 gap-4"
  >
    <div v-if="create" class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center textlabel !leading-6">
        {{ $t("common.project") }}
        <RequiredStar />
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :only-userself="false"
        :selected-id="projectId"
        @select-project-id="handleProjectSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center shrink-0 textlabel !leading-6">
        {{ $t("common.database") }}
        <RequiredStar />
      </span>
      <div v-if="create" class="flex flex-row justify-start items-center">
        <EnvironmentSelect
          class="!w-60 mr-4 shrink-0"
          name="environment"
          :select-default="false"
          :selected-id="state.environmentId"
          @select-environment-id="handleEnvironmentSelect"
        />
        <DatabaseSelect
          class="!w-128"
          :selected-id="state.databaseId ?? String(UNKNOWN_ID)"
          :mode="'ALL'"
          :environment-id="state.environmentId"
          :project-id="state.projectId"
          :sync-status="'OK'"
          :customize-item="true"
          @select-database-id="handleDatabaseSelect"
        >
          <template #customizeItem="{ database }">
            <div class="flex items-center">
              <InstanceV1EngineIcon :instance="database.instanceEntity" />
              <span class="mx-2">{{ database.databaseName }}</span>
              <span class="text-gray-400">
                ({{ instanceV1Name(database.instanceEntity) }})
              </span>
            </div>
          </template>
        </DatabaseSelect>
      </div>
      <div
        v-else-if="selectedDatabase"
        class="flex flex-row justify-start items-center"
      >
        <InstanceV1EngineIcon
          :instance="selectedDatabase.instanceEntity"
          :link="false"
          class="mr-1"
        />
        {{ selectedDatabase.databaseName }}
      </div>
    </div>
    <div class="w-full flex flex-row justify-start items-start">
      <span class="flex w-40 items-start textlabel !leading-6">
        {{
          create
            ? $t("issue.grant-request.expire-days")
            : $t("issue.grant-request.expired-at")
        }}
        <RequiredStar />
      </span>
      <div v-if="create">
        <NRadioGroup
          v-model:value="state.expireDays"
          class="!grid grid-cols-6 gap-4"
          name="radiogroup"
        >
          <div
            v-for="day in expireDaysOptions"
            :key="day.value"
            class="col-span-1 flex flex-row justify-start items-center"
          >
            <NRadio :value="day.value" :label="day.label" />
          </div>
          <div class="col-span-2 flex flex-row justify-start items-center">
            <NRadio :value="-1" :label="$t('issue.grant-request.customize')" />
            <NInputNumber
              v-model:value="state.customDays"
              class="!w-24 ml-2"
              :disabled="state.expireDays !== -1"
              :min="1"
              :show-button="false"
              :placeholder="''"
            >
              <template #suffix>{{ $t("common.date.days") }}</template>
            </NInputNumber>
          </div>
        </NRadioGroup>
      </div>
      <div v-else>
        {{ state.expiredAt }}
      </div>
    </div>
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center textlabel !leading-6">
        {{ $t("issue.grant-request.export-rows") }}
        <RequiredStar />
      </span>
      <input
        v-model="state.maxRowCount"
        required
        type="number"
        class="textfield"
        :readonly="!create"
        placeholder="Max row count"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-start">
      <span class="flex w-40 items-center textlabel !leading-6">
        {{ $t("issue.grant-request.export-format") }}
        <RequiredStar />
      </span>
      <div v-if="create">
        <NRadioGroup
          v-model:value="state.exportFormat"
          class="w-full !flex flex-row justify-start items-center gap-4"
          name="radiogroup"
        >
          <NRadio :value="'CSV'" label="CSV" />
          <NRadio :value="'JSON'" label="JSON" />
        </NRadioGroup>
      </div>
      <div v-else class="flex flex-row justify-start items-start gap-4">
        {{ state.exportFormat }}
      </div>
    </div>
    <div class="w-full flex flex-row justify-start items-start">
      <span class="flex w-40 items-center shrink-0 textlabel !leading-6 mt-4"
        >SQL<RequiredStar
      /></span>
      <div class="whitespace-pre-wrap w-full overflow-hidden border">
        <MonacoEditor
          class="w-full h-[360px] py-2"
          :value="state.statement"
          :auto-focus="false"
          :language="'sql'"
          :dialect="dialect"
          :readonly="!create"
          @change="handleStatementChange"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NRadioGroup, NRadio, NInputNumber } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useIssueLogic } from "../logic";
import {
  GrantRequestContext,
  GrantRequestPayload,
  Issue,
  IssueCreate,
  PresetRoleType,
  SQLDialect,
  UNKNOWN_ID,
  dialectOfEngineV1,
} from "@/types";
import { extractUserUID, instanceV1Name, memberListInProjectV1 } from "@/utils";
import { useDatabaseV1Store, useProjectV1Store } from "@/store";
import MonacoEditor from "@/components/MonacoEditor";
import RequiredStar from "@/components/RequiredStar.vue";
import { DatabaseResource } from "./SelectDatabaseResourceForm/common";
import { convertFromCEL } from "@/utils/issue/cel";
import { InstanceV1EngineIcon } from "@/components/v2";
import DatabaseSelect from "@/components/DatabaseSelect.vue";
import { Engine } from "@/types/proto/v1/common";

interface LocalState {
  // For creating
  projectId?: string;
  environmentId?: string;
  databaseId?: string;
  selectedDatabaseResourceList: DatabaseResource[];
  expireDays: number;
  customDays: number;
  maxRowCount: number;
  exportFormat: "CSV" | "JSON";
  statement: string;
  // For reviewing
  expiredAt: string;
}

const { t } = useI18n();
const { create, issue } = useIssueLogic();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  selectedDatabaseResourceList: [],
  expireDays: 1,
  customDays: 7,
  maxRowCount: 1000,
  exportFormat: "CSV",
  statement: "",
  expiredAt: "",
});

const projectId = computed(() => {
  return create.value ? state.projectId : (issue.value as Issue).project.id;
});

const selectedDatabase = computed(() => {
  if (!state.databaseId || state.databaseId === String(UNKNOWN_ID)) {
    return undefined;
  }
  return databaseStore.getDatabaseByUID(state.databaseId);
});

const expireDaysOptions = computed(() => [
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
]);

const dialect = computed((): SQLDialect => {
  const db = selectedDatabase.value;
  return dialectOfEngineV1(db?.instanceEntity.engine ?? Engine.MYSQL);
});

onMounted(() => {
  if (create.value) {
    const projectId = String((issue.value as IssueCreate).projectId);
    if (projectId && projectId !== String(UNKNOWN_ID)) {
      handleProjectSelect(projectId);
    }
  }
});

const handleProjectSelect = async (projectId: string) => {
  if (!create.value) {
    return;
  }

  const issueCreate = issue.value as IssueCreate;
  state.projectId = projectId;
  issueCreate.projectId = parseInt(projectId, 10);
  // update issue assignee
  const project = await useProjectV1Store().getOrFetchProjectByUID(projectId);
  const memberList = memberListInProjectV1(project, project.iamPolicy);
  const ownerList = memberList.filter((member) =>
    member.roleList.includes(PresetRoleType.OWNER)
  );
  const projectOwner = head(ownerList);
  if (projectOwner) {
    const userUID = extractUserUID(projectOwner.user.name);
    issueCreate.assigneeId = Number(userUID);
  }
};

const handleEnvironmentSelect = (environmentId: string) => {
  state.environmentId = environmentId;
  const database = databaseStore.getDatabaseByUID(
    state.databaseId || String(UNKNOWN_ID)
  );
  // Unselect database if it doesn't belong to the newly selected environment.
  if (
    database &&
    database.uid !== String(UNKNOWN_ID) &&
    database.instanceEntity.environmentEntity.uid !== state.environmentId
  ) {
    state.databaseId = undefined;
  }
};

const handleDatabaseSelect = (databaseId: string) => {
  state.databaseId = databaseId;
  const database = databaseStore.getDatabaseByUID(
    state.databaseId || String(UNKNOWN_ID)
  );
  if (database && database.uid !== String(UNKNOWN_ID)) {
    state.environmentId = database.instanceEntity.environmentEntity.uid;
    handleProjectSelect(database.projectEntity.uid);
  }
};

const handleStatementChange = (value: string) => {
  state.statement = value;
};

watch(
  () => [
    state.databaseId,
    state.selectedDatabaseResourceList,
    state.expireDays,
    state.customDays,
    state.maxRowCount,
    state.exportFormat,
    state.statement,
  ],
  () => {
    if (create.value) {
      const context = (issue.value as IssueCreate)
        .createContext as GrantRequestContext;
      if (selectedDatabase.value) {
        context.databaseResources = [
          {
            databaseId: selectedDatabase.value.uid,
            databaseName: selectedDatabase.value.name,
          },
        ];
      } else {
        context.databaseResources = [];
      }
      if (state.expireDays === -1) {
        context.expireDays = state.customDays;
      } else {
        context.expireDays = state.expireDays;
      }
      context.maxRowCount = state.maxRowCount;
      context.exportFormat = state.exportFormat;
      context.statement = state.statement;
    }
  },
  {
    immediate: true,
  }
);

watch(
  create,
  async () => {
    if (!create.value) {
      const payload = ((issue.value as Issue).payload as any)
        .grantRequest as GrantRequestPayload;
      if (payload.role !== PresetRoleType.EXPORTER) {
        throw "Only support EXPORTER role";
      }

      const conditionExpression = await convertFromCEL(
        payload.condition.expression
      );
      if (
        conditionExpression.databaseResources !== undefined &&
        conditionExpression.databaseResources.length > 0
      ) {
        const resource = head(conditionExpression.databaseResources);
        if (resource) {
          state.databaseId = String(resource.databaseId);
        }
      }
      if (conditionExpression.expiredTime !== undefined) {
        state.expiredAt = new Date(
          conditionExpression.expiredTime
        ).toLocaleString();
      } else {
        state.expiredAt = "-";
      }
      if (conditionExpression.statement !== undefined) {
        state.statement = conditionExpression.statement;
      }
      if (conditionExpression.rowLimit !== undefined) {
        state.maxRowCount = conditionExpression.rowLimit;
      }
      if (conditionExpression.exportFormat !== undefined) {
        state.exportFormat = conditionExpression.exportFormat as "CSV" | "JSON";
      }
    }
  },
  {
    immediate: true,
  }
);
</script>
