<template>
  <NDrawer
    class="min-w-[calc(100%-10rem)] max-w-full"
    :show="true"
    :auto-focus="false"
    :trap-focus="false"
    :close-on-esc="true"
    :native-scrollbar="true"
    resizable
    @update:show="(show: boolean) => !show && emit('dismiss')"
  >
    <NDrawerContent :title="$t('database.new-branch')" :closable="true">
      <div class="space-y-3 w-full overflow-x-auto">
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
              state.schemaDesignTitle = (
                $event.target as HTMLInputElement
              ).value
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
        <SchemaDesigner
          ref="schemaDesignerRef"
          :key="refreshId"
          class="!mt-6"
          :readonly="true"
          :engine="state.schemaDesign.engine"
          :schema-design="state.schemaDesign"
        />
      </div>

      <template #footer>
        <div class="flex-1 flex items-center justify-between">
          <div></div>

          <div class="flex items-center justify-end gap-x-3">
            <NButton @click.prevent="cancel">
              {{ cancelText }}
            </NButton>
            <NButton
              type="primary"
              :disabled="!allowConfirm"
              @click.prevent="handleConfirm"
            >
              {{ confirmText }}
            </NButton>
          </div>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { cloneDeep, uniqueId } from "lodash-es";
import { NButton, NDrawer, NDrawerContent, NDivider } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectV1ByUID,
  useSheetV1Store,
} from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { databaseNamePrefix } from "@/store/modules/v1/common";
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
import { mergeSchemaEditToMetadata, validateBranchName } from "./common/util";
import SchemaDesigner from "./index.vue";

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
const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "created", schemaDesign: SchemaDesign): void;
}>();

const { t } = useI18n();
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesigner>>();
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

const cancelText = computed(() => {
  return t("common.cancel");
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

const cancel = () => {
  emit("dismiss");
};

const handleConfirm = async () => {
  if (!state.schemaDesign) {
    return;
  }

  const designerState = schemaDesignerRef.value;
  if (!designerState) {
    // Should not happen.
    throw new Error("schema designer is undefined");
  }
  if (!validateBranchName(state.schemaDesignTitle)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Branch name should valid characters: /^[a-zA-Z0-9-_/]+$/",
    });
    return;
  }

  const { project } = useProjectV1ByUID(state.projectId || "");
  const database = useDatabaseV1Store().getDatabaseByUID(
    state.baselineSchema.databaseId || ""
  );

  const metadata = mergeSchemaEditToMetadata(
    designerState.editableSchemas,
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
    })
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.created-succeed"),
  });
  emit("created", createdSchemaDesign);
};
</script>
