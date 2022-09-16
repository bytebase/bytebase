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
      <p class="mb-2">
        Synchronize schema from the base database to the target database with
        the selected migration version.
      </p>
      <div class="w-full">
        <p class="mt-4 mb-2 text-gray-600">Base database</p>
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
        <p class="mt-4 mb-2 text-gray-600">Target database</p>
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
      <div class="w-full flex flex-row justify-between items-center mb-2">
        <div class="flex flex-row justify-start items-center">
          <span>The statements of synchronize</span>
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
            type="button"
            class="btn-icon border px-3 leading-7 hover:border-gray-500"
            @click.prevent="handleEditButtonClick"
          >
            <template v-if="!state.isEditting">
              {{ $t("common.edit") }}
              <heroicons-solid:pencil class="h-5 w-5 ml-1 -mr-1" />
            </template>
            <template v-else>{{ $t("common.cancel") }}</template>
          </button>
        </div>
      </div>
      <p
        class="text-sm border px-2 mb-3 -mt-1 rounded leading-6 text-yellow-600 border-yellow-600 bg-yellow-50"
      >
        Please check the following generated DDL statement.
      </p>
      <div
        class="whitespace-pre-wrap w-full overflow-hidden border"
        :class="state.isEditting ? 'border-blue-600 border-2' : ''"
      >
        <MonacoEditor
          ref="editorRef"
          class="w-full h-auto max-h-[360px]"
          data-label="bb-issue-sql-editor"
          :value="state.recommandSchema"
          :readonly="!state.isEditting"
          :auto-focus="false"
          :dialect="(state.engineType as SQLDialect)"
          @change="onStatementChange"
          @ready="handleMonacoEditorReady"
        />
      </div>
      <div class="w-full flex flex-row justify-start items-center mt-4 mb-2">
        <span>The schema comparison result</span>
      </div>
      <code-diff
        class="w-full"
        :old-string="targetDatabaseLatestDoneMigrationHistory?.schema ?? ''"
        :new-string="state.baseSchemaInfo.migrationHistory?.schema ?? ''"
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
import axios from "axios";
import { computed, reactive, ref, watch } from "vue";
import { useEventListener } from "@vueuse/core";
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
  UNKNOWN_ID,
  UpdateSchemaContext,
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
import dayjs from "dayjs";
import { useRouter } from "vue-router";
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
  recommandSchema: string;
  isEditting: boolean;
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
  recommandSchema: "",
  isEditting: false,
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

const databaseMigrationHistoryList = (databaseId: DatabaseId) => {
  const database = databaseStore.getDatabaseById(databaseId);
  const list = instanceStore.getMigrationHistoryListByInstanceIdAndDatabaseName(
    database.instance.id,
    database.name
  );

  return list;
};

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
  }

  return true;
});

const handleCancelButtonClick = () => {
  if (state.currentStep === 0) {
    emit("dismiss");
  } else {
    state.currentStep = 0;
    state.isEditting = false;
    state.recommandSchema = "";
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
    state.recommandSchema = schema;
  } else if (state.currentStep === 1) {
    const targetDatabase = databaseStore.getDatabaseById(
      state.targetSchemaInfo.databaseId as DatabaseId
    );
    const updateSchemaContext: UpdateSchemaContext = {
      migrationType: "MIGRATE",
      updateSchemaDetailList: [
        {
          databaseId: targetDatabase.id,
          databaseName: targetDatabase.name,
          statement: state.recommandSchema,
          earliestAllowedTs: Date.now() / 1000,
        },
      ],
    };

    const databaseName = isDbNameTemplateMode.value
      ? generatedDatabaseName.value
      : state.databaseName;
    const instanceId = state.instanceId as InstanceId;
    let owner = "";
    if (requireDatabaseOwnerName.value && state.instanceUserId) {
      const instanceUser = await useInstanceStore().fetchInstanceUser(
        instanceId,
        state.instanceUserId
      );
      owner = instanceUser.name;
    }

    if (isTenantProject.value) {
      if (!hasFeature("bb.feature.multi-tenancy")) {
        state.showFeatureModal = true;
        return;
      }
    }
    // Do not submit non-selected optional labels
    const labelList = state.labelList.filter((label) => !!label.value);

    // Otherwise we create a simple database.create issue.
    const newIssue: IssueCreate = {
      name: `Create database '${databaseName}'`,
      type: "bb.issue.database.create",
      description: "",
      assigneeId: state.assigneeId!,
      projectId: state.projectId!,
      pipeline: {
        stageList: [],
        name: "",
      },
      createContext: updateSchemaContext,
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

const handleMonacoEditorReady = () => {
  updateEditorHeight();
};

const onStatementChange = (value: string) => {
  state.recommandSchema = value;
  updateEditorHeight();
};

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  const actualHeight = contentHeight;
  editorRef.value?.setEditorContentHeight(actualHeight);
};

const copyStatement = () => {
  toClipboard(state.recommandSchema).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Statement copied to clipboard.`,
    });
  });
};

const handleEditButtonClick = () => {
  state.isEditting = !state.isEditting;
};

watch(
  () => [state.baseSchemaInfo.environmentId, state.baseSchemaInfo.databaseId],
  () => {
    const databaseId = state.baseSchemaInfo.databaseId;
    if (isValidId(databaseId)) {
      prepareMigrationHistoryList(databaseId as DatabaseId);
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

watch(
  () => [state.baseSchemaInfo.databaseId],
  () => {
    if (!isValidId(state.baseSchemaInfo.databaseId)) {
      state.engineType = undefined;
      return;
    }

    const database = databaseStore.getDatabaseById(
      state.baseSchemaInfo.databaseId as DatabaseId
    );
    state.engineType = database.instance.engine;
  }
);
</script>
