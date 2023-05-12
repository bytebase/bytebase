<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start space-y-4 my-8 gap-4"
  >
    <div v-if="create" class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center">
        {{ $t("issue.grant-request.select-project") }}
        <RequiredStar />
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :selected-id="projectId"
        @select-project-id="handleSourceProjectSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-start">
      <span class="flex w-40 items-center shrink-0">
        {{ $t("issue.grant-request.database") }}
        <RequiredStar />
      </span>
      <div v-if="create" class="flex flex-row justify-start items-center">
        <EnvironmentSelect
          class="!w-60 mr-4 shrink-0"
          name="environment"
          :select-default="false"
          :selected-id="state.environmentId"
          @select-environment-id="(id) => (state.environmentId = id)"
        />
        <DatabaseSelect
          class="!w-128"
          :selected-id="(state.databaseId as DatabaseId)"
          :mode="'ALL'"
          :environment-id="state.environmentId"
          :project-id="state.projectId"
          :sync-status="'OK'"
          :customize-item="true"
          @select-database-id="(id) => (state.databaseId = id)"
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
      <span class="flex w-40 items-center">
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
      <span class="flex w-40 items-center">
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
      <span class="flex w-40 items-center shrink-0">SQL<RequiredStar /></span>
      <div class="whitespace-pre-wrap w-full overflow-hidden border">
        <MonacoEditor
          ref="editorRef"
          class="w-full h-[360px]"
          data-label="bb-issue-sql-editor"
          :value="state.statement"
          :auto-focus="false"
          :language="'sql'"
          :dialect="dialect"
          :readonly="!create"
          @change="handleStatementChange"
          @ready="handleMonacoEditorReady"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useIssueLogic } from "../logic";
import {
  Database,
  DatabaseId,
  EnvironmentId,
  GrantRequestContext,
  GrantRequestPayload,
  Issue,
  IssueCreate,
  ProjectId,
  SQLDialect,
  UNKNOWN_ID,
  dialectOfEngine,
} from "@/types";
import { getProjectMemberList } from "@/utils";
import { useDBSchemaStore, useDatabaseStore } from "@/store";
import MonacoEditor from "@/components/MonacoEditor";
import { TableMetadata } from "@/types/proto/store/database";
import { useInstanceV1Store } from "@/store/modules/v1/instance";
import {
  environmentNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import RequiredStar from "@/components/RequiredStar.vue";

interface LocalState {
  // For creating
  projectId?: ProjectId;
  environmentId?: EnvironmentId;
  databaseId?: DatabaseId;
  maxRowCount: number;
  exportFormat: "CSV" | "JSON";
  statement: string;
}

const { create, issue } = useIssueLogic();
const instanceV1Store = useInstanceV1Store();
const databaseStore = useDatabaseStore();
const dbSchemaStore = useDBSchemaStore();
const state = reactive<LocalState>({
  maxRowCount: 1000,
  exportFormat: "CSV",
  statement: "",
});
const editorRef = ref<InstanceType<typeof MonacoEditor>>();

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
    const projectId = (issue.value as IssueCreate).projectId;
    if (projectId && projectId !== UNKNOWN_ID) {
      state.projectId = projectId;
    }
  }
});

const handleSourceProjectSelect = async (projectId: ProjectId) => {
  if (!create.value) {
    return;
  }

  state.projectId = projectId;
  (issue.value as IssueCreate).projectId = projectId;
  // update issue assignee
  const projectOwner = head(
    (await getProjectMemberList(projectId)).filter(
      (member) => !member.roleList.includes("OWNER")
    )
  );
  if (projectOwner) {
    (issue.value as IssueCreate).assigneeId = projectOwner.principal.id;
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
      }
      context.maxRowCount = state.maxRowCount;
      context.exportFormat = state.exportFormat;
      context.statement = state.statement;
    }
  }
);

// Handle and update monaco editor auto completion context.
const useDatabaseAndTableList = () => {
  const databaseList = computed(() => {
    if (selectedDatabase.value) return [selectedDatabase.value];
    return [];
  });

  watch(
    databaseList,
    (list) => {
      list.forEach((db) => {
        if (db.id && db.id !== UNKNOWN_ID) {
          dbSchemaStore.getOrFetchTableListByDatabaseId(db.id);
        }
      });
    },
    { immediate: true }
  );

  const tableList = computed(() => {
    return databaseList.value
      .map((item) => dbSchemaStore.getTableListByDatabaseId(item.id))
      .flat();
  });

  return { databaseList, tableList };
};

const { databaseList } = useDatabaseAndTableList();

const handleUpdateEditorAutoCompletionContext = async () => {
  const databaseMap: Map<Database, TableMetadata[]> = new Map();
  for (const database of databaseList.value) {
    const tableList = dbSchemaStore.getTableListByDatabaseId(database.id);
    databaseMap.set(database, tableList);
  }
  editorRef.value?.setEditorAutoCompletionContext(databaseMap);
};

const handleMonacoEditorReady = () => {
  handleUpdateEditorAutoCompletionContext();
};

watch(
  create,
  async () => {
    if (!create.value) {
      const payload = ((issue.value as Issue).payload as any)
        .grantRequest as GrantRequestPayload;
      if (payload.role !== "roles/EXPORTER") {
        throw "Only support EXPORTER role";
      }
      const expressionList = payload.condition.expression.split(" && ");
      for (const expression of expressionList) {
        const fields = expression.split(" ");
        if (fields[0] === "request.statement") {
          state.statement = atob(JSON.parse(fields[2]));
        } else if (fields[0] === "resource.database") {
          const databaseIdList = [];
          for (const url of JSON.parse(fields[2])) {
            const value = url.split("/");
            const instanceName = value[1] || "";
            const databaseName = value[3] || "";
            const instance = await instanceV1Store.getOrFetchInstanceByName(
              environmentNamePrefix + "-/" + instanceNamePrefix + instanceName
            );
            const databaseList =
              await databaseStore.getOrFetchDatabaseListByInstanceId(
                instance.uid
              );
            const database = databaseList.find(
              (db) => db.name === databaseName
            );
            if (database) {
              databaseIdList.push(database.id);
            }
          }
          state.databaseId = head(databaseIdList);
        } else if (fields[0] === "request.row_limit") {
          state.maxRowCount = Number(fields[2]);
        } else if (fields[0] === "request.export_format") {
          state.exportFormat = JSON.parse(fields[2]) as "CSV" | "JSON";
        }
      }
    }
  },
  {
    immediate: true,
  }
);
</script>
