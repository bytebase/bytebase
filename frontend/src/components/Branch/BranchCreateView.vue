<template>
  <div class="space-y-3 w-full overflow-x-auto px-4 pb-8 pt-4">
    <div
      v-if="showProjectSelector"
      class="w-full flex flex-row justify-start items-center"
    >
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.project") }}
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :project="state.projectId"
        @update:project="handleProjectSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-center mt-1">
      <span class="flex w-40 items-center text-sm">{{
        $t("database.branch-name")
      }}</span>
      <NInput
        v-model:value="state.schemaDesignTitle"
        required
        type="text"
        class="!w-60 text-sm"
        :placeholder="'feature/add-billing'"
      />
      <span class="ml-8 mr-4 flex items-center text-sm">{{
        $t("schema-designer.parent-branch")
      }}</span>
      <BranchSelector
        class="!w-60"
        clearable
        :branch="state.parentBranchName"
        :project="state.projectId"
        @update:branch="(branch) => (state.parentBranchName = branch ?? '')"
      />
    </div>
    <NDivider />
    <div class="w-full flex flex-row justify-start items-center mt-1">
      <span class="flex w-full items-center text-sm font-medium">{{
        state.parentBranchName
          ? $t("schema-designer.baseline-version-from-parent")
          : $t("schema-designer.baseline-version")
      }}</span>
    </div>
    <BaselineSchemaSelector
      :project-id="state.projectId"
      :database-id="state.baselineSchema.databaseId"
      :change-history="state.baselineSchema.changeHistory"
      :readonly="disallowToChangeBaseline"
      @update="handleBaselineSchemaChange"
    />
    <div class="!mt-6 w-full h-[32rem]">
      <SchemaEditorV1
        :key="refreshId"
        :project="project"
        :resource-type="'branch'"
        :branches="[state.schemaDesign]"
        :readonly="true"
      />
    </div>
    <div class="w-full flex items-center justify-between mt-4">
      <div></div>

      <div class="flex items-center justify-end gap-x-3">
        <NButton
          type="primary"
          :disabled="!allowConfirm"
          :loading="state.isCreating"
          @click.prevent="handleConfirm"
        >
          {{ confirmText }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, uniqueId } from "lodash-es";
import { NButton, NDivider, NInput } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import SchemaEditorV1 from "@/components/SchemaEditorV1/index.vue";
import { mergeSchemaEditToMetadata } from "@/components/SchemaEditorV1/utils";
import { ProjectSelect } from "@/components/v2";
import {
  pushNotification,
  useChangeHistoryStore,
  useDatabaseV1Store,
  useProjectV1Store,
  useSchemaEditorV1Store,
  useSheetV1Store,
} from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import {
  databaseNamePrefix,
  getProjectAndSchemaDesignSheetId,
} from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import {
  ChangeHistory,
  ChangeHistoryView,
  DatabaseMetadata,
} from "@/types/proto/v1/database_service";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";
import {
  Sheet_Source,
  Sheet_Type,
  Sheet_Visibility,
} from "@/types/proto/v1/sheet_service";
import { extractChangeHistoryUID, projectV1Slug } from "@/utils";
import BaselineSchemaSelector from "./BaselineSchemaSelector.vue";
import { validateBranchName } from "./utils";

interface BaselineSchema {
  // The uid of database.
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

interface LocalState {
  projectId?: string;
  schemaDesignTitle: string;
  baselineSchema: BaselineSchema;
  schemaDesign: SchemaDesign;
  parentBranchName?: string;
  isCreating: boolean;
}

const props = defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
});

const { t } = useI18n();
const router = useRouter();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const schemaDesignStore = useSchemaDesignStore();
const changeHistoryStore = useChangeHistoryStore();
const sheetStore = useSheetV1Store();
const state = reactive<LocalState>({
  schemaDesignTitle: "",
  projectId: props.projectId,
  baselineSchema: {},
  schemaDesign: SchemaDesign.fromPartial({
    type: SchemaDesign_Type.MAIN_BRANCH,
  }),
  isCreating: false,
});
const showProjectSelector = ref<boolean>(true);
const refreshId = ref<string>("");

const project = computed(() => {
  const project = projectStore.getProjectByUID(state.projectId || "");
  return project;
});

const disallowToChangeBaseline = computed(() => {
  return !!state.parentBranchName;
});

onMounted(async () => {
  if (props.projectId) {
    state.projectId = props.projectId;
    // When we are creating a branch from a project page, we don't show the project selector.
    showProjectSelector.value = false;
  }
});

watch(
  () => [
    state.schemaDesign.baselineDatabase,
    state.schemaDesign.baselineSchema,
  ],
  () => {
    refreshId.value = uniqueId();
  }
);

watch(
  () => state.parentBranchName,
  async () => {
    if (!state.parentBranchName) {
      state.baselineSchema = {};
      state.schemaDesign = SchemaDesign.fromPartial({
        type: SchemaDesign_Type.MAIN_BRANCH,
      });
      return;
    }

    const branch = await schemaDesignStore.fetchSchemaDesignByName(
      state.parentBranchName,
      false /* !useCache */
    );
    const database = await databaseStore.getOrFetchDatabaseByName(
      branch.baselineDatabase
    );
    state.projectId = database.projectEntity.uid;
    state.baselineSchema.databaseId = database.uid;
    if (
      branch.baselineChangeHistoryId &&
      branch.baselineChangeHistoryId !== String(UNKNOWN_ID)
    ) {
      const changeHistoryName = `${database.name}/changeHistories/${branch.baselineChangeHistoryId}`;
      state.baselineSchema.changeHistory =
        await changeHistoryStore.getOrFetchChangeHistoryByName(
          changeHistoryName
        );
    } else {
      state.baselineSchema.changeHistory = undefined;
    }
    state.schemaDesign = branch;
    refreshId.value = uniqueId();
  }
);

const prepareFullChangeHistorySchema = async (changeHistory: ChangeHistory) => {
  // While a database has no change histories, the state.baselineSchema.changeHistory
  // is a mock ChangeHistory entity with uid = -1
  // so we should use the changeHistory it self and need not to fetch the
  // full view of a real ChangeHistory entity.
  const uid = extractChangeHistoryUID(changeHistory.name);
  if (uid === String(UNKNOWN_ID)) {
    return changeHistory.schema;
  }

  const changeHistoryWithFullView =
    await useChangeHistoryStore().fetchChangeHistory({
      name: changeHistory.name,
      view: ChangeHistoryView.CHANGE_HISTORY_VIEW_FULL,
    });
  return changeHistoryWithFullView?.schema ?? changeHistory.schema;
};

const prepareSchemaDesign = async () => {
  const changeHistory = state.baselineSchema.changeHistory;
  if (changeHistory && state.baselineSchema.databaseId) {
    const database = databaseStore.getDatabaseByUID(
      state.baselineSchema.databaseId
    );
    const sheet = await sheetStore.getOrFetchSheetByName(
      changeHistory.statementSheet
    );
    const fullSchema = await prepareFullChangeHistorySchema(changeHistory);
    const baselineMetadata = await schemaDesignStore.parseSchemaString(
      fullSchema,
      database.instanceEntity.engine
    );
    baselineMetadata.schemaConfigs =
      sheet?.payload?.databaseConfig?.schemaConfigs ?? [];
    return SchemaDesign.fromPartial({
      engine: database.instanceEntity.engine,
      baselineSchema: fullSchema,
      baselineSchemaMetadata: baselineMetadata,
      schema: fullSchema,
      schemaMetadata: baselineMetadata,
      type: SchemaDesign_Type.MAIN_BRANCH,
    });
  }
  return SchemaDesign.fromPartial({
    type: SchemaDesign_Type.MAIN_BRANCH,
  });
};

const allowConfirm = computed(() => {
  return (
    state.projectId &&
    state.schemaDesignTitle &&
    state.baselineSchema.databaseId &&
    !state.isCreating
  );
});

const confirmText = computed(() => {
  return t("common.create");
});

const handleProjectSelect = async (projectId?: string) => {
  state.projectId = projectId;
  state.parentBranchName = "";
};

const handleBaselineSchemaChange = async (baselineSchema: BaselineSchema) => {
  if (state.parentBranchName) {
    return;
  }

  state.baselineSchema = baselineSchema;
  if (
    baselineSchema.databaseId &&
    baselineSchema.databaseId !== String(UNKNOWN_ID)
  ) {
    const database = databaseStore.getDatabaseByUID(baselineSchema.databaseId);
    state.projectId = database.projectEntity.uid;
  }
  state.schemaDesign = await prepareSchemaDesign();
};

const handleConfirm = async () => {
  if (!state.schemaDesign) {
    return;
  }

  if (!validateBranchName(state.schemaDesignTitle)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Branch name valid characters: /^[a-zA-Z][a-zA-Z0-9-_/]+$/",
    });
    return;
  }

  const database = useDatabaseV1Store().getDatabaseByUID(
    state.baselineSchema.databaseId || ""
  );
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    state.schemaDesign.name
  );
  if (!branchSchema) {
    return;
  }

  state.isCreating = true;
  const metadata = mergeSchemaEditToMetadata(
    branchSchema.schemaList,
    cloneDeep(
      state.schemaDesign.baselineSchemaMetadata ||
        DatabaseMetadata.fromPartial({})
    )
  );
  const baselineDatabase = `${database.instanceEntity.name}/${databaseNamePrefix}${state.baselineSchema.databaseId}`;
  // Create a baseline sheet for the schema design.
  const baselineSheet = await sheetStore.createSheet(project.value.name, {
    title: `baseline schema of ${state.schemaDesignTitle}`,
    database: baselineDatabase,
    content: new TextEncoder().encode(state.schemaDesign.baselineSchema),
    visibility: Sheet_Visibility.VISIBILITY_PROJECT,
    source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
    type: Sheet_Type.TYPE_SQL,
  });

  let createdSchemaDesign;
  if (!state.parentBranchName) {
    createdSchemaDesign = await schemaDesignStore.createSchemaDesign(
      project.value.name,
      SchemaDesign.fromPartial({
        title: state.schemaDesignTitle,
        // Keep schema empty in frontend. Backend will generate the design schema.
        schema: "",
        schemaMetadata: metadata,
        baselineSchema: state.schemaDesign.baselineSchema,
        baselineSchemaMetadata: state.schemaDesign.baselineSchemaMetadata,
        engine: state.schemaDesign.engine,
        type: SchemaDesign_Type.MAIN_BRANCH,
        baselineDatabase: baselineDatabase,
        baselineSheetName: baselineSheet.name,
        baselineChangeHistoryId: state.baselineSchema.changeHistory?.uid,
        protection: {
          // For main branches, we don't allow force pushes by default.
          allowForcePushes: false,
        },
      })
    );
  } else {
    const parentBranch = await schemaDesignStore.fetchSchemaDesignByName(
      state.parentBranchName,
      false /* useCache */
    );
    createdSchemaDesign = await schemaDesignStore.createSchemaDesignDraft({
      ...parentBranch,
      title: state.schemaDesignTitle,
    });
  }
  state.isCreating = false;
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.created-succeed"),
  });

  // Go to branch detail page after created.
  const [_, sheetId] = getProjectAndSchemaDesignSheetId(
    createdSchemaDesign.name
  );
  router.replace({
    name: "workspace.branch.detail",
    params: {
      projectSlug: projectV1Slug(project.value),
      branchName: sheetId,
    },
  });
};
</script>
