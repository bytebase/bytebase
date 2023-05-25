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
    <div class="w-full flex flex-row justify-start items-start">
      <span class="flex w-40 items-center textlabel shrink-0 !leading-6">
        {{ $t("db.tables") }}
        <RequiredStar />
      </span>
      <div v-if="create">
        <NRadioGroup
          v-model:value="state.allTables"
          class="w-full !flex flex-row justify-start items-start gap-4"
          name="radiogroup"
        >
          <NRadio
            class="!leading-6 whitespace-nowrap"
            :value="true"
            :label="$t('issue.grant-request.all-tables')"
          />
          <div class="flex flex-row justify-start flex-wrap gap-y-2">
            <NRadio
              class="!leading-6"
              :value="false"
              :disabled="!state.projectId || !state.databaseId"
              :label="$t('issue.grant-request.manually-select')"
              @click="handleManuallySelectClick"
            />
            <button
              v-if="state.projectId && state.databaseId"
              class="ml-2 normal-link h-6 disabled:cursor-not-allowed"
              @click="state.showSelectDatabasePanel = true"
            >
              {{ $t("common.select") }}
            </button>
            <div
              v-if="state.selectedDatabaseResourceList.length > 0"
              class="ml-6 flex flex-row justify-start items-start flex-wrap gap-2 gap-x-4"
            >
              <div
                v-for="databaseResource in state.selectedDatabaseResourceList"
                :key="`${databaseResource.databaseId}`"
                class="flex flex-row justify-start items-center"
              >
                <DatabaseResourceView :database-resource="databaseResource" />
              </div>
            </div>
          </div>
        </NRadioGroup>
      </div>
      <div
        v-else
        class="flex flex-row justify-start items-start flex-wrap gap-2 gap-x-4"
      >
        <span v-if="state.selectedDatabaseResourceList.length === 0">{{
          $t("issue.grant-request.all-databases")
        }}</span>
        <div
          v-for="databaseResource in state.selectedDatabaseResourceList"
          v-else
          :key="`${databaseResource.databaseId}`"
          class="flex flex-row justify-start items-center"
        >
          <DatabaseResourceView :database-resource="databaseResource" />
        </div>
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

  <BBModal
    v-if="state.showSelectDatabasePanel && state.projectId && state.databaseId"
    class="relative overflow-hidden"
    :title="$t('issue.grant-request.manually-select')"
    @close="state.showSelectDatabasePanel = false"
  >
    <SelectDatabaseResourceForm
      :project-id="state.projectId"
      :database-id="state.databaseId"
      :selected-database-resource-list="state.selectedDatabaseResourceList"
      @close="state.showSelectDatabasePanel = false"
      @update="handleSelectedDatabaseResourceChanged"
    />
  </BBModal>
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
import { memberListInProjectV1 } from "@/utils";
import {
  convertUserToPrincipal,
  useDatabaseStore,
  useProjectV1Store,
} from "@/store";
import MonacoEditor from "@/components/MonacoEditor";
import RequiredStar from "@/components/RequiredStar.vue";
import { DatabaseResource } from "./SelectDatabaseResourceForm/common";
import SelectDatabaseResourceForm from "./SelectDatabaseResourceForm/index.vue";
import { converFromCEL } from "@/utils/issue/cel";

interface LocalState {
  showSelectDatabasePanel: boolean;
  // For creating
  projectId?: string;
  environmentId?: string;
  databaseId?: DatabaseId;
  allTables: boolean;
  selectedDatabaseResourceList: DatabaseResource[];
  maxRowCount: number;
  exportFormat: "CSV" | "JSON";
  statement: string;
}

const { create, issue } = useIssueLogic();
const databaseStore = useDatabaseStore();
const state = reactive<LocalState>({
  showSelectDatabasePanel: false,
  allTables: true,
  selectedDatabaseResourceList: [],
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

const handleManuallySelectClick = () => {
  if (state.selectedDatabaseResourceList.length === 0) {
    state.showSelectDatabasePanel = true;
  }
};

const handleSelectedDatabaseResourceChanged = (
  databaseResourceList: DatabaseResource[]
) => {
  state.selectedDatabaseResourceList = databaseResourceList;
  state.showSelectDatabasePanel = false;
  state.allTables = false;

  if (create.value) {
    (
      (issue.value as IssueCreate).createContext as GrantRequestContext
    ).databaseResources = databaseResourceList;
  }
};

const handleStatementChange = (value: string) => {
  state.statement = value;
};

watch(
  () => [
    state.databaseId,
    state.allTables,
    state.selectedDatabaseResourceList,
    state.maxRowCount,
    state.exportFormat,
    state.statement,
  ],
  () => {
    if (create.value) {
      const context = (issue.value as IssueCreate)
        .createContext as GrantRequestContext;
      if (state.databaseId) {
        if (state.allTables) {
          context.databaseResources = [
            {
              databaseId: state.databaseId,
            },
          ];
        } else {
          context.databaseResources = state.selectedDatabaseResourceList || [];
        }
      } else {
        context.databaseResources = [];
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

      const conditionExpression = await converFromCEL(
        payload.condition.expression
      );
      if (
        conditionExpression.databaseResources !== undefined &&
        conditionExpression.databaseResources.length > 0
      ) {
        const isAllTables = conditionExpression.databaseResources.find(
          (resource) => resource.schema === undefined
        );
        if (!isAllTables) {
          state.selectedDatabaseResourceList =
            conditionExpression.databaseResources;
        }
        const resource = head(conditionExpression.databaseResources);
        if (resource) {
          state.databaseId = resource.databaseId;
        }
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
