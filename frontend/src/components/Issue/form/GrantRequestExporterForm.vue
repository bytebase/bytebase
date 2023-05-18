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
          :selected-id="state.databaseId"
          :mode="'ALL'"
          :environment-id="state.environmentId"
          :project-id="state.projectId"
          :sync-status="'OK'"
          :customize-item="true"
          @select-database-id="handleDatabaseSelect"
        >
          <template #customizeItem="{ database }">
            <div class="flex items-center">
              <InstanceEngineIcon :instance="database.instance" />
              <span class="mx-2">{{ database.name }}</span>
              <span class="text-gray-400">({{ database.instance.name }})</span>
            </div>
          </template>
        </DatabaseSelect>
      </div>
      <div
        v-else-if="selectedDatabase"
        class="flex flex-row justify-start items-center"
      >
        <InstanceEngineIcon
          class="mr-1"
          :instance="selectedDatabase.instance"
        />
        {{ selectedDatabase.name }}
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
import { computed, onMounted, reactive, watch } from "vue";
import { useIssueLogic } from "../logic";
import {
  DatabaseId,
  GrantRequestContext,
  GrantRequestPayload,
  Issue,
  IssueCreate,
  PresetRoleType,
  SQLDialect,
  UNKNOWN_ID,
  dialectOfEngine,
} from "@/types";
import {
  getDatabaseIdByName,
  memberListInProjectV1,
  parseConditionExpressionString,
} from "@/utils";
import {
  convertUserToPrincipal,
  useDatabaseStore,
  useProjectV1Store,
} from "@/store";
import MonacoEditor from "@/components/MonacoEditor";
import RequiredStar from "@/components/RequiredStar.vue";

interface LocalState {
  // For creating
  projectId?: string;
  environmentId?: string;
  databaseId?: DatabaseId;
  maxRowCount: number;
  exportFormat: "CSV" | "JSON";
  statement: string;
}

const { create, issue } = useIssueLogic();
const databaseStore = useDatabaseStore();
const state = reactive<LocalState>({
  maxRowCount: 1000,
  exportFormat: "CSV",
  statement: "",
});

const projectId = computed(() => {
  return create.value ? state.projectId : (issue.value as Issue).project.id;
});

const selectedDatabase = computed(() => {
  if (!state.databaseId || state.databaseId === UNKNOWN_ID) {
    return undefined;
  }
  return databaseStore.getDatabaseById(state.databaseId as DatabaseId);
});

const dialect = computed((): SQLDialect => {
  const db = selectedDatabase.value;
  return dialectOfEngine(db?.instance.engine || "MYSQL");
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
    const ownerPrincipal = convertUserToPrincipal(projectOwner.user);
    issueCreate.assigneeId = ownerPrincipal.id;
  }
};

const handleEnvironmentSelect = (environmentId: string) => {
  state.environmentId = environmentId;
  const database = databaseStore.getDatabaseById(
    state.databaseId || UNKNOWN_ID
  );
  // Unselect database if it doesn't belong to the newly selected environment.
  if (
    database &&
    String(database.id) !== String(UNKNOWN_ID) &&
    String(database.instance.environment.id) !== state.environmentId
  ) {
    state.databaseId = undefined;
  }
};

const handleDatabaseSelect = (databaseId: DatabaseId) => {
  state.databaseId = databaseId;
  const database = databaseStore.getDatabaseById(
    state.databaseId || UNKNOWN_ID
  );
  if (database && String(database.id) !== String(UNKNOWN_ID)) {
    state.environmentId = String(database.instance.environment.id);
    handleProjectSelect(String(database.projectId));
  }
};

const handleStatementChange = (value: string) => {
  state.statement = value;
};

watch(
  () => [
    state.databaseId,
    state.maxRowCount,
    state.exportFormat,
    state.statement,
  ],
  () => {
    if (create.value) {
      const context = (issue.value as IssueCreate)
        .createContext as GrantRequestContext;
      if (state.databaseId) {
        context.databases = [state.databaseId as string];
      } else {
        context.databases = [];
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

      const conditionExpression = parseConditionExpressionString(
        payload.condition.expression
      );
      if (conditionExpression.statement) {
        state.statement = conditionExpression.statement;
      }
      if (
        conditionExpression.databases &&
        conditionExpression.databases.length > 0
      ) {
        const databaseIdList = [];
        for (const databaseName of conditionExpression.databases) {
          const id = await getDatabaseIdByName(databaseName);
          if (id && id !== UNKNOWN_ID) {
            databaseIdList.push(id);
          }
        }
        state.databaseId = head(databaseIdList);
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
