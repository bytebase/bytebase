<template>
  <NDrawer
    class="min-w-[calc(100%-10rem)] max-w-full"
    :show="true"
    :auto-focus="false"
    :close-on-esc="true"
    :native-scrollbar="true"
    resizable
    @update:show="(show: boolean) => !show && emit('dismiss')"
  >
    <NDrawerContent :title="$t('database.branch')" :closable="true">
      <div class="space-y-3 w-full overflow-x-auto">
        <div class="w-full flex flex-row justify-between items-center">
          <div class="w-full flex flex-row justify-start items-center">
            <span class="flex w-40 items-center shrink-0 text-sm">
              {{ $t("common.project") }}
            </span>
            <a
              class="normal-link inline-flex items-center"
              :href="`/project/${projectV1Slug(project)}`"
              >{{ project.title }}</a
            >
          </div>
          <div>
            <div class="w-full flex flex-row justify-between items-center">
              <div
                v-if="!viewMode && !state.isEditing"
                class="flex flex-row justify-end items-center space-x-2"
              >
                <NButton
                  v-if="parentBranch"
                  @click="() => handleMergeSchemaDesign()"
                  >{{ $t("schema-designer.merge-branch") }}</NButton
                >
                <NButton type="primary" @click="handleApplySchemaDesignClick">{{
                  $t("schema-designer.apply-to-database")
                }}</NButton>
              </div>
            </div>
          </div>
        </div>

        <div class="w-full flex flex-row justify-start items-center mt-1">
          <span class="flex w-40 items-center text-sm">{{
            $t("database.branch-name")
          }}</span>
          <div class="flex flex-row justify-start items-center gap-x-4">
            <BBTextField
              class="w-60 text-sm"
              :readonly="!state.isEditingTitle"
              :value="state.schemaDesignTitle"
              :placeholder="$t('database.branch-name')"
              @input="
                state.schemaDesignTitle = (
                  $event.target as HTMLInputElement
                ).value
              "
            />

            <NButton
              v-if="!state.isEditingTitle"
              text
              @click="state.isEditingTitle = true"
            >
              <template #icon>
                <NIcon>
                  <Pen class="w-4 h-auto" />
                </NIcon>
              </template>
            </NButton>
            <template v-else>
              <NButton text type="warning" @click="handleCancelEditTitle">
                <template #icon>
                  <NIcon>
                    <X />
                  </NIcon>
                </template>
              </NButton>
              <NButton type="success" text @click="handleSaveBranchTitle">
                <template #icon>
                  <NIcon>
                    <Check />
                  </NIcon>
                </template>
              </NButton>
            </template>

            <NTag v-if="parentBranch" round>
              {{ $t("schema-designer.parent-branch") }}:
              {{ parentBranch.title }}
            </NTag>
          </div>
        </div>

        <NDivider />

        <!-- 
        TODO(steven): show baseline database selectors.  

        <BaselineSchemaSelector
          :project-id="project.uid"
          :baseline-schema="state.baselineSchema"
          @update="handleBaselineSchemaChange"
        />

        <NDivider /> -->

        <div class="w-full flex flex-row justify-end gap-2">
          <template v-if="!state.isEditing">
            <NButton @click="handleEdit">{{ $t("common.edit") }}</NButton>
          </template>
          <template v-else>
            <NButton @click="handleCancelEdit">{{
              $t("common.cancel")
            }}</NButton>
            <NButton type="primary" @click="handleSaveSchemaDesignDraft">{{
              $t("common.save")
            }}</NButton>
          </template>
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
            :button-text="$t('database.delete-this-branch')"
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

  <MergeBranchPanel
    v-if="state.showDiffEditor"
    :source-branch-name="parentBranch!.name"
    :target-branch-name="state.schemaDesignName"
    @dismiss="state.showDiffEditor = false"
    @try-merge="handleMergeAfterConflictResolved"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { Pen, X, Check } from "lucide-vue-next";
import {
  NButton,
  NDrawer,
  NDrawerContent,
  NDivider,
  useDialog,
  NTag,
  NIcon,
} from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { pushNotification, useDatabaseV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import {
  SchemaDesign,
  SchemaDesign_Type,
} from "@/types/proto/v1/schema_design_service";
import { projectV1Slug } from "@/utils";
import MergeBranchPanel from "./MergeBranchPanel.vue";
import {
  generateForkedBranchName,
  mergeSchemaEditToMetadata,
  validateDatabaseMetadata,
} from "./common/util";
import SchemaDesigner from "./index.vue";

interface LocalState {
  schemaDesignTitle: string;
  // Pre edit or editing schema design name.
  schemaDesignName: string;
  isEditing: boolean;
  isEditingTitle: boolean;
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
  isEditingTitle: false,
  showDiffEditor: false,
});
const createdBranchName = ref<string>("");
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesigner>>();

const schemaDesign = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(state.schemaDesignName || "");
});

const parentBranch = computed(() => {
  if (schemaDesign.value.type === SchemaDesign_Type.PERSONAL_DRAFT) {
    return schemaDesignStore.getSchemaDesignByName(
      schemaDesign.value.baselineSheetName || ""
    );
  }
  return undefined;
});

const isSchemaDesignDraft = computed(() => {
  return schemaDesign.value.type === SchemaDesign_Type.PERSONAL_DRAFT;
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

const handleSaveBranchTitle = async () => {
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
  if (updateMask.length !== 0) {
    await schemaDesignStore.updateSchemaDesign(
      SchemaDesign.fromPartial({
        name: schemaDesign.value.name,
        title: state.schemaDesignTitle,
      }),
      updateMask
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
  }
  state.isEditingTitle = false;
};

const handleCancelEditTitle = () => {
  state.schemaDesignTitle = schemaDesign.value.title;
  state.isEditingTitle = false;
};

const handleEdit = async () => {
  // Allow editing directly if it's a personal draft.
  if (schemaDesign.value.type === SchemaDesign_Type.PERSONAL_DRAFT) {
    state.isEditing = true;
  } else if (schemaDesign.value.type === SchemaDesign_Type.MAIN_BRANCH) {
    // Create a new draft if it's a main branch.
    const schemaDesignDraft = await schemaDesignStore.createSchemaDesignDraft({
      ...schemaDesign.value,
      title: generateForkedBranchName(schemaDesign.value),
    });
    // Select the newly created draft.
    state.schemaDesignName = schemaDesignDraft.name;
    createdBranchName.value = schemaDesignDraft.name;
    // Trigger the edit mode.
    handleEdit();
  } else {
    throw new Error(
      `Unsupported schema design type: ${schemaDesign.value.type}`
    );
  }
};

const handleCancelEdit = async () => {
  const metadata = mergeSchemaEditToMetadata(
    schemaDesignerRef.value?.editableSchemas || [],
    cloneDeep(
      schemaDesign.value.schemaMetadata || DatabaseMetadata.fromPartial({})
    )
  );

  const parentBranchName = parentBranch.value?.name || "";
  // Only try to delete the branch if it's a new created personal draft.
  if (
    isSchemaDesignDraft.value &&
    createdBranchName.value === state.schemaDesignName
  ) {
    // If the metadata is changed, we need to confirm with user.
    if (!isEqual(metadata, schemaDesign.value.schemaMetadata)) {
      dialog.create({
        type: "warning",
        negativeText: t("common.cancel"),
        positiveText: t("schema-designer.diff-editor.discard-changes"),
        title: t("schema-designer.diff-editor.action-confirm"),
        content: t("schema-designer.diff-editor.discard-changes-confirm"),
        autoFocus: false,
        closable: false,
        maskClosable: false,
        closeOnEsc: false,
        onNegativeClick: () => {
          // nothing to do
        },
        onPositiveClick: async () => {
          await schemaDesignStore.deleteSchemaDesign(createdBranchName.value);
          state.schemaDesignName = parentBranchName;
          state.isEditing = false;
        },
      });
      return;
    } else {
      await schemaDesignStore.deleteSchemaDesign(createdBranchName.value);
      state.schemaDesignName = parentBranchName;
    }
  }

  // If the metadata is changed, we need to rebuild the editing state.
  if (!isEqual(metadata, schemaDesign.value.schemaMetadata)) {
    schemaDesignerRef.value?.rebuildEditingState();
  }
  state.isEditing = false;
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
      schemaDesign.value.baselineSchemaMetadata ||
        DatabaseMetadata.fromPartial({})
    )
  );
  const validationMessages = validateDatabaseMetadata(mergedMetadata);
  if (validationMessages.length > 0) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Invalid schema design",
      description: validationMessages.join("\n"),
    });
    return;
  }
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

  // If it's a personal draft, we will try to merge it to the parent branch.
  if (isSchemaDesignDraft.value) {
    await handleMergeSchemaDesign(true);
  }
};

const handleMergeSchemaDesign = async (ignoreNotify = false) => {
  // If it's in edit mode, we need to save the draft first.
  if (state.isEditing) {
    await handleSaveSchemaDesignDraft();
  }

  const parentBranchName = parentBranch.value!.name;
  try {
    await schemaDesignStore.mergeSchemaDesign({
      name: schemaDesign.value.name,
      targetName: parentBranchName,
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

  if (!ignoreNotify) {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.merge-to-main-successfully"),
    });
  }
  // Auto select the parent branch after merged.
  state.schemaDesignName = parentBranchName;
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
  await schemaDesignStore.deleteSchemaDesign(schemaDesign.value.name);
  emit("dismiss");
};
</script>
