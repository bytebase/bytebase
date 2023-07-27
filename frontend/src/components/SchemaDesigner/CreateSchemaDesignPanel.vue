<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    :close-on-esc="true"
    @update:show="(show: boolean) => !show && emit('dismiss')"
  >
    <NDrawerContent
      :title="$t('schema-designer.quick-action')"
      :closable="true"
    >
      <div
        class="space-y-3 py-1 w-[calc(100vw-8rem)] sm:w-[64rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
      >
        <div class="w-full flex flex-row justify-start items-center">
          <span class="flex w-40 items-center text-sm">{{
            $t("common.name")
          }}</span>
          <BBTextField
            class="w-60 !py-1.5"
            :value="state.schemaDesignName"
            :placeholder="$t('schema-designer.schema-design')"
            @input="
              state.schemaDesignName = ($event.target as HTMLInputElement).value
            "
          />
        </div>
        <BaselineSchemaSelector
          :baseline-schema="state.baselineSchema"
          @update="handleBaselineSchemaChange"
        />
        <SchemaDesigner
          ref="schemaDesignerRef"
          :key="refreshId"
          class="!mt-6"
          :readonly="readonly"
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
import { isUndefined, uniqueId } from "lodash-es";
import { NButton, NDrawer, NDrawerContent } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  ChangeHistory,
  DatabaseMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectV1ByUID,
} from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { databaseNamePrefix } from "@/store/modules/v1/common";
import { mergeSchemaEditToMetadata } from "./common/util";
import BaselineSchemaSelector from "./BaselineSchemaSelector.vue";
import SchemaDesigner from "./index.vue";
import { UNKNOWN_ID } from "@/types";

interface BaselineSchema {
  // The uid of project.
  projectId?: string;
  // The uid of database.
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

interface LocalState {
  schemaDesignName: string;
  baselineSchema: BaselineSchema;
  schemaDesign: SchemaDesign;
}

defineProps({
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
const state = reactive<LocalState>({
  schemaDesignName: "",
  baselineSchema: {},
  schemaDesign: SchemaDesign.fromPartial({}),
});
const refreshId = ref<string>("");
const readonly = computed(() => {
  return (
    isUndefined(state.baselineSchema.projectId) ||
    isUndefined(state.baselineSchema.databaseId) ||
    isUndefined(state.baselineSchema.changeHistory)
  );
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
    state.schemaDesignName &&
    state.baselineSchema.projectId &&
    state.baselineSchema.databaseId
  );
});

const cancelText = computed(() => {
  return t("common.cancel");
});

const confirmText = computed(() => {
  return t("common.create");
});

const handleBaselineSchemaChange = async (baselineSchema: BaselineSchema) => {
  state.baselineSchema = baselineSchema;
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
  if (state.schemaDesignName === "") {
    return;
  }

  const { project } = useProjectV1ByUID(state.baselineSchema.projectId || "");
  const database = useDatabaseV1Store().getDatabaseByUID(
    state.baselineSchema.databaseId || ""
  );

  const metadata = mergeSchemaEditToMetadata(
    designerState.editableSchemas,
    state.schemaDesign.baselineSchemaMetadata ||
      DatabaseMetadata.fromPartial({})
  );
  const baselineDatabase = `${database.instanceEntity.name}/${databaseNamePrefix}${state.baselineSchema.databaseId}`;
  const schemaVersion =
    !state.baselineSchema.changeHistory ||
    state.baselineSchema.changeHistory?.uid === String(UNKNOWN_ID)
      ? ""
      : state.baselineSchema.changeHistory?.name;

  const createdSchemaDesign = await schemaDesignStore.createSchemaDesign(
    project.value.name,
    SchemaDesign.fromPartial({
      title: state.schemaDesignName,
      // Keep schema empty in frontend. Backend will generate the design schema.
      schema: "",
      schemaMetadata: metadata,
      baselineSchema: state.schemaDesign.baselineSchema,
      baselineSchemaMetadata: state.schemaDesign.baselineSchemaMetadata,
      engine: state.schemaDesign.engine,
      baselineDatabase: baselineDatabase,
      schemaVersion: schemaVersion,
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
