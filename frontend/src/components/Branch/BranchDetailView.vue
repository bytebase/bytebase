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
            v-if="!readonly"
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
              <NButton :loading="state.isReverting" @click="handleCancelEdit">{{
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
        <DatabaseInfo class="flex-nowrap mr-4 shrink-0" :database="database" />
      </div>
    </div>

    <div class="w-full h-[32rem]">
      <SchemaDesignEditorLite
        ref="schemaDesignerRef"
        :project="project"
        :readonly="!state.isEditing"
        :branch="branch"
      />
    </div>
    <!-- Don't show delete button in view mode. -->
    <div v-if="!readonly">
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
    :project="project"
    :head-branch-name="mergeBranchPanelContext.headBranchName"
    :branch-name="mergeBranchPanelContext.branchName"
    @dismiss="state.showDiffEditor = false"
    @merged="handleMergeAfterConflictResolved"
  />
</template>

<script lang="ts" setup>
import { asyncComputed } from "@vueuse/core";
import dayjs from "dayjs";
import { cloneDeep, head, isEqual } from "lodash-es";
import { NButton, NDivider, NInput, NTag } from "naive-ui";
import { CSSProperties, computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import { validateDatabaseMetadata } from "@/components/SchemaEditorV1/utils";
import TargetDatabasesSelectPanel from "@/components/SyncDatabaseSchema/TargetDatabasesSelectPanel.vue";
import { branchServiceClient } from "@/grpcweb";
import { pushNotification, useDatabaseV1Store } from "@/store";
import { useBranchList, useBranchStore } from "@/store/modules/branch";
import {
  getProjectAndBranchId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { projectV1Slug } from "@/utils";
import { provideSQLCheckContext } from "../SQLCheck";
import MergeBranchPanel from "./MergeBranchPanel.vue";
import SchemaDesignEditorLite from "./SchemaDesignEditorLite.vue";
import { validateBranchName } from "./utils";

interface LocalState {
  branchId: string;
  isEditing: boolean;
  isEditingBranchId: boolean;
  showDiffEditor: boolean;
  isReverting: boolean;
  isSaving: boolean;
}

const props = defineProps<{
  projectId: string;
  branch: Branch;
  readonly?: boolean;
}>();
const emit = defineEmits<{
  (event: "update:branch", branch: Branch): void;
  (event: "update:branch-id", id: string): void;
}>();

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const branchStore = useBranchStore();
const { branchList, ready } = useBranchList(props.projectId);
const { runSQLCheck } = provideSQLCheckContext();
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesignEditorLite>>();
const state = reactive<LocalState>({
  branchId: "",
  isEditing: false,
  isEditingBranchId: false,
  showDiffEditor: false,
  isReverting: false,
  isSaving: false,
});
const mergeBranchPanelContext = ref<{
  headBranchName: string;
  branchName: string;
}>();
const selectTargetDatabasesContext = ref<{
  show: boolean;
}>({
  show: false,
});

const parentBranch = asyncComputed(async () => {
  // Show parent branch when the current branch is a personal draft and it's not the new created one.
  if (props.branch.parentBranch !== "") {
    return await branchStore.fetchBranchByName(
      props.branch.parentBranch,
      true /* useCache */
    );
  }
  return undefined;
}, undefined);

const database = computed(() => {
  return databaseStore.getDatabaseByName(props.branch.baselineDatabase);
});
const project = computed(() => {
  return database.value.projectEntity;
});

const rebuildMetadataEdit = () => {
  const schemaDesigner = schemaDesignerRef.value;
  schemaDesigner?.schemaEditor?.rebuildMetadataEdit(
    database.value,
    props.branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({}),
    props.branch.schemaMetadata ?? DatabaseMetadata.fromPartial({})
  );
};

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

watch(
  () => props.branch.branchId,
  (title) => {
    state.branchId = title;
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
  if (props.branch.branchId !== state.branchId) {
    updateMask.push("branch_id");
  }
  if (updateMask.length !== 0) {
    await branchStore.updateBranch(
      Branch.fromPartial({
        name: props.branch.name,
        branchId: state.branchId,
        baselineDatabase: props.branch.baselineDatabase,
      }),
      updateMask
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
  }
  emit("update:branch-id", state.branchId);
  state.isEditingBranchId = false;
};

const handleMergeBranch = () => {
  const tempList = branchList.value.filter((item) => {
    const [projectName] = getProjectAndBranchId(item.name);
    return (
      `${projectNamePrefix}${projectName}` === project.value.name &&
      item.engine === props.branch.engine &&
      item.name !== props.branch.name
    );
  });
  const branchName = parentBranch.value
    ? parentBranch.value.name
    : head(tempList)?.name;
  if (!branchName) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "No branch to merge.",
    });
    return;
  }

  mergeBranchPanelContext.value = {
    headBranchName: props.branch.name,
    branchName: branchName,
  };
  state.showDiffEditor = true;
};

const handleEdit = async () => {
  state.isEditing = true;
};

const handleCancelEdit = async () => {
  state.isReverting = true;
  const originalBranch = await branchStore.fetchBranchByName(
    props.branch.name,
    /* !useCache */ false
  );
  emit("update:branch", originalBranch);
  state.isReverting = false;
  state.isEditing = false;
  await nextTick();
  rebuildMetadataEdit();
};

const handleSaveBranch = async () => {
  if (!state.isEditing) {
    return;
  }
  if (state.isSaving) {
    return;
  }
  const cleanup = async (success = false) => {
    state.isSaving = false;
    if (success) {
      state.isEditing = false;
      await nextTick();
      rebuildMetadataEdit();
    }
  };

  state.isSaving = true;

  const applyMetadataEdit =
    schemaDesignerRef.value?.schemaEditor?.applyMetadataEdit;
  if (typeof applyMetadataEdit !== "function") {
    return cleanup();
  }

  const check = runSQLCheck.value;
  if (check && !(await check())) {
    return cleanup();
  }
  const editing = props.branch.schemaMetadata
    ? cloneDeep(props.branch.schemaMetadata)
    : DatabaseMetadata.fromPartial({});
  await applyMetadataEdit(database.value, editing);

  const validationMessages = validateDatabaseMetadata(editing);
  if (validationMessages.length > 0) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Invalid schema design",
      description: validationMessages.join("\n"),
    });
    return cleanup();
  }
  const updateMask = ["schema_metadata"];
  await branchStore.updateBranch(
    Branch.fromPartial({
      name: props.branch.name,
      schemaMetadata: editing,
    }),
    updateMask
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.updated-succeed"),
  });
  cleanup(/* success */ true);
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
        sourceMetadata: props.branch.baselineSchemaMetadata,
        targetMetadata: props.branch.schemaMetadata,
        engine: props.branch.engine,
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
      props.branch.baselineSchemaMetadata?.schemaConfigs,
      props.branch.schemaMetadata?.schemaConfigs
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
    branch: props.branch.name,
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
  await branchStore.deleteBranch(props.branch.name);
  router.replace({
    name: "workspace.project.detail",
    hash: "#branches",
    params: {
      projectSlug: projectV1Slug(project.value),
    },
  });
};
</script>
