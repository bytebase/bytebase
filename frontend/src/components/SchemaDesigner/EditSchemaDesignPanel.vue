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
                  @click="() => (state.showDiffEditor = true)"
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
              :placeholder="'feature/add-billing'"
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
                <NIcon size="16">
                  <Pen />
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

        <div class="w-full flex flex-row justify-start items-center mt-1">
          <span class="flex w-40 items-center text-sm font-medium">{{
            $t("schema-designer.baseline-version")
          }}</span>
        </div>

        <div class="w-full flex flex-row justify-start items-center">
          <span class="flex w-40 items-center shrink-0 text-sm">
            {{ $t("common.database") }}
          </span>
          <DatabaseInfo :database="baselineDatabase" />
        </div>

        <div class="w-full flex flex-row justify-start items-center">
          <span class="flex w-40 items-center shrink-0 text-sm">
            {{ $t("schema-designer.schema-version") }}
          </span>
          <div class="w-[calc(100%-10rem)]">
            <div
              v-if="changeHistory"
              class="w-full flex flex-row justify-start items-center"
            >
              <span class="block pr-2 w-full max-w-[80%] truncate">
                {{ changeHistory.version }} -
                {{ changeHistory.description }}
              </span>
              <span class="text-control-light">
                {{ humanizeDate(changeHistory.updateTime) }}
              </span>
            </div>
            <template v-else>
              {{ "Previously latest schema" }}
            </template>
          </div>
        </div>

        <NDivider />

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
    :source-branch-name="schemaDesign.baselineSheetName"
    :target-branch-name="state.schemaDesignName"
    @dismiss="state.showDiffEditor = false"
    @try-merge="handleMergeAfterConflictResolved"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
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
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import {
  pushNotification,
  useChangeHistoryStore,
  useCurrentUserV1,
  useDatabaseV1Store,
} from "@/store";
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
  validateBranchName,
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
const currentUser = useCurrentUserV1();
const databaseStore = useDatabaseV1Store();
const changeHistoryStore = useChangeHistoryStore();
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
  // Show parent branch when the current branch is a personal draft and it's not the new created one.
  if (
    schemaDesign.value.type === SchemaDesign_Type.PERSONAL_DRAFT &&
    state.schemaDesignName !== createdBranchName.value
  ) {
    return schemaDesignStore.getSchemaDesignByName(
      schemaDesign.value.baselineSheetName || ""
    );
  }
  return undefined;
});

const changeHistory = computed(() => {
  const changeHistoryName = `${baselineDatabase.value.name}/changeHistories/${schemaDesign.value.baselineChangeHistoryId}`;
  return changeHistoryStore.getChangeHistoryByName(changeHistoryName);
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
  const database = await databaseStore.getOrFetchDatabaseByName(
    schemaDesign.value.baselineDatabase
  );
  await changeHistoryStore.getOrFetchChangeHistoryListOfDatabase(database.name);
};

watch(
  () => [state.schemaDesignName],
  async () => {
    // Only change branch title when it's not a new created one.
    if (state.schemaDesignName !== createdBranchName.value) {
      state.schemaDesignTitle = schemaDesign.value.title;
    }
    await prepareBaselineDatabase();
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
      title: "Branch name cannot be empty.",
    });
    return;
  }
  if (!validateBranchName(state.schemaDesignTitle)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Branch name should valid characters: /^[a-zA-Z0-9-_/]+$/",
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
    const branchName = generateForkedBranchName(schemaDesign.value);
    const foundBranch = schemaDesignStore.schemaDesignList.find(
      (schemaDesign) => {
        return (
          schemaDesign.creator === `users/${currentUser.value.email}` &&
          schemaDesign.title === branchName
        );
      }
    );
    if (foundBranch) {
      // Show a confirm dialog to let user decide whether to use the existing branch or create a new one.
      dialog.create({
        negativeText: t("schema-designer.action.use-the-existing-branch"),
        positiveText: t("schema-designer.action.create-new-branch"),
        title: t("schema-designer.diff-editor.action-confirm"),
        content: t("schema-designer.diff-editor.duplicated-branch-name-found"),
        autoFocus: true,
        closable: true,
        maskClosable: true,
        closeOnEsc: true,
        onNegativeClick: () => {
          // Use the existing branch.
          state.schemaDesignName = foundBranch.name;
          handleEdit();
        },
        onPositiveClick: async () => {
          // Create a new personal branch.
          const newBranch = await schemaDesignStore.createSchemaDesignDraft({
            ...schemaDesign.value,
            title: branchName + `-${dayjs().format("YYYYMMDD")}`,
          });
          // Select the newly created draft.
          state.schemaDesignName = newBranch.name;
          createdBranchName.value = newBranch.name;
          // Trigger the edit mode.
          handleEdit();
        },
      });
    } else {
      const newBranch = await schemaDesignStore.createSchemaDesignDraft({
        ...schemaDesign.value,
        title: branchName,
      });
      // Select the newly created draft.
      state.schemaDesignName = newBranch.name;
      createdBranchName.value = newBranch.name;
      // Trigger the edit mode.
      handleEdit();
    }
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

  // Only try to delete the branch if it's a new created personal draft.
  if (
    isSchemaDesignDraft.value &&
    state.schemaDesignName === createdBranchName.value
  ) {
    const parentBranchName = schemaDesign.value.baselineSheetName;
    // If the metadata is changed, we need to confirm with user.
    if (!isEqual(metadata, schemaDesign.value.schemaMetadata)) {
      dialog.create({
        type: "warning",
        negativeText: t("common.cancel"),
        positiveText: t("schema-designer.diff-editor.discard-changes"),
        title: t("schema-designer.diff-editor.action-confirm"),
        content: t("schema-designer.diff-editor.discard-changes-confirm"),
        autoFocus: true,
        closable: true,
        maskClosable: true,
        closeOnEsc: true,
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
  // Don't update branch title for new created branch.
  if (schemaDesign.value.name !== createdBranchName.value) {
    if (schemaDesign.value.title !== state.schemaDesignTitle) {
      updateMask.push("title");
    }
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
    // Only show the notification when the branch is not a new created one.
    if (schemaDesign.value.name !== createdBranchName.value) {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("schema-designer.message.updated-succeed"),
      });
    }
  }
  state.isEditing = false;

  // If it's a personal draft, we will try to merge it to the parent branch.
  if (schemaDesign.value.name === createdBranchName.value) {
    await handleMergeSchemaDesign(true);
  }
};

const handleMergeSchemaDesign = async (ignoreNotify = false) => {
  // If it's in edit mode, we need to save the draft first.
  if (state.isEditing) {
    await handleSaveSchemaDesignDraft();
  }

  const parentBranchName = schemaDesign.value.baselineSheetName;
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
        autoFocus: true,
        closable: true,
        maskClosable: true,
        closeOnEsc: true,
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
  createdBranchName.value = "";
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
