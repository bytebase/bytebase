<template>
  <div class="flex flex-col gap-y-3 w-full h-full relative" v-bind="$attrs">
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

    <div class="w-full flex flex-row justify-between items-center gap-2">
      <div class="flex flex-row justify-start items-center gap-x-2">
        <span class="text-xl leading-[34px]">{{ cleanBranch.branchId }}</span>
        <span
          v-if="parentBranch"
          class="group text-sm border rounded-full px-2 py-1 cursor-pointer hover:bg-gray-100"
          @click="handleParentBranchClick"
        >
          <span class="text-gray-500 mr-1"
            >{{ $t("schema-designer.parent-branch") }}:</span
          >
          <span class="group-hover:underline group-hover:text-blue-600">{{
            parentBranch.branchId
          }}</span>
        </span>
      </div>
      <div class="shrink-0 flex flex-row justify-between items-center gap-2">
        <template v-if="!state.isEditing">
          <NButton v-if="showEditButton" @click="handleEdit">{{
            $t("common.edit")
          }}</NButton>
          <NButton v-if="showMergeBranchButton" @click="handleGotoMergeBranch">
            {{ $t("branch.merge-rebase.merge-branch") }}
          </NButton>
          <NButton
            v-if="showRebaseBranchButton"
            @click="handleGotoRebaseBranch"
          >
            {{ $t("branch.merge-rebase.rebase-branch") }}
          </NButton>
          <NButton
            v-if="showApplyBranchButton"
            type="primary"
            @click="handleApplyBranchToDatabase"
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

    <NDivider class="!my-0" />

    <div
      class="w-full flex flex-row justify-between items-center text-sm gap-4 h-[32px]"
    >
      <div
        class="flex-1 flex flex-row justify-start items-center opacity-80 whitespace-nowrap"
      >
        <span class="mr-4">{{ $t("schema-designer.baseline-version") }}:</span>
        <DatabaseInfo class="flex-nowrap" :database="database" />
      </div>
      <div
        v-if="!state.isEditing"
        class="flex flex-row justify-end items-center gap-x-1 whitespace-nowrap"
      >
        <NCheckbox v-model:checked="state.showDiff">
          {{ $t("branch.show-diff-with-branch-baseline") }}
        </NCheckbox>
      </div>
    </div>

    <div class="w-full flex-1 flex flex-col overflow-hidden">
      <SchemaDesignEditorLite
        ref="schemaDesignerRef"
        :project="project"
        :readonly="!state.isEditing"
        :branch="dirtyBranch"
        :disable-diff-coloring="!state.isEditing && !state.showDiff"
        @update-is-editing="updateIsEditing"
      />
    </div>
    <!-- Don't show delete button in view mode. -->
    <div v-if="allowDelete">
      <BBButtonConfirm
        :type="'DELETE'"
        :button-text="$t('database.delete-this-branch')"
        :require-confirm="true"
        @confirm="deleteBranch(false)"
      />
    </div>
  </div>

  <TargetDatabasesSelectPanel
    v-if="selectTargetDatabasesContext.show"
    :project="project.name"
    :engine="dirtyBranch.engine"
    :loading="!!state.applyingToDatabaseStatus"
    @close="selectTargetDatabasesContext.show = false"
    @update="handleApplyToDatabase"
  />
</template>

<script lang="ts" setup>
import { asyncComputed, computedAsync } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { NButton, NCheckbox, NDivider, useDialog } from "naive-ui";
import { Status } from "nice-grpc-common";
import { v4 as uuidv4 } from "uuid";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBButtonConfirm } from "@/bbkit";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import { validateDatabaseMetadata } from "@/components/SchemaEditorLite";
import TargetDatabasesSelectPanel from "@/components/SyncDatabaseSchema/TargetDatabasesSelectPanel.vue";
import {
  PROJECT_V1_ROUTE_BRANCHES,
  PROJECT_V1_ROUTE_BRANCH_DETAIL,
  PROJECT_V1_ROUTE_BRANCH_MERGE,
  PROJECT_V1_ROUTE_BRANCH_ROLLOUT,
  PROJECT_V1_ROUTE_BRANCH_REBASE,
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  extractUserEmail,
  pushNotification,
  useDatabaseV1Store,
  useCurrentUserV1,
} from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import { getProjectAndBranchId } from "@/store/modules/v1/common";
import {
  unknownDatabase,
  type ComposedProject,
  type Permission,
} from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import {
  defer,
  extractProjectResourceName,
  generateIssueTitle,
  hasProjectPermissionV2,
} from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";
import { provideSQLCheckContext } from "../SQLCheck";
import { generateDiffDDL } from "../SchemaEditorLite";
import MaskSpinner from "../misc/MaskSpinner.vue";
import SchemaDesignEditorLite from "./SchemaDesignEditorLite.vue";

interface LocalState {
  branchId: string;
  showDiff: boolean;
  isEditing: boolean;
  isEditingBranchId: boolean;
  isReverting: boolean;
  savingStatus: string;
  applyingToDatabaseStatus: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  cleanBranch: Branch;
  dirtyBranch: Branch;
}>();

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const branchStore = useBranchStore();
const { runSQLCheck } = provideSQLCheckContext();
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesignEditorLite>>();
const state = reactive<LocalState>({
  branchId: "",
  // default true to child branches, default false to main branches
  showDiff: props.dirtyBranch.parentBranch ? true : false,
  isEditing: false,
  isEditingBranchId: false,
  isReverting: false,
  savingStatus: "",
  applyingToDatabaseStatus: false,
});
const $dialog = useDialog();
const selectTargetDatabasesContext = ref<{
  show: boolean;
}>({
  show: false,
});

const currentUser = useCurrentUserV1();

const checkPermission = (permission: Permission): boolean => {
  return (
    hasProjectPermissionV2(props.project, permission) ||
    extractUserEmail(props.cleanBranch.creator) === currentUser.value.email
  );
};

const allowDelete = computed(() => {
  if (props.dirtyBranch.parentBranch === "") {
    return checkPermission("bb.branches.admin");
  }
  return checkPermission("bb.branches.delete");
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

const database = computedAsync(() => {
  return databaseStore.getOrFetchDatabaseByName(
    props.dirtyBranch.baselineDatabase
  );
}, unknownDatabase());

const showEditButton = computed(() => {
  if (checkPermission("bb.branches.admin")) {
    return true;
  }
  // main branches (parent-less branches) cannot be updated.
  if (!parentBranch.value) {
    return false;
  }
  return true;
});

const showMergeBranchButton = computed(() => {
  // main branches (parent-less branches) cannot be merged.
  if (!parentBranch.value) {
    return false;
  }
  return checkPermission("bb.branches.update");
});

const showRebaseBranchButton = computed(() => {
  // For main branches: only project owners are allowed
  if (!parentBranch.value) {
    return hasProjectPermissionV2(props.project, "bb.branches.admin");
  }

  return checkPermission("bb.branches.update");
});

const showApplyBranchButton = computed(() => {
  // only main branches can be applied to databases.
  return !parentBranch.value;
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

watch(
  () => props.dirtyBranch.branchId,
  (title) => {
    state.branchId = title;
  },
  {
    immediate: true,
  }
);

const handleParentBranchClick = async () => {
  if (!parentBranch.value) {
    return;
  }

  const [_, branchId] = getProjectAndBranchId(parentBranch.value.name);
  router.push({
    name: PROJECT_V1_ROUTE_BRANCH_DETAIL,
    params: {
      branchName: `${branchId}`,
    },
  });
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
    // TODO: optimize: check() could return the generated DDL to avoid
    // generating one more time below. useful for large schemas
  }

  state.savingStatus = "Validating schema";
  const branch = props.dirtyBranch;
  const editing = branch.schemaMetadata
    ? cloneDeep(branch.schemaMetadata)
    : DatabaseMetadata.fromPartial({});
  applyMetadataEdit(database.value, editing);

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
  Object.assign(props.cleanBranch, updatedBranch);
  Object.assign(props.dirtyBranch, cloneDeep(updatedBranch));

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.updated-succeed"),
  });
  cleanup(/* success */ true);
};

const handleGotoMergeBranch = () => {
  router.push({
    name: PROJECT_V1_ROUTE_BRANCH_MERGE,
    params: {
      branchName: props.cleanBranch.branchId,
    },
  });
};
const handleGotoRebaseBranch = () => {
  router.push({
    name: PROJECT_V1_ROUTE_BRANCH_REBASE,
    params: {
      branchName: props.cleanBranch.branchId,
    },
  });
};

const handleApplyBranchToDatabase = () => {
  router.push({
    name: PROJECT_V1_ROUTE_BRANCH_ROLLOUT,
    params: {
      branchName: props.cleanBranch.branchId,
    },
  });
};

const handleApplyToDatabase = async (databaseNameList: string[]) => {
  const cleanup = () => {
    state.applyingToDatabaseStatus = false;
  };

  state.applyingToDatabaseStatus = true;
  // Use the raw branch since the branch might be dirty by schema editor
  const branch = props.cleanBranch;

  const source =
    branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({});
  const target = branch.schemaMetadata ?? DatabaseMetadata.fromPartial({});
  const result = await generateDiffDDL(
    database.value,
    source,
    target,
    /* !allowEmptyDiffDDLWithConfigChange */ false
  );

  if (result.fatal) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: result.errors.join("\n"),
    });
    return cleanup();
  }

  const targetDatabaseList = databaseNameList.map((name) =>
    databaseStore.getDatabaseByName(name)
  );
  const sqlStorageKey = `bb.issues.sql.${uuidv4()}`;
  localStorage.setItem(sqlStorageKey, result.statement);
  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    mode: "normal",
    ghost: undefined,
    branch: branch.name,
    sqlStorageKey,
  };
  query.databaseList = targetDatabaseList.map((db) => db.name).join(",");
  query.name = generateIssueTitle(
    "bb.issue.database.schema.update",
    targetDatabaseList.map((db) => db.databaseName)
  );
  const routeInfo = {
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(props.project.name),
      issueSlug: "create",
    },
    query,
  };
  router.push(routeInfo);
};

const confirmForceDeleteBranch = () => {
  const d = defer<boolean>();
  $dialog.warning({
    title: t("common.warning"),
    content: t("branch.deleting-parent-branch"),
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: t("branch.force-delete"),
    onNegativeClick: () => {
      d.resolve(false);
    },
    onPositiveClick: () => {
      d.resolve(true);
    },
  });
  return d.promise;
};

const deleteBranch = async (force: boolean) => {
  const branch = props.dirtyBranch;
  try {
    await branchStore.deleteBranch(branch.name, force);
    router.replace({
      name: PROJECT_V1_ROUTE_BRANCHES,
    });
  } catch (err) {
    if (getErrorCode(err) === Status.FAILED_PRECONDITION) {
      const confirmed = await confirmForceDeleteBranch();
      if (confirmed) {
        deleteBranch(true);
      }
    }
  }
};

const updateIsEditing = (newValue: boolean) => {
  state.isEditing = newValue;
};
</script>
