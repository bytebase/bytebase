<template>
  <DrawerContent>
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>{{ $t("schema-designer.self") }}</span>
      </div>
    </template>

    <div
      class="space-y-3 w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div>
        <NRadioGroup v-model:value="state.tab">
          <NRadio
            :value="'LIST'"
            :label="$t('schema-designer.existing-schema-design')"
          />
          <NRadio
            :value="'CREATE'"
            :label="$t('schema-designer.new-schema-design')"
          />
        </NRadioGroup>
      </div>

      <template v-if="!isCreating && !isEditing">
        <SchemaDesignTable
          v-if="ready"
          :schema-designs="schemaDesignList"
          @click="handleSchemaDesignItemClick"
        />
        <div v-else class="w-full h-[20rem] flex items-center justify-center">
          <BBSpin />
        </div>
      </template>
      <template v-else>
        <div class="w-full flex flex-row justify-start items-center">
          <span class="flex w-40 items-center text-sm">{{
            $t("common.name")
          }}</span>
          <BBTextField
            class="w-60 !py-1.5"
            :value="state.schemaDesignName"
            @input="
              state.schemaDesignName = ($event.target as HTMLInputElement).value
            "
          />
        </div>
        <BaselineSchemaSelector
          v-if="isCreating"
          :baseline-schema="state.baselineSchema"
          @update="handleBaselineSchemaChange"
        />
        <div v-if="isEditing" class="w-full">
          <div class="flex flex-row justify-start items-center">
            <span class="text-sm w-40">{{
              $t("schema-designer.baseline-database")
            }}</span>
            <InstanceV1EngineIcon
              class="mr-1"
              :instance="
                databaseStore.getDatabaseByUID(
                  state.baselineSchema.databaseId || ''
                ).instanceEntity
              "
            />
            <DatabaseV1Name
              :database="
                databaseStore.getDatabaseByUID(
                  state.baselineSchema.databaseId || ''
                )
              "
            />
          </div>
        </div>
        <template v-if="state.selectedSchemaDesign">
          <SchemaDesigner
            ref="schemaDesignerRef"
            :key="schemaDesignId"
            :engine="state.selectedSchemaDesign.engine"
            :schema-design="state.selectedSchemaDesign"
          />
        </template>
      </template>
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
  </DrawerContent>
</template>

<script lang="ts" setup>
import { isEqual, uniqueId } from "lodash-es";
import { NRadioGroup, NRadio } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  ChangeHistory,
  DatabaseMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import {
  pushNotification,
  useChangeHistoryStore,
  useDatabaseV1Store,
  useProjectV1ByUID,
} from "@/store";
import {
  useSchemaDesignList,
  useSchemaDesignStore,
} from "@/store/modules/schemaDesign";
import SchemaDesignTable from "./SchemaDesignTable.vue";
import BaselineSchemaSelector from "../BaselineSchemaSelector.vue";
import SchemaDesigner from "../index.vue";
import { databaseNamePrefix } from "@/store/modules/v1/common";
import { watch } from "vue";
import { mergeSchemaEditToMetadata } from "../common/util";
import { DatabaseV1Name, InstanceV1EngineIcon } from "@/components/v2";

interface BaselineSchema {
  // The uid of project.
  projectId?: string;
  // The uid of database.
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

interface LocalState {
  tab: "LIST" | "CREATE";
  schemaDesignName: string;
  baselineSchema: BaselineSchema;
  selectedSchemaDesign?: SchemaDesign;
}

defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
});
const emit = defineEmits(["dismiss"]);

const { t } = useI18n();
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesigner>>();
const databaseStore = useDatabaseV1Store();
const schemaDesignStore = useSchemaDesignStore();
const { schemaDesignList, ready } = useSchemaDesignList();
const state = reactive<LocalState>({
  tab: "LIST",
  schemaDesignName: "",
  baselineSchema: {},
});
const isCreating = computed(() => state.tab === "CREATE");
const isEditing = computed(
  () => state.tab === "LIST" && !!state.selectedSchemaDesign
);

watch(
  () => state.tab,
  () => {
    state.selectedSchemaDesign = undefined;
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
  return undefined;
};

const schemaDesignId = computed(() => {
  if (!state.selectedSchemaDesign || !state.selectedSchemaDesign.name) {
    return uniqueId();
  } else {
    return state.selectedSchemaDesign.name;
  }
});

const allowConfirm = computed(() => {
  if (isCreating.value) {
    return (
      state.schemaDesignName &&
      state.baselineSchema.projectId &&
      state.baselineSchema.databaseId
    );
  } else if (isEditing.value) {
    return state.schemaDesignName;
  }
  return false;
});

const cancelText = computed(() => {
  if (isCreating.value) {
    return t("common.cancel");
  } else if (isEditing.value) {
    return t("common.back");
  }
  return t("common.cancel");
});

const confirmText = computed(() => {
  if (isCreating.value) {
    return t("common.create");
  } else if (isEditing.value) {
    return t("common.update");
  }
  return t("common.next");
});

const handleSchemaDesignItemClick = async (schemaDesign: SchemaDesign) => {
  state.schemaDesignName = schemaDesign.title;
  state.selectedSchemaDesign = schemaDesign;
  const database = await databaseStore.getOrFetchDatabaseByName(
    schemaDesign.baselineDatabase
  );
  const baselineSchema: BaselineSchema = {
    projectId: database.projectEntity.uid,
    databaseId: database.uid,
  };
  if (schemaDesign.schemaVersion) {
    const changeHistory =
      await useChangeHistoryStore().getOrFetchChangeHistoryByName(
        schemaDesign.schemaVersion
      );
    baselineSchema.changeHistory = changeHistory;
  }
  state.baselineSchema = baselineSchema;
};

const handleBaselineSchemaChange = async (baselineSchema: BaselineSchema) => {
  state.baselineSchema = baselineSchema;

  if (isCreating.value) {
    state.selectedSchemaDesign = await prepareSchemaDesign();
  }
};

const cancel = () => {
  if (isEditing.value) {
    state.tab = "LIST";
    state.selectedSchemaDesign = undefined;
  } else {
    emit("dismiss");
  }
};

const handleConfirm = async () => {
  if (!state.selectedSchemaDesign) {
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

  if (isCreating.value) {
    const metadata = mergeSchemaEditToMetadata(
      designerState.editableSchemas,
      state.selectedSchemaDesign.baselineSchemaMetadata ||
        DatabaseMetadata.fromPartial({})
    );
    const baselineDatabase = `${database.instanceEntity.name}/${databaseNamePrefix}${state.baselineSchema.databaseId}`;
    await schemaDesignStore.createSchemaDesign(
      project.value.name,
      SchemaDesign.fromPartial({
        title: state.schemaDesignName,
        // Keep schema empty in frontend. Backend will generate the design schema.
        schema: "",
        schemaMetadata: metadata,
        baselineSchema: state.selectedSchemaDesign.baselineSchema,
        baselineSchemaMetadata:
          state.selectedSchemaDesign.baselineSchemaMetadata,
        engine: state.selectedSchemaDesign.engine,
        baselineDatabase: baselineDatabase,
        schemaVersion: state.baselineSchema.changeHistory?.name || "",
      })
    );
    state.tab = "LIST";
    state.selectedSchemaDesign = undefined;
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.created-succeed"),
    });
  } else {
    const updateMarks = [];
    if (state.selectedSchemaDesign.title !== state.schemaDesignName) {
      updateMarks.push("title");
    }
    const metadata = mergeSchemaEditToMetadata(
      designerState.editableSchemas,
      state.selectedSchemaDesign.schemaMetadata ||
        DatabaseMetadata.fromPartial({})
    );
    if (isEqual(metadata, state.selectedSchemaDesign.schemaMetadata)) {
      updateMarks.push("schema");
    }
    await schemaDesignStore.updateSchemaDesign(
      SchemaDesign.fromPartial({
        name: state.selectedSchemaDesign.name,
        title: state.schemaDesignName,
        engine: state.selectedSchemaDesign.engine,
        baselineSchema: state.selectedSchemaDesign.baselineSchema,
        schemaMetadata: metadata,
      }),
      updateMarks
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
  }
};
</script>
