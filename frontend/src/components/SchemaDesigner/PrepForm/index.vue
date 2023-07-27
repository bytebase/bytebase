<template>
  <DrawerContent>
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>{{ $t("schema-designer.quick-action") }}</span>
      </div>
    </template>

    <div
      class="space-y-3 w-[calc(100vw-8rem)] sm:w-[64rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div
        class="w-full border-b pb-2 mb-2 flex flex-row justify-between items-center"
      >
        <div class="flex flex-row justify-start items-center space-x-2">
          <template v-if="isViewing || isEditing">
            <NButton
              class="w-full flex flex-row justify-start items-center"
              @click="state.selectedSchemaDesign = undefined"
            >
              <heroicons-outline:chevron-left
                class="mr-1 w-4 h-auto text-gray-500 group-hover:text-gray-500 group-focus:text-gray-600"
              />
              {{ $t("common.back") }}
            </NButton>
            <NInput
              v-model:value="state.schemaDesignName"
              :disabled="isViewing"
            />
          </template>
        </div>
        <div>
          <NButton
            v-if="!isViewing && !isEditing"
            type="primary"
            @click="state.showCreatePanel = true"
          >
            <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
            <span>{{ $t("schema-designer.new-design") }}</span>
          </NButton>
          <div v-else class="w-full flex flex-row justify-between items-center">
            <div class="flex flex-row justify-end items-center space-x-2">
              <template v-if="isViewing">
                <NButton @click="state.isEditing = true">{{
                  $t("common.edit")
                }}</NButton>
                <NButton type="primary" @click="handleApplySchemaDesignClick">{{
                  $t("schema-designer.apply-to-database")
                }}</NButton>
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

      <template v-if="!isViewing && !isEditing">
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
        </div>
        <template v-if="state.selectedSchemaDesign">
          <SchemaDesigner
            ref="schemaDesignerRef"
            :readonly="isViewing"
            :engine="state.selectedSchemaDesign.engine"
            :schema-design="state.selectedSchemaDesign"
          />
        </template>
      </template>
    </div>
  </DrawerContent>

  <CreateSchemaDesignPanel
    v-if="state.showCreatePanel"
    @dismiss="state.showCreatePanel = false"
    @created="
      (schemaDesign) => {
        state.showCreatePanel = false;
        handleSchemaDesignItemClick(schemaDesign);
      }
    "
  />
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { NButton, NInput } from "naive-ui";
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
  useProjectV1Store,
} from "@/store";
import {
  useSchemaDesignList,
  useSchemaDesignStore,
} from "@/store/modules/schemaDesign";
import SchemaDesignTable from "./SchemaDesignTable.vue";
import SchemaDesigner from "../index.vue";
import { mergeSchemaEditToMetadata } from "../common/util";
import {
  DatabaseV1Name,
  DrawerContent,
  InstanceV1EngineIcon,
} from "@/components/v2";
import CreateSchemaDesignPanel from "../CreateSchemaDesignPanel.vue";
import { useRouter } from "vue-router";
import { projectV1Slug } from "@/utils";
import { useEventListener } from "@vueuse/core";

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
  isEditing: boolean;
  selectedSchemaDesign?: SchemaDesign;
  showCreatePanel: boolean;
}

defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
});
const emit = defineEmits(["dismiss"]);

useEventListener("keydown", (e) => {
  if (e.code == "Escape") {
    emit("dismiss");
  }
});

const { t } = useI18n();
const router = useRouter();
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesigner>>();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const schemaDesignStore = useSchemaDesignStore();
const { schemaDesignList, ready } = useSchemaDesignList();
const state = reactive<LocalState>({
  schemaDesignName: "",
  baselineSchema: {},
  isEditing: false,
  showCreatePanel: false,
});
const isViewing = computed(
  () => !!state.selectedSchemaDesign && !state.isEditing
);
const isEditing = computed(
  () => !!state.selectedSchemaDesign && state.isEditing
);

const project = computed(() => {
  return projectStore.getProjectByUID(state.baselineSchema.projectId || "");
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

const handleCancelEdit = () => {
  state.isEditing = false;
  if (state.selectedSchemaDesign) {
    handleSchemaDesignItemClick(state.selectedSchemaDesign);
  }
};

const handleUpdateSchemaDesign = async () => {
  if (!state.selectedSchemaDesign) {
    return;
  }

  const designerState = schemaDesignerRef.value;
  if (!designerState) {
    throw new Error("schema designer is undefined");
  }
  if (state.schemaDesignName === "") {
    return;
  }

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
  state.selectedSchemaDesign = await schemaDesignStore.updateSchemaDesign(
    SchemaDesign.fromPartial({
      name: state.selectedSchemaDesign.name,
      title: state.schemaDesignName,
      engine: state.selectedSchemaDesign.engine,
      baselineSchema: state.selectedSchemaDesign.baselineSchema,
      schemaMetadata: metadata,
    }),
    updateMarks
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
      schemaDesignName: state.selectedSchemaDesign?.name,
    },
  });
};
</script>
