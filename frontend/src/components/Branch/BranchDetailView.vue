<template>
  <div class="space-y-3 w-full overflow-x-auto px-4 pt-1">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="w-full flex flex-row justify-start items-center gap-x-2">
        <a
          class="normal-link inline-flex items-center"
          :href="`/project/${projectV1Slug(project)}`"
          >{{ project.title }}</a
        >
        <span class="ml-1 -mr-2">/</span>
        <NInput
          v-model:value="state.schemaDesignTitle"
          class="!w-auto"
          :passively-activated="true"
          :style="titleInputStyle"
          :readonly="!state.isEditingTitle"
          :placeholder="'feature/add-billing'"
          @focus="state.isEditingTitle = true"
          @blur="handleBranchTitleInputBlur"
        />
        <NTag v-if="parentBranch" round>
          {{ $t("schema-designer.parent-branch") }}:
          {{ parentBranch.title }}
        </NTag>
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

    <NDivider />

    <div class="w-full flex flex-row justify-between items-center mt-1 gap-4">
      <div class="flex flex-row justify-start items-center">
        <span class="mr-4">{{ $t("schema-designer.baseline-version") }}:</span>
        <DatabaseInfo
          class="flex-nowrap mr-4 shrink-0"
          :database="baselineDatabase"
        />
        <div>
          <NTooltip v-if="changeHistory" trigger="hover">
            <template #trigger> @{{ changeHistory.version }} </template>
            <div class="w-full flex flex-row justify-start items-center">
              <span class="block pr-2 w-full max-w-[32rem] truncate">
                {{ changeHistory.version }} -
                {{ changeHistory.description }}
              </span>
              <span class="opacity-60">
                {{ humanizeDate(changeHistory.updateTime) }}
              </span>
            </div>
          </NTooltip>
          <div v-else>
            {{ "Previously latest schema" }}
          </div>
        </div>
      </div>
      <div class="w-full flex flex-row justify-end gap-2">
        <template v-if="!state.isEditing">
          <NButton @click="handleEdit">{{ $t("common.edit") }}</NButton>
        </template>
        <template v-else>
          <NButton @click="handleCancelEdit">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" @click="handleSaveSchemaDesignDraft">{{
            $t("common.save")
          }}</NButton>
        </template>
      </div>
    </div>

    <div class="w-full h-[32rem]">
      <SchemaEditorV1
        :key="schemaEditorKey"
        :readonly="!state.isEditing"
        :project="project"
        :resource-type="'branch'"
        :branches="[schemaDesign]"
      />
    </div>
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

  <MergeBranchPanel
    v-if="state.showDiffEditor"
    :source-branch-name="state.schemaDesignName"
    :target-branch-name="schemaDesign.baselineSheetName"
    @dismiss="state.showDiffEditor = false"
    @merged="handleMergeAfterConflictResolved"
  />
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { cloneDeep, isEqual, uniqueId } from "lodash-es";
import { NButton, NDivider, NInput, NTooltip, useDialog, NTag } from "naive-ui";
import { Status } from "nice-grpc-common";
import { CSSProperties, computed, reactive, ref, watch } from "vue";
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import SchemaEditorV1 from "@/components/SchemaEditorV1/index.vue";
import {
  pushNotification,
  useChangeHistoryStore,
  useCurrentUserV1,
  useDatabaseV1Store,
  useSchemaEditorV1Store,
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
} from "./utils";

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
const schemaEditorKey = ref<string>(uniqueId());

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

const titleInputStyle = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    "--n-color-disabled": "transparent",
    "--n-font-size": "16px",
  };
  const border = state.isEditingTitle
    ? "1px solid var(--color-control-border)"
    : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

onMounted(async () => {
  if (
    schemaDesign.value.type === SchemaDesign_Type.PERSONAL_DRAFT &&
    state.schemaDesignName !== createdBranchName.value
  ) {
    await schemaDesignStore.getOrFetchSchemaDesignByName(
      schemaDesign.value.baselineSheetName || ""
    );
  }
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

const handleBranchTitleInputBlur = async () => {
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
      title: "Branch name valid characters: /^[a-zA-Z][a-zA-Z0-9-_/]+$/",
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
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    schemaDesign.value.name
  );
  if (!branchSchema) {
    return;
  }

  const editableSchemas = branchSchema.schemaList;
  const metadata = mergeSchemaEditToMetadata(
    editableSchemas,
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

  if (!isEqual(metadata, schemaDesign.value.schemaMetadata)) {
    // If the metadata is changed, we need to rebuild the editing state.
    schemaEditorKey.value = uniqueId();
  }
  state.isEditing = false;
};

const handleSaveSchemaDesignDraft = async () => {
  if (!state.isEditing) {
    return;
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
  if (state.schemaDesignName !== createdBranchName.value) {
    if (schemaDesign.value.title !== state.schemaDesignTitle) {
      updateMask.push("title");
    }
  }

  const schemaEditorV1Store = useSchemaEditorV1Store();
  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    schemaDesign.value.name
  );
  if (!branchSchema) {
    return;
  }
  const mergedMetadata = mergeSchemaEditToMetadata(
    branchSchema.schemaList,
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
  const branchName = schemaDesign.value.name;
  try {
    await schemaDesignStore.mergeSchemaDesign({
      name: branchName,
      targetName: parentBranchName,
    });
  } catch (error: any) {
    // If there is conflict, we need to show the conflict and let user resolve it.
    if (error.code === Status.FAILED_PRECONDITION) {
      dialog.create({
        negativeText: t("schema-designer.save-draft"),
        positiveText: t("schema-designer.diff-editor.resolve"),
        title: t("schema-designer.diff-editor.auto-merge-failed"),
        content: t("schema-designer.diff-editor.need-to-resolve-conflicts"),
        autoFocus: true,
        closable: true,
        maskClosable: true,
        closeOnEsc: true,
        onNegativeClick: () => {
          // Clear the created branch state if save draft clicked.
          if (branchName === createdBranchName.value) {
            createdBranchName.value = "";
            state.schemaDesignTitle = schemaDesign.value.title;
          }
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
  // Delete the draft after merged.
  await schemaDesignStore.deleteSchemaDesign(branchName);
};

const handleMergeAfterConflictResolved = (branchName: string) => {
  createdBranchName.value = "";
  state.schemaDesignName = branchName;
  state.showDiffEditor = false;
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
  router.replace({
    name: "workspace.branch",
  });
};
</script>
