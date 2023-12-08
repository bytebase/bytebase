<template>
  <div
    class="flex flex-col gap-y-3 w-full overflow-x-auto relative"
    v-bind="$attrs"
  >
    <MaskSpinner
      v-if="state.isReverting || state.savingStatus"
      class="!bg-white/75"
    >
      <div class="text-sm">
        <template v-if="state.savingStatus">
          {{ state.savingStatus }}
        </template>
      </div>
    </MaskSpinner>

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
                :loading="!!state.savingStatus"
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
        :branch="dirtyBranch"
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
    :engine="dirtyBranch.engine"
    :selected-database-id-list="[]"
    :loading="!!state.applyingToDatabaseStatus"
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
import { cloneDeep, head } from "lodash-es";
import { NButton, NDivider, NInput, NTag } from "naive-ui";
import { CSSProperties, computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import { validateDatabaseMetadata } from "@/components/SchemaEditorV1/utils";
import TargetDatabasesSelectPanel from "@/components/SyncDatabaseSchema/TargetDatabasesSelectPanel.vue";
import { pushNotification, useDatabaseV1Store } from "@/store";
import { useBranchListByProject, useBranchStore } from "@/store/modules/branch";
import {
  getProjectAndBranchId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { projectV1Slug } from "@/utils";
import { provideSQLCheckContext } from "../SQLCheck";
import { generateDiffDDL } from "../SchemaEditorLite";
import MaskSpinner from "../misc/MaskSpinner.vue";
import MergeBranchPanel from "./MergeBranchPanel.vue";
import SchemaDesignEditorLite from "./SchemaDesignEditorLite.vue";
import { validateBranchName } from "./utils";

interface LocalState {
  branchId: string;
  isEditing: boolean;
  isEditingBranchId: boolean;
  showDiffEditor: boolean;
  isReverting: boolean;
  savingStatus: string;
  applyingToDatabaseStatus: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  cleanBranch: Branch;
  dirtyBranch: Branch;
  readonly?: boolean;
}>();
const emit = defineEmits<{
  (event: "update:branch-id", id: string): void;
}>();

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const branchStore = useBranchStore();
const { branchList, ready } = useBranchListByProject(
  computed(() => props.project.name)
);
const { runSQLCheck } = provideSQLCheckContext();
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesignEditorLite>>();
const state = reactive<LocalState>({
  branchId: "",
  isEditing: false,
  isEditingBranchId: false,
  showDiffEditor: false,
  isReverting: false,
  savingStatus: "",
  applyingToDatabaseStatus: false,
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
  const branch = props.dirtyBranch;
  // Show parent branch when the current branch is a personal draft and it's not the new created one.
  if (branch.parentBranch !== "") {
    return await branchStore.fetchBranchByName(
      branch.parentBranch,
      true /* useCache */
    );
  }
  return undefined;
}, undefined);

const database = computed(() => {
  return databaseStore.getDatabaseByName(props.dirtyBranch.baselineDatabase);
});

const rebuildMetadataEdit = () => {
  const rebuild = schemaDesignerRef.value?.schemaEditor?.rebuildMetadataEdit;
  if (typeof rebuild !== "function") {
    console.warn("<SchemaEditor> ref is missing");
    return;
  }
  const branch = props.dirtyBranch;
  rebuild(
    database.value,
    branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({}),
    branch.schemaMetadata ?? DatabaseMetadata.fromPartial({})
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
  () => props.dirtyBranch.branchId,
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

  const branch = props.dirtyBranch;
  const updateMask = [];
  if (branch.branchId !== state.branchId) {
    updateMask.push("branch_id");
  }
  if (updateMask.length !== 0) {
    await branchStore.updateBranch(
      Branch.fromPartial({
        name: branch.name,
        branchId: state.branchId,
        baselineDatabase: branch.baselineDatabase,
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
  const branch = props.dirtyBranch;
  const tempList = branchList.value.filter((item) => {
    const [projectName] = getProjectAndBranchId(item.name);
    return (
      `${projectNamePrefix}${projectName}` === props.project.name &&
      item.engine === branch.engine &&
      item.name !== branch.name
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
    headBranchName: branch.name,
    branchName: branchName,
  };
  state.showDiffEditor = true;
};

const handleEdit = async () => {
  state.isEditing = true;
};

const handleCancelEdit = async () => {
  state.isReverting = true;

  Object.assign(props.dirtyBranch, cloneDeep(props.cleanBranch));

  await nextTick();
  rebuildMetadataEdit();

  state.isReverting = false;
  state.isEditing = false;
};

const handleSaveBranch = async () => {
  if (!state.isEditing) {
    return;
  }
  if (state.savingStatus) {
    return;
  }

  const applyMetadataEdit =
    schemaDesignerRef.value?.schemaEditor?.applyMetadataEdit;
  if (typeof applyMetadataEdit !== "function") {
    return;
  }
  const cleanup = async (success = false) => {
    state.savingStatus = "";
    if (success) {
      state.isEditing = false;
      await nextTick();
      rebuildMetadataEdit();
    }
  };

  const check = runSQLCheck.value;
  if (check) {
    state.savingStatus = "Checking SQL";
    if (!(await check())) {
      return cleanup();
    }
  }

  state.savingStatus = "Validating schema";
  const branch = props.dirtyBranch;
  const editing = branch.schemaMetadata
    ? cloneDeep(branch.schemaMetadata)
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

  state.savingStatus = "Saving";
  const updateMask = ["schema_metadata"];
  const updatedBranch = await branchStore.updateBranch(
    Branch.fromPartial({
      name: branch.name,
      schemaMetadata: editing,
    }),
    updateMask
  );
  Object.assign(props.dirtyBranch, updatedBranch);

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
      projectSlug: projectV1Slug(props.project),
      branchName: branchId,
    },
  });
};

const handleSelectedDatabaseIdListChanged = async (
  databaseIdList: string[]
) => {
  const cleanup = () => {
    state.applyingToDatabaseStatus = false;
  };

  state.applyingToDatabaseStatus = true;
  // Use the raw branch since the branch might be dirty by schema editor
  const branch = props.cleanBranch;

  const source =
    branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({});
  const target = branch.schemaMetadata ?? DatabaseMetadata.fromPartial({});
  const result = await generateDiffDDL(database.value, source, target);

  if (result.fatal) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: result.errors.join("\n"),
    });
    return cleanup();
  }

  const targetDatabaseList = databaseIdList.map((id) =>
    databaseStore.getDatabaseByUID(id)
  );
  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: props.project.uid,
    mode: "normal",
    ghost: undefined,
    branch: branch.name,
  };
  query.databaseList = databaseIdList.join(",");
  query.sql = result.statement;
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
  const branch = props.dirtyBranch;
  await branchStore.deleteBranch(branch.name);
  router.replace({
    name: "workspace.project.detail",
    hash: "#branches",
    params: {
      projectSlug: projectV1Slug(props.project),
    },
  });
};
</script>
