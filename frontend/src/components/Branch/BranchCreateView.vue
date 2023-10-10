<template>
  <div class="space-y-3 w-full overflow-x-auto px-4 pb-8">
    <div class="w-full flex flex-row justify-start items-center pt-1">
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.project") }}
      </span>
      <ProjectSelect
        class="!w-60 shrink-0"
        :selected-id="state.projectId"
        @select-project-id="handleProjectSelect"
      />
    </div>
    <div class="w-full flex flex-row justify-start items-center mt-1">
      <span class="flex w-40 items-center text-sm">{{
        $t("database.branch-name")
      }}</span>
      <BBTextField
        class="w-60 text-sm"
        :value="state.schemaDesignTitle"
        :placeholder="'feature/add-billing'"
        @input="
          state.schemaDesignTitle = ($event.target as HTMLInputElement).value
        "
      />
    </div>
    <NDivider />
    <div class="w-full flex flex-row justify-start items-center mt-1">
      <span class="flex w-40 items-center text-sm font-medium">{{
        $t("schema-designer.baseline-version")
      }}</span>
    </div>
    <BaselineSchemaSelector
      :project-id="state.projectId"
      :baseline-schema="state.baselineSchema"
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
import { NButton, NDivider } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import SchemaEditorV1 from "@/components/SchemaEditorV1/index.vue";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectV1Store,
  useSchemaEditorV1Store,
  useSheetV1Store,
} from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import {
  databaseNamePrefix,
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import {
  ChangeHistory,
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
import BaselineSchemaSelector from "./BaselineSchemaSelector.vue";
import { mergeSchemaEditToMetadata, validateBranchName } from "./utils";

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
}

const props = defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
});

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const schemaDesignStore = useSchemaDesignStore();
const sheetStore = useSheetV1Store();
const state = reactive<LocalState>({
  schemaDesignTitle: "",
  projectId: props.projectId,
  baselineSchema: {},
  schemaDesign: SchemaDesign.fromPartial({
    type: SchemaDesign_Type.MAIN_BRANCH,
  }),
});
const refreshId = ref<string>("");

onMounted(async () => {
  const projectName = route.params.projectName;
  if (projectName !== "-") {
    const project = await projectStore.getOrFetchProjectByName(
      `${projectNamePrefix}${projectName}`
    );
    state.projectId = project.uid;
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

const prepareSchemaDesign = async () => {
  const changeHistory = state.baselineSchema.changeHistory;
  if (changeHistory && state.baselineSchema.databaseId) {
    const database = databaseStore.getDatabaseByUID(
      state.baselineSchema.databaseId
    );
    const baselineMetadata = await schemaDesignStore.parseSchemaString(
      changeHistory.schema,
      database.instanceEntity.engine
    );
    return SchemaDesign.fromPartial({
      engine: database.instanceEntity.engine,
      baselineSchema: changeHistory.schema,
      baselineSchemaMetadata: baselineMetadata,
      schema: changeHistory.schema,
      schemaMetadata: baselineMetadata,
    });
  }
  return SchemaDesign.fromPartial({});
};

const allowConfirm = computed(() => {
  return (
    state.projectId &&
    state.schemaDesignTitle &&
    state.baselineSchema.databaseId
  );
});

const confirmText = computed(() => {
  return t("common.create");
});

const handleProjectSelect = async (projectId: string) => {
  state.projectId = projectId;
};

const handleBaselineSchemaChange = async (baselineSchema: BaselineSchema) => {
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

const project = computed(() => {
  const project = projectStore.getProjectByUID(state.projectId || "");
  return project;
});

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

  const createdSchemaDesign = await schemaDesignStore.createSchemaDesign(
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
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.created-succeed"),
  });

  // Go to branch detail page after created.
  const [projectName, sheetId] = getProjectAndSchemaDesignSheetId(
    createdSchemaDesign.name
  );
  router.replace({
    name: "workspace.branch.detail",
    params: {
      projectName,
      branchName: sheetId,
    },
  });
};
</script>
