<template>
  <div class="space-y-3 w-full overflow-x-auto" v-bind="$attrs">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="w-full flex flex-row justify-start items-center gap-x-2">
        <NInput
          v-model:value="state.branchId"
          class="!w-auto"
          :passively-activated="true"
          :style="branchIdInputStyle"
          :readonly="!state.isEditingBranchId"
          :placeholder="'feature/add-billing'"
          @focus="state.isEditingBranchId = true"
          @blur="handleBranchIdInputBlur"
        />
        <NTag v-if="parentBranch" round>
          {{ $t("schema-designer.parent-branch") }}:
          {{ parentBranch.branchId }}
        </NTag>
      </div>
      <div>
        <div class="w-full flex flex-row justify-between items-center">
          <div
            v-if="!viewMode"
            class="flex flex-row justify-end items-center space-x-2"
          >
            <template v-if="!state.isEditing">
              <NButton @click="handleEdit">{{ $t("common.edit") }}</NButton>
              <NButton
                :disabled="!ready"
                :loading="!ready"
                @click="handleMergeBranch"
                >{{ $t("schema-designer.merge-branch") }}</NButton
              >
              <NButton
                type="primary"
                @click="selectTargetDatabasesContext.show = true"
                >{{ $t("schema-designer.apply-to-database") }}</NButton
              >
            </template>
            <template v-else>
              <NButton @click="handleCancelEdit">{{
                $t("common.cancel")
              }}</NButton>
              <NButton
                type="primary"
                :loading="state.isSaving"
                @click="handleSaveBranch"
                >{{ $t("common.save") }}</NButton
              >
            </template>
          </div>
        </div>
      </div>
    </div>

    <NDivider />

    <div
      class="w-full flex flex-row justify-between items-center text-sm mt-1 gap-4"
    >
      <div class="flex flex-row justify-start items-center opacity-80">
        <span class="mr-4 shrink-0"
          >{{ $t("schema-designer.baseline-version") }}:</span
        >
        <DatabaseInfo
          class="flex-nowrap mr-4 shrink-0"
          :database="baselineDatabase"
        />
      </div>
    </div>

    <div class="w-full h-[32rem]">
      <SchemaDesignEditor
        :key="schemaEditorKey"
        :project="project"
        :readonly="!state.isEditing"
        :branch="branch"
      />
    </div>
    <!-- Don't show delete button in view mode. -->
    <div v-if="!viewMode">
      <BBButtonConfirm
        :style="'DELETE'"
        :button-text="$t('database.delete-this-branch')"
        :require-confirm="true"
        @confirm="deleteBranch"
      />
    </div>
  </div>

  <TargetDatabasesSelectPanel
    v-if="selectTargetDatabasesContext.show"
    :project-id="project.uid"
    :engine="branch.engine"
    :selected-database-id-list="[]"
    @close="selectTargetDatabasesContext.show = false"
    @update="handleSelectedDatabaseIdListChanged"
  />

  <MergeBranchPanel
    v-if="state.showDiffEditor && mergeBranchPanelContext"
    :source-branch-name="mergeBranchPanelContext.sourceBranchName"
    :target-branch-name="mergeBranchPanelContext.targetBranchName"
    @dismiss="state.showDiffEditor = false"
    @merged="handleMergeAfterConflictResolved"
  />
</template>

<script lang="ts" setup>
import { asyncComputed } from "@vueuse/core";
import dayjs from "dayjs";
import { cloneDeep, head, isEqual, uniqueId } from "lodash-es";
import { NButton, NDivider, NInput, useDialog, NTag } from "naive-ui";
import { Status } from "nice-grpc-common";
import { CSSProperties, computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import {
  mergeSchemaEditToMetadata,
  validateDatabaseMetadata,
} from "@/components/SchemaEditorV1/utils";
import TargetDatabasesSelectPanel from "@/components/SyncDatabaseSchema/TargetDatabasesSelectPanel.vue";
import { branchServiceClient } from "@/grpcweb";
import {
  pushNotification,
  useDatabaseV1Store,
  useSchemaEditorV1Store,
} from "@/store";
import { useBranchList, useBranchStore } from "@/store/modules/branch";
import {
  getProjectAndBranchId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { Branch } from "@/types/proto/v1/branch_service";
import { projectV1Slug } from "@/utils";
import { provideSQLCheckContext } from "../SQLCheck";
import { fetchBaselineMetadataOfBranch } from "../SchemaEditorV1/utils/branch";
import MergeBranchPanel from "./MergeBranchPanel.vue";
import SchemaDesignEditor from "./SchemaDesignEditor.vue";
import { generateForkedBranchName, validateBranchName } from "./utils";

interface LocalState {
  branchId: string;
  isEditing: boolean;
  isEditingBranchId: boolean;
  showDiffEditor: boolean;
  isSaving: boolean;
}

const props = defineProps<{
  // Should be a schema design name of main branch.
  branch: Branch;
  viewMode?: boolean;
}>();

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const branchStore = useBranchStore();
const { branchList, ready } = useBranchList(
  getProjectAndBranchId(props.branch.name)[0]
);
const { runSQLCheck } = provideSQLCheckContext();
const dialog = useDialog();
const state = reactive<LocalState>({
  branchId: "",
  isEditing: false,
  isEditingBranchId: false,
  showDiffEditor: false,
  isSaving: false,
});
const mergeBranchPanelContext = ref<{
  sourceBranchName: string;
  targetBranchName: string;
}>();
const schemaEditorKey = ref<string>(uniqueId());
const selectTargetDatabasesContext = ref<{
  show: boolean;
}>({
  show: false,
});

const branch = computed(() => {
  return props.branch;
});

const parentBranch = asyncComputed(async () => {
  // Show parent branch when the current branch is a personal draft and it's not the new created one.
  if (branch.value.parentBranch !== "") {
    return await branchStore.fetchBranchByName(
      branch.value.parentBranch,
      true /* useCache */
    );
  }
  return undefined;
}, undefined);

const baselineDatabase = computed(() => {
  return databaseStore.getDatabaseByName(branch.value.baselineDatabase);
});

const project = computed(() => {
  return baselineDatabase.value.projectEntity;
});

const branchIdInputStyle = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    minWidth: "10rem",
    "--n-color-disabled": "transparent",
    "--n-font-size": "20px",
  };
  const border = state.isEditingBranchId
    ? "1px solid rgb(var(--color-control-border))"
    : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

const prepareBaselineDatabase = async () => {
  await databaseStore.getOrFetchDatabaseByName(branch.value.baselineDatabase);
};

watch(
  () => [props.branch],
  async () => {
    state.branchId = branch.value.branchId;
    await prepareBaselineDatabase();
    // Prepare the parent branch for personal draft.
    if (branch.value.parentBranch !== "") {
      await branchStore.fetchBranchByName(
        branch.value.parentBranch,
        true /* useCache */
      );
    }
  },
  {
    immediate: true,
  }
);

const handleBranchIdInputBlur = async () => {
  if (state.branchId === "") {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Branch name cannot be empty.",
    });
    return;
  }
  if (!validateBranchName(state.branchId)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Branch name valid characters: /^[a-zA-Z][a-zA-Z0-9-_/]+$/",
    });
    return;
  }

  const updateMask = [];
  if (branch.value.branchId !== state.branchId) {
    updateMask.push("branch_id");
  }
  if (updateMask.length !== 0) {
    await branchStore.updateBranch(
      Branch.fromPartial({
        name: branch.value.name,
        branchId: state.branchId,
        baselineDatabase: branch.value.baselineDatabase,
      }),
      updateMask
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
  }
  state.isEditingBranchId = false;
};

const handleMergeBranch = () => {
  const tempList = branchList.value.filter((item) => {
    const [projectName] = getProjectAndBranchId(item.name);
    return (
      `${projectNamePrefix}${projectName}` === project.value.name &&
      item.engine === branch.value.engine &&
      item.name !== branch.value.name
    );
  });
  const targetBranchName = parentBranch.value
    ? parentBranch.value.name
    : head(tempList)?.name;
  if (!targetBranchName) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "No branch to merge.",
    });
    return;
  }

  mergeBranchPanelContext.value = {
    sourceBranchName: branch.value.name,
    targetBranchName: targetBranchName,
  };
  state.showDiffEditor = true;
};

const handleEdit = async () => {
  state.isEditing = true;
};

const handleCancelEdit = async () => {
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    branch.value.name
  );
  if (!branchSchema) {
    return;
  }

  const baselineMetadata = await fetchBaselineMetadataOfBranch(
    branchSchema.branch
  );
  const mergedMetadata = mergeSchemaEditToMetadata(
    branchSchema.schemaList,
    cloneDeep(baselineMetadata)
  );
  if (!isEqual(mergedMetadata, branch.value.schemaMetadata)) {
    // If the metadata is changed, we need to rebuild the editing state.
    schemaEditorKey.value = uniqueId();
  }
  state.isEditing = false;
};

const handleSaveBranch = async () => {
  if (!state.isEditing) {
    return;
  }
  if (state.isSaving) {
    return;
  }
  const check = runSQLCheck.value;
  if (check && !(await check())) {
    return;
  }

  const updateMask = [];
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    branch.value.name
  );
  if (!branchSchema) {
    return;
  }

  const baselineMetadata = await fetchBaselineMetadataOfBranch(
    branchSchema.branch
  );
  const mergedMetadata = mergeSchemaEditToMetadata(
    branchSchema.schemaList,
    cloneDeep(baselineMetadata)
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
  if (!isEqual(mergedMetadata, branch.value.schemaMetadata)) {
    updateMask.push("metadata");
  }

  state.isSaving = true;
  if (updateMask.length !== 0) {
    if (branch.value.parentBranch === "") {
      const branchName = generateForkedBranchName(branch.value);
      const newBranch = await branchStore.createBranchDraft(
        project.value.name,
        branchName,
        branch.value.name
      );
      try {
        await branchStore.mergeBranch({
          name: branch.value.name,
          headBranch: newBranch.name,
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
              // Go to draft branch detail page after merge failed.
              const [_, branchId] = getProjectAndBranchId(newBranch.name);
              state.isEditing = false;
              router.replace({
                name: "workspace.project.branch.detail",
                params: {
                  projectSlug: projectV1Slug(project.value),
                  branchName: branchId,
                },
              });
            },
            onPositiveClick: () => {
              state.showDiffEditor = true;
              mergeBranchPanelContext.value = {
                sourceBranchName: newBranch.name,
                targetBranchName: branch.value.name,
              };
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
        state.isSaving = false;
        return;
      }

      // Delete the draft after merged.
      await branchStore.deleteBranch(newBranch.name);
      // Fetch the latest schema design after merged.
      await branchStore.fetchBranchByName(
        branch.value.name,
        false /* !useCache */
      );
    } else {
      await branchStore.updateBranch(
        Branch.fromPartial({
          name: branch.value.name,
          branchId: state.branchId,
          engine: branch.value.engine,
          baselineDatabase: branch.value.baselineDatabase,
          baselineSchema: branch.value.baselineSchema,
          schemaMetadata: mergedMetadata,
        }),
        updateMask
      );
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
  }
  state.isSaving = false;
  state.isEditing = false;
};

const handleMergeAfterConflictResolved = (branchName: string) => {
  state.showDiffEditor = false;
  state.isEditing = false;
  const [_, branchId] = getProjectAndBranchId(branchName);
  router.replace({
    name: "workspace.project.branch.detail",
    params: {
      projectSlug: projectV1Slug(project.value),
      branchName: branchId,
    },
  });
};

const handleSelectedDatabaseIdListChanged = async (
  databaseIdList: string[]
) => {
  let statement = "";
  try {
    const diffResponse = await branchServiceClient.diffMetadata(
      {
        sourceMetadata: branch.value.baselineSchemaMetadata,
        targetMetadata: branch.value.schemaMetadata,
        engine: branch.value.engine,
      },
      {
        silent: true,
      }
    );
    statement = diffResponse.diff;
  } catch {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("schema-editor.message.invalid-schema"),
    });
    return;
  }

  if (
    statement === "" &&
    !isEqual(
      branch.value.baselineSchemaMetadata?.schemaConfigs,
      branch.value.schemaMetadata?.schemaConfigs
    )
  ) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("schema-editor.message.cannot-change-config"),
    });
    return;
  }

  const targetDatabaseList = databaseIdList.map((id) =>
    databaseStore.getDatabaseByUID(id)
  );
  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: project.value.uid,
    mode: "normal",
    ghost: undefined,
    branch: branch.value.name,
  };
  query.databaseList = databaseIdList.join(",");
  query.sql = statement;
  query.name = generateIssueName(
    targetDatabaseList.map((db) => db.databaseName)
  );
  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
  router.push(routeInfo);
};

const generateIssueName = (databaseNameList: string[]) => {
  const issueNameParts: string[] = [];
  if (databaseNameList.length === 1) {
    issueNameParts.push(`[${databaseNameList[0]}]`);
  } else {
    issueNameParts.push(`[${databaseNameList.length} databases]`);
  }
  issueNameParts.push(`Alter schema`);
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);
  return issueNameParts.join(" ");
};

const deleteBranch = async () => {
  await branchStore.deleteBranch(branch.value.name);
  router.replace({
    name: "workspace.project.detail",
    hash: "#branches",
    params: {
      projectSlug: projectV1Slug(project.value),
    },
  });
};
</script>
