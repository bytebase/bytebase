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
        class="space-y-3 w-[calc(100vw-24rem)] min-w-[64rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
      >
        <div
          class="w-full border-b pb-2 mb-2 flex flex-row justify-between items-center"
        >
          <div class="flex flex-row justify-start items-center space-x-2">
            <span v-if="!state.isEditing">
              {{ state.schemaDesignName }}
            </span>
            <NInput v-else v-model:value="state.schemaDesignName" />
          </div>
          <div>
            <div class="w-full flex flex-row justify-between items-center">
              <div class="flex flex-row justify-end items-center space-x-2">
                <template v-if="!state.isEditing">
                  <NButton @click="state.isEditing = true">{{
                    $t("common.edit")
                  }}</NButton>
                  <NButton
                    type="primary"
                    @click="handleApplySchemaDesignClick"
                    >{{ $t("schema-designer.apply-to-database") }}</NButton
                  >
                </template>
                <template v-else>
                  <NButton @click="handleCancelEdit">{{
                    $t("common.cancel")
                  }}</NButton>
                  <NButton type="primary" @click="handleUpdateSchemaDesign">{{
                    $t("common.update")
                  }}</NButton>
                </template>
              </div>
            </div>
          </div>
        </div>

        <div class="w-full flex flex-row justify-start items-center space-x-6">
          <div class="flex flex-row justify-start items-center">
            <span class="text-sm">{{ $t("common.project") }}</span>
            <span class="mx-1">-</span>
            <a
              class="normal-link inline-flex items-center"
              :href="`/project/${projectV1Slug(project)}`"
              >{{ project.title }}</a
            >
          </div>
          <div class="flex flex-row justify-start items-center">
            <span class="text-sm">{{
              $t("schema-designer.baseline-database")
            }}</span>
            <span class="mx-1">-</span>
            <div class="flex flex-row justify-start items-center space-x-0.5">
              <InstanceV1EngineIcon
                :instance="baselineDatabase.instanceEntity"
              />
              <DatabaseV1Name :database="baselineDatabase" />
            </div>
          </div>
        </div>
        <SchemaDesigner
          ref="schemaDesignerRef"
          :readonly="!state.isEditing"
          :engine="schemaDesign.engine"
          :schema-design="schemaDesign"
        />
        <div>
          <BBButtonConfirm
            :style="'DELETE'"
            :button-text="$t('schema-designer.delete-this-design')"
            :require-confirm="true"
            @confirm="deleteSchemaDesign"
          />
        </div>
      </div>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { NButton, NDrawer, NDrawerContent, NInput } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { pushNotification, useDatabaseV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import SchemaDesigner from "./index.vue";
import { mergeSchemaEditToMetadata } from "./common/util";
import { DatabaseV1Name, InstanceV1EngineIcon } from "@/components/v2";
import { useRouter } from "vue-router";
import { projectV1Slug } from "@/utils";

interface LocalState {
  schemaDesignName: string;
  isEditing: boolean;
}

const props = defineProps<{
  schemaDesignName: string;
}>();
const emit = defineEmits(["dismiss"]);

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const schemaDesignStore = useSchemaDesignStore();
const state = reactive<LocalState>({
  schemaDesignName: "",
  isEditing: false,
});
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesigner>>();

const schemaDesign = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(props.schemaDesignName || "");
});

const baselineDatabase = computed(() => {
  return databaseStore.getDatabaseByName(schemaDesign.value.baselineDatabase);
});

const project = computed(() => {
  return baselineDatabase.value.projectEntity;
});

const prepareBaselineDatabase = async () => {
  await databaseStore.getOrFetchDatabaseByName(
    schemaDesign.value.baselineDatabase
  );
};

onMounted(async () => {
  await prepareBaselineDatabase();
  state.schemaDesignName = schemaDesign.value.title;
});

const handleCancelEdit = () => {
  state.isEditing = false;
  state.schemaDesignName = schemaDesign.value.title;

  const metadata = mergeSchemaEditToMetadata(
    schemaDesignerRef.value?.editableSchemas || [],
    schemaDesign.value.schemaMetadata || DatabaseMetadata.fromPartial({})
  );
  // If the metadata is changed, we need to rebuild the editing state.
  if (!isEqual(metadata, schemaDesign.value.schemaMetadata)) {
    schemaDesignerRef.value?.rebuildEditingState();
  }
};

const handleUpdateSchemaDesign = async () => {
  if (!state.isEditing) {
    return;
  }

  const designerState = schemaDesignerRef.value;
  if (!designerState) {
    throw new Error("schema designer is undefined");
  }
  if (state.schemaDesignName === "") {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Schema design name cannot be empty.",
    });
    return;
  }

  const updateMask = [];
  if (schemaDesign.value.title !== state.schemaDesignName) {
    updateMask.push("title");
  }
  const mergedMetadata = mergeSchemaEditToMetadata(
    designerState.editableSchemas,
    cloneDeep(
      schemaDesign.value.schemaMetadata || DatabaseMetadata.fromPartial({})
    )
  );
  if (!isEqual(mergedMetadata, schemaDesign.value.schemaMetadata)) {
    updateMask.push("schema");
  }
  await schemaDesignStore.updateSchemaDesign(
    SchemaDesign.fromPartial({
      name: schemaDesign.value.name,
      title: state.schemaDesignName,
      engine: schemaDesign.value.engine,
      baselineSchema: schemaDesign.value.baselineSchema,
      schemaMetadata: mergedMetadata,
    }),
    updateMask
  );
  state.isEditing = false;
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.updated-succeed"),
  });
};

const handleApplySchemaDesignClick = () => {
  router.push({
    name: "workspace.sync-schema",
    query: {
      schemaDesignName: schemaDesign.value.name,
    },
  });
};

const deleteSchemaDesign = async () => {
  await schemaDesignStore.deleteSchemaDesign(schemaDesign.value.name);
  emit("dismiss");
};
</script>
