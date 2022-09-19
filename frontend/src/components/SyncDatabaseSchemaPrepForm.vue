<template>
  <div
    class="space-y-4 overflow-x-hidden w-144 transition-all"
    :class="state.currentStep === 1 ? 'w-176' : ''"
  >
    <!-- Base and target database selectors -->
    <div
      v-show="state.currentStep === 0"
      class="w-full flex flex-col justify-start items-start"
    >
      <p class="mb-2">{{ $t("database.sync-schema.description") }}</p>
      <div class="w-full">
        <p class="mt-4 mb-2 text-gray-600">
          {{ $t("database.sync-schema.base-database") }}
        </p>
        <div class="w-full flex flex-row justify-start items-center">
          <EnvironmentSelect
            class="!w-48 mr-2 shrink-0"
            name="environment"
            :selected-id="state.baseSchemaInfo.environmentId"
            :select-default="false"
            @select-environment-id="
              (environmentId) =>
                (state.baseSchemaInfo.environmentId = environmentId)
            "
          />
          <DatabaseSelect
            class="!w-48 mr-2 shrink-0"
            :selected-id="(state.baseSchemaInfo.databaseId as DatabaseId)"
            :mode="'ENVIRONMENT'"
            :environment-id="state.baseSchemaInfo.environmentId"
            :project-id="props.projectId"
            :show-engine-icon="true"
            @select-database-id="
              (databaseId: DatabaseId) => {
                state.baseSchemaInfo.databaseId = databaseId;
              }
            "
          />
          <BBSelect
            class=""
            :selected-item="state.baseSchemaInfo.migrationHistory"
            :item-list="
              databaseMigrationHistoryList(state.baseSchemaInfo.databaseId as DatabaseId)
            "
            :placeholder="$t('migration-history.select')"
            :show-prefix-item="true"
            @select-item="(migrationHistory: MigrationHistory) => state.baseSchemaInfo.migrationHistory = migrationHistory"
          >
            <template #menuItem="{ item: migrationHistory }">
              <div class="flex items-center">
                {{ migrationHistory.version }}
              </div>
            </template>
          </BBSelect>
        </div>
      </div>
      <div class="w-full">
        <p class="mt-4 mb-2 text-gray-600">
          {{ $t("database.sync-schema.target-database") }}
        </p>
        <div class="w-full flex flex-row justify-start items-center">
          <EnvironmentSelect
            class="!w-48 mr-2 shrink-0"
            name="environment"
            :selected-id="state.targetSchemaInfo.environmentId"
            :select-default="false"
            @select-environment-id="
              (environmentId) =>
                (state.targetSchemaInfo.environmentId = environmentId)
            "
          />
          <DatabaseSelect
            class="!grow !w-full"
            :selected-id="(state.targetSchemaInfo.databaseId as DatabaseId)"
            :mode="'ENVIRONMENT'"
            :environment-id="state.targetSchemaInfo.environmentId"
            :project-id="props.projectId"
            :engine-type="state.engineType"
            :show-engine-icon="true"
            @select-database-id="
              (databaseId: DatabaseId) => {
                state.targetSchemaInfo.databaseId = databaseId;
              }
            "
          />
        </div>
      </div>
    </div>

    <!-- Schema diff statement editor container -->
    <div
      v-show="state.currentStep === 1"
      class="w-full flex flex-col justify-start items-start"
    >
      <div
        class="w-full flex flex-row justify-between items-center mb-2 leading-8"
      >
        <div class="flex flex-row justify-start items-center">
          <span>{{ $t("database.sync-schema.synchronize-statements") }}</span>
          <button
            type="button"
            class="btn-icon ml-2"
            @click.prevent="copyStatement"
          >
            <heroicons-solid:clipboard class="h-5 w-5" />
          </button>
        </div>
        <div>
          <button
            v-if="state.recommandStatement !== state.editStatement"
            type="button"
            class="btn-icon border px-3 pl-4 leading-7 hover:border-gray-500"
            @click.prevent="handleGenerateButtonClick"
          >
            {{ $t("common.restore") }}
            <heroicons-solid:refresh class="h-4 w-4 ml-1" />
          </button>
        </div>
      </div>
      <p
        class="text-sm border px-2 mb-3 -mt-1 rounded leading-6 text-yellow-600 border-yellow-600 bg-yellow-50"
      >
        {{ $t("database.sync-schema.check-generated-ddl-statement") }}
      </p>
      <div class="whitespace-pre-wrap w-full overflow-hidden border">
        <MonacoEditor
          ref="editorRef"
          class="w-full h-auto max-h-[300px]"
          data-label="bb-issue-sql-editor"
          :value="state.editStatement"
          :auto-focus="false"
          :dialect="(state.engineType as SQLDialect)"
          @change="onStatementChange"
          @ready="updateEditorHeight"
        />
      </div>
      <div class="w-full flex flex-row justify-start items-center mt-4 mb-2">
        <span>{{ $t("database.sync-schema.schema-comparison-result") }}</span>
      </div>
      <code-diff
        class="w-full"
        :old-string="state.baseSchemaInfo.migrationHistory?.schema ?? ''"
        :new-string="targetDatabaseLatestDoneMigrationHistory?.schema ?? ''"
        output-format="side-by-side"
        data-label="bb-migration-history-code-diff-block"
      />
    </div>

    <!-- Buttons group -->
    <div class="pt-2 flex items-center justify-between">
      <span></span>
      <div class="flex items-center justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="handleCancelButtonClick"
        >
          {{
            state.currentStep === 0 ? $t("common.cancel") : $t("common.back")
          }}
        </button>
        <button
          :disabled="!allowNext"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          @click="handleNextButtonClick"
        >
          {{
            state.currentStep === 0 ? $t("common.next") : $t("common.confirm")
          }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import axios from "axios";
import { computed, reactive, ref, watch } from "vue";
import { useEventListener } from "@vueuse/core";
import { useRouter } from "vue-router";
import { CodeDiff } from "v-code-diff";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import {
  DatabaseId,
  EngineType,
  EnvironmentId,
  IssueCreate,
  MigrationHistory,
  MigrationSchemaStatus,
  ProjectId,
  SQLDialect,
  SYSTEM_BOT_ID,
  UNKNOWN_ID,
  MigrationContext,
} from "@/types";
import {
  pushNotification,
  useDatabaseStore,
  useInstanceStore,
  useIssueStore,
} from "@/store";
import { isNullOrUndefined } from "@/plugins/demo/utils";
import EnvironmentSelect from "./EnvironmentSelect.vue";
import DatabaseSelect from "./DatabaseSelect.vue";
import MonacoEditor from "./MonacoEditor/MonacoEditor.vue";
import { issueSlug } from "@/utils";

type LocalState = {
  currentStep: 0 | 1;
  baseSchemaInfo: {
    environmentId?: EnvironmentId;
    databaseId?: DatabaseId;
    migrationHistory?: MigrationHistory;
    migrationSchemaStatus?: MigrationSchemaStatus;
  };
  targetSchemaInfo: {
    environmentId?: EnvironmentId;
    databaseId?: DatabaseId;
  };
  engineType?: EngineType;
  recommandStatement: string;
  editStatement: string;
};

const props = withDefaults(
  defineProps<{
    projectId: ProjectId;
  }>(),
  {
    projectId: UNKNOWN_ID,
  }
);
const emit = defineEmits(["dismiss"]);

const router = useRouter();

const editorRef = ref<InstanceType<typeof MonacoEditor>>();
const instanceStore = useInstanceStore();
const databaseStore = useDatabaseStore();

useEventListener(window, "keydown", (e) => {
  if (e.code === "Escape") {
    emit("dismiss");
  }
});

const state = reactive<LocalState>({
  currentStep: 0,
  baseSchemaInfo: {},
  targetSchemaInfo: {},
  recommandStatement: "",
  editStatement: "",
});

const targetDatabaseLatestDoneMigrationHistory = computed(() => {
  const database = databaseStore.getDatabaseById(
    state.targetSchemaInfo.databaseId as DatabaseId
  );
  const list = instanceStore
    .getMigrationHistoryListByInstanceIdAndDatabaseName(
      database.instance.id,
      database.name
    )
    .filter((history) => history.status === "DONE");

  return list[0];
});

const isValidId = (id: any) => {
  if (isNullOrUndefined(id) || id === UNKNOWN_ID) {
    return false;
  }
  return true;
};

const allowNext = computed(() => {
  if (state.currentStep === 0) {
    return (
      isValidId(state.baseSchemaInfo.environmentId) &&
      isValidId(state.baseSchemaInfo.databaseId) &&
      !isNullOrUndefined(state.baseSchemaInfo.migrationHistory) &&
      isValidId(state.targetSchemaInfo.environmentId) &&
      isValidId(state.targetSchemaInfo.databaseId)
    );
  } else {
    return state.editStatement !== "";
  }
});

const databaseMigrationHistoryList = (databaseId: DatabaseId) => {
  const database = databaseStore.getDatabaseById(databaseId);
  const list = instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
    database.instance.id,
    database.name
  );

  return list;
};

const handleCancelButtonClick = () => {
  if (state.currentStep === 0) {
    emit("dismiss");
  } else {
    state.currentStep = 0;
    state.recommandStatement = "";
    state.editStatement = "";
  }
};

const handleNextButtonClick = async () => {
  if (state.currentStep === 0 && allowNext.value) {
    state.currentStep = 1;
    const schema = await getSchemaDiff(
      state.engineType as EngineType,
      state.baseSchemaInfo.migrationHistory?.schema ?? "",
      targetDatabaseLatestDoneMigrationHistory.value.schema ?? ""
    );
    state.recommandStatement = schema;
    state.editStatement = schema;
  } else if (state.currentStep === 1) {
    if (state.editStatement === "") {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Statements shouldn't be empty",
      });
      return;
    }

    const sourceDatabase = databaseStore.getDatabaseById(
      state.baseSchemaInfo.databaseId as DatabaseId
    );
    const targetDatabase = databaseStore.getDatabaseById(
      state.targetSchemaInfo.databaseId as DatabaseId
    );

    const migrationContext: MigrationContext = {
      migrationType: "MIGRATE",
      detailList: [
        {
          databaseId: targetDatabase.id,
          databaseName: targetDatabase.name,
          statement: state.editStatement,
          earliestAllowedTs: 0,
        },
      ],
    };

    const databaseName = targetDatabase.name;
    const newIssue: IssueCreate = {
      name: `[${databaseName}] Sync Schema from ${
        sourceDatabase.name
      } @ ${dayjs().format("MM-DD HH:mm")}`,
      type: "bb.issue.database.schema.update",
      description: "",
      assigneeId: SYSTEM_BOT_ID,
      projectId: targetDatabase.projectId,
      pipeline: {
        stageList: [],
        name: "",
      },
      createContext: migrationContext,
      payload: {},
    };

    useIssueStore()
      .createIssue(newIssue)
      .then((createdIssue) => {
        router.push(`/issue/${issueSlug(createdIssue.name, createdIssue.id)}`);
      });
  }
};

const prepareMigrationHistoryList = async (databaseId: DatabaseId) => {
  const database = databaseStore.getDatabaseById(databaseId);
  if (database && database.instance.id) {
    const migration = await instanceStore.checkMigrationSetup(
      database.instance.id
    );
    state.baseSchemaInfo.migrationSchemaStatus = migration.status;
    if (state.baseSchemaInfo.migrationSchemaStatus == "OK") {
      await instanceStore.fetchMigrationHistory({
        instanceId: database.instance.id,
        databaseName: database.name,
      });
    }
  }
};

const getSchemaDiff = async (
  engineType: EngineType,
  sourceSchema: string,
  targetSchema: string
) => {
  const { data } = await axios.post("/v1/sql/schema/diff", {
    engineType,
    sourceSchema,
    targetSchema,
  });
  return data;
};

const onStatementChange = (value: string) => {
  state.editStatement = value;
  updateEditorHeight();
};

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  const actualHeight = contentHeight;
  editorRef.value?.setEditorContentHeight(actualHeight);
};

const copyStatement = () => {
  toClipboard(state.editStatement).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Statement copied to clipboard.`,
    });
  });
};

const handleGenerateButtonClick = async () => {
  const schema = await getSchemaDiff(
    state.engineType as EngineType,
    state.baseSchemaInfo.migrationHistory?.schema ?? "",
    targetDatabaseLatestDoneMigrationHistory.value.schema ?? ""
  );
  state.recommandStatement = schema;
  state.editStatement = schema;
};

watch(
  () => [state.baseSchemaInfo.environmentId, state.baseSchemaInfo.databaseId],
  () => {
    const databaseId = state.baseSchemaInfo.databaseId;

    if (isValidId(databaseId)) {
      prepareMigrationHistoryList(databaseId as DatabaseId);
      const database = databaseStore.getDatabaseById(
        state.baseSchemaInfo.databaseId as DatabaseId
      );
      state.engineType = database.instance.engine;
    } else {
      state.engineType = undefined;
    }
    state.baseSchemaInfo.migrationHistory = undefined;
  }
);

watch(
  () => [
    state.targetSchemaInfo.environmentId,
    state.targetSchemaInfo.databaseId,
  ],
  () => {
    const databaseId = state.targetSchemaInfo.databaseId;
    if (isValidId(databaseId)) {
      prepareMigrationHistoryList(databaseId as DatabaseId);
    }
  }
);
</script>
