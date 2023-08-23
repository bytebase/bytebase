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
              {{ state.schemaDesignTitle }}
            </span>
            <NInput v-else v-model:value="state.schemaDesignTitle" />
            <NTag
              v-if="schemaDesign.type === SchemaDesign_Type.PERSONAL_DRAFT"
              type="warning"
              size="small"
              round
            >
              {{ $t("schema-designer.personal-draft") }}
            </NTag>
            <NDropdown
              v-if="!state.isEditing"
              class="max-w-[10rem]"
              trigger="click"
              :options="schemaDesignDraftDropdownOptions"
              :render-label="renderDraftsLabel"
              @select="handleSchemaDesignDraftSelect"
            >
              <NButton size="small" quaternary>
                <template #icon>
                  <NIcon size="16">
                    <History class="opacity-80" />
                  </NIcon>
                </template>
              </NButton>
            </NDropdown>
          </div>
          <div>
            <div class="w-full flex flex-row justify-between items-center">
              <div class="flex flex-row justify-end items-center space-x-2">
                <template v-if="!state.isEditing">
                  <NButton @click="handleEdit">{{ $t("common.edit") }}</NButton>
                  <NButton
                    v-if="!viewMode && !isSchemaDesignDraft"
                    type="primary"
                    @click="handleApplySchemaDesignClick"
                    >{{ $t("schema-designer.apply-to-database") }}</NButton
                  >
                </template>
                <template v-else>
                  <NButton @click="handleCancelEdit">{{
                    $t("common.cancel")
                  }}</NButton>
                  <NButton @click="handleSaveSchemaDesignDraft">{{
                    $t("schema-designer.save-draft")
                  }}</NButton>
                </template>
                <NButton
                  v-if="isSchemaDesignDraft && !viewMode"
                  type="primary"
                  @click="handleMergeSchemaDesign"
                  >{{ $t("schema-designer.merge-to-main") }}</NButton
                >
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
        <!-- Don't show delete button in view mode. -->
        <div v-if="!viewMode">
          <BBButtonConfirm
            :style="'DELETE'"
            :button-text="$t('schema-designer.delete-this-design')"
            :require-confirm="true"
            @confirm="deleteSchemaDesign"
          />
        </div>
      </div>

      <template v-if="viewMode" #footer>
        <div class="flex-1 flex items-center justify-between">
          <div></div>

          <div class="flex items-center justify-end gap-x-3">
            <NButton @click.prevent="emit('dismiss')">
              {{ $t("common.close") }}
            </NButton>
          </div>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>

  <ResolveConflictPanel
    v-if="state.showDiffEditor"
    :schema-design-name="state.schemaDesignName"
    @dismiss="state.showDiffEditor = false"
    @try-merge="handleMergeAfterConflictResolved"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { History } from "lucide-vue-next";
import {
  NButton,
  NDropdown,
  NDrawer,
  NDrawerContent,
  NIcon,
  NInput,
  NTag,
  useDialog,
  DropdownOption,
  NEllipsis,
} from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, h, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { DatabaseV1Name, InstanceV1EngineIcon } from "@/components/v2";
import { pushNotification, useDatabaseV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";
import { projectV1Slug } from "@/utils";
import ResolveConflictPanel from "./ResolveConflictPanel.vue";
import { mergeSchemaEditToMetadata } from "./common/util";
import SchemaDesigner from "./index.vue";

interface LocalState {
  schemaDesignTitle: string;
  // Pre edit or editing schema design name.
  schemaDesignName: string;
  isEditing: boolean;
  showDiffEditor: boolean;
}

const props = defineProps<{
  // Should be a schema design name of main branch.
  schemaDesignName: string;
  viewMode?: boolean;
}>();
const emit = defineEmits(["dismiss"]);

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const schemaDesignStore = useSchemaDesignStore();
const dialog = useDialog();
const state = reactive<LocalState>({
  schemaDesignTitle: "",
  schemaDesignName: props.schemaDesignName,
  isEditing: false,
  showDiffEditor: false,
});
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesigner>>();

const schemaDesign = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(state.schemaDesignName || "");
});

const isSchemaDesignDraft = computed(() => {
  return schemaDesign.value.type === SchemaDesign_Type.PERSONAL_DRAFT;
});

const schemaDesignDrafts = computed(() => {
  return schemaDesignStore.schemaDesignList.filter((schemaDesign) => {
    return (
      schemaDesign.type === SchemaDesign_Type.PERSONAL_DRAFT &&
      schemaDesign.baselineSheetName === props.schemaDesignName
    );
  });
});

const schemaDesignDraftDropdownOptions = computed(() => {
  return schemaDesignDrafts.value.map((schemaDesign) => {
    return {
      label: schemaDesign.title,
      key: schemaDesign.name,
    };
  });
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
});

watch(
  () => [state.schemaDesignName],
  () => {
    state.schemaDesignTitle = schemaDesign.value.title;
  },
  {
    immediate: true,
  }
);

const renderDraftsLabel = (option: DropdownOption) => {
  const schemaDesign = schemaDesignStore.getSchemaDesignByName(
    option.key as string
  );
  if (!schemaDesign) {
    // Should not reach here.
    return h(NEllipsis, {}, () => [h("span", {}, () => [option.label])]);
  }

  return h(
    "div",
    {
      class: "w-full flex justify-between items-center",
    },
    [
      h(
        NEllipsis,
        {
          class: "shrink !max-w-[4rem]",
        },
        [schemaDesign.title]
      ),
      h(
        "span",
        {
          class: "text-xs font-mono shrink-0 text-gray-400 ml-1",
        },
        [schemaDesign.etag.slice(0, 6)]
      ),
    ]
  );
};

const handleEdit = async () => {
  // Allow editing directly if it's a personal draft.
  if (schemaDesign.value.type === SchemaDesign_Type.PERSONAL_DRAFT) {
    state.isEditing = true;
  } else if (schemaDesign.value.type === SchemaDesign_Type.MAIN_BRANCH) {
    // Create a new draft if it's a main branch.
    const schemaDesignDraft = await schemaDesignStore.createSchemaDesignDraft(
      schemaDesign.value
    );
    // Select the newly created draft.
    state.schemaDesignName = schemaDesignDraft.name;
    // Trigger the edit mode.
    handleEdit();
  } else {
    throw new Error(
      `Unsupported schema design type: ${schemaDesign.value.type}`
    );
  }
};

const handleSchemaDesignDraftSelect = (name: string) => {
  state.schemaDesignName = name;
};

const handleCancelEdit = () => {
  state.isEditing = false;
  state.schemaDesignTitle = schemaDesign.value.title;

  const metadata = mergeSchemaEditToMetadata(
    schemaDesignerRef.value?.editableSchemas || [],
    schemaDesign.value.schemaMetadata || DatabaseMetadata.fromPartial({})
  );
  // If the metadata is changed, we need to rebuild the editing state.
  if (!isEqual(metadata, schemaDesign.value.schemaMetadata)) {
    schemaDesignerRef.value?.rebuildEditingState();
  }
};

const handleSaveSchemaDesignDraft = async () => {
  if (!state.isEditing) {
    return;
  }

  const designerState = schemaDesignerRef.value;
  if (!designerState) {
    throw new Error("schema designer is undefined");
  }
  if (state.schemaDesignTitle === "") {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Schema design name cannot be empty.",
    });
    return;
  }

  const updateMask = [];
  if (schemaDesign.value.title !== state.schemaDesignTitle) {
    updateMask.push("title");
  }
  const mergedMetadata = mergeSchemaEditToMetadata(
    designerState.editableSchemas,
    cloneDeep(
      schemaDesign.value.schemaMetadata || DatabaseMetadata.fromPartial({})
    )
  );
  if (!isEqual(mergedMetadata, schemaDesign.value.schemaMetadata)) {
    updateMask.push("metadata");
  }
  if (updateMask.length !== 0) {
    await schemaDesignStore.updateSchemaDesign(
      SchemaDesign.fromPartial({
        name: schemaDesign.value.name,
        title: state.schemaDesignTitle,
        engine: schemaDesign.value.engine,
        baselineSchema: schemaDesign.value.baselineSchema,
        schemaMetadata: mergedMetadata,
      }),
      updateMask
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
  }
  state.isEditing = false;
};

const handleMergeSchemaDesign = async () => {
  // If it's in edit mode, we need to save the draft first.
  if (state.isEditing) {
    await handleSaveSchemaDesignDraft();
  }

  try {
    await schemaDesignStore.mergeSchemaDesign({
      name: schemaDesign.value.name,
      targetName: props.schemaDesignName,
    });
  } catch (error: any) {
    // If there is conflict, we need to show the conflict and let user resolve it.
    if (error.code === Status.FAILED_PRECONDITION) {
      dialog.create({
        positiveText: t("schema-designer.diff-editor.resolve"),
        negativeText: t("common.cancel"),
        title: t("schema-designer.diff-editor.auto-merge-failed"),
        content: t("schema-designer.diff-editor.need-to-resolve-conflicts"),
        autoFocus: false,
        closable: false,
        maskClosable: false,
        closeOnEsc: false,
        onNegativeClick: () => {
          // nothing to do
        },
        onPositiveClick: () => {
          state.showDiffEditor = true;
        },
      });
    } else {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `Request error occurred`,
        description: error.details,
      });
    }
    return;
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.merge-to-main-successfully"),
  });
  // Auto select the main branch after merged.
  state.schemaDesignName = props.schemaDesignName;
};

const handleMergeAfterConflictResolved = () => {
  state.showDiffEditor = false;
  handleMergeSchemaDesign();
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
  const schemaDesignType = schemaDesign.value.type;
  await schemaDesignStore.deleteSchemaDesign(schemaDesign.value.name);
  if (schemaDesignType === SchemaDesign_Type.MAIN_BRANCH) {
    emit("dismiss");
  } else if (schemaDesignType === SchemaDesign_Type.PERSONAL_DRAFT) {
    state.schemaDesignName = props.schemaDesignName;
  }
};
</script>
