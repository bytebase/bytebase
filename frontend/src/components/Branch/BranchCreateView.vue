<template>
  <div
    class="w-full h-full flex flex-col gap-y-3 relative overflow-y-hidden overflow-x-auto"
  >
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center text-sm">{{
        $t("database.branch-name")
      }}</span>
      <NInput
        v-model:value="branchId"
        type="text"
        class="!w-60 text-sm"
        :placeholder="'feature/add-billing'"
      />
      <span class="ml-8 mr-4 flex items-center text-sm">{{
        $t("schema-designer.parent-branch")
      }}</span>
      <BranchSelector
        v-model:branch="parentBranchName"
        :project="projectId"
        class="!w-60"
        clearable
      />
    </div>
    <NDivider class="!my-0" />
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-full items-center text-sm font-medium">{{
        parentBranchName
          ? $t("schema-designer.baseline-version-from-parent")
          : $t("schema-designer.baseline-version")
      }}</span>
    </div>
    <BaselineSchemaSelector
      v-model:database-id="databaseId"
      :project-id="projectId"
      :readonly="disallowToChangeBaseline"
    />
    <div class="w-full">
      <div>isPreparingBranch: {{ isPreparingBranch }}</div>
      <div>databaseId: {{ databaseId }}</div>
      <div>parentBranchName: {{ parentBranchName }}</div>
      <div>state.parent: {{ state?.parent }}</div>
      <div>state.branch.name: {{ state?.branch.name }}</div>
      <div>
        len(tables):
        {{ state?.branch.schemaMetadata?.schemas.map((s) => s.tables.length) }}
      </div>
      <div>
        size(state.branch):
        <template v-if="state?.branch">{{
          bytesToString(JSON.stringify(state.branch).length)
        }}</template>
      </div>
    </div>
    <div class="w-full flex-1 overflow-y-hidden">
      <SchemaEditorLite
        v-if="state?.branch"
        :key="state.branch.name"
        :loading="isPreparingBranch"
        :project="project"
        :resource-type="'branch'"
        :branch="state.branch"
        :readonly="true"
      />
    </div>
    <div class="w-full flex items-center justify-end">
      <NButton
        type="primary"
        :disabled="!allowConfirm"
        :loading="isCreating"
        @click.prevent="handleConfirm"
      >
        {{ confirmText }}
      </NButton>
    </div>

    <MaskSpinner v-show="isPreparingBranch" />
  </div>
</template>

<script lang="ts" setup>
import { useDebounce } from "@vueuse/core";
import { cloneDeep, uniqueId } from "lodash-es";
import { NButton, NDivider, NInput } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import SchemaEditorLite from "@/components/SchemaEditorLite";
import {
  pushNotification,
  useDBSchemaV1Store,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import { UNKNOWN_ID } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { bytesToString, projectV1Slug } from "@/utils";
import MaskSpinner from "../misc/MaskSpinner.vue";
import BaselineSchemaSelector from "./BaselineSchemaSelector.vue";
import { validateBranchName } from "./utils";

type BranchPrepareState = {
  branch: Branch;
  parent: string | undefined;
};

const props = defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
});

const DEBOUNCE_RATE = 100;
const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const projectStore = useProjectV1Store();
const branchStore = useBranchStore();
const projectId = ref(props.projectId);
const databaseId = ref<string>();
const parentBranchName = ref<string>();
const isCreating = ref(false);
const branchId = ref<string>("");
const isPreparingBranch = ref(false);

const project = computed(() => {
  const project = projectStore.getProjectByUID(projectId.value || "");
  return project;
});

const debouncedDatabaseId = useDebounce(databaseId, DEBOUNCE_RATE);
const debouncedParentBranchName = useDebounce(parentBranchName, DEBOUNCE_RATE);
const disallowToChangeBaseline = computed(() => {
  return !!parentBranchName.value;
});

const nextFakeBranchName = () => {
  return `${project.value.name}/branches/-${uniqueId()}`;
};

const prepareBranchFromParentBranch = async (parent: string) => {
  const tag = `prepareBranchFromParentBranch(${parent})`;
  console.time(tag);
  const parentBranch = await branchStore.fetchBranchByName(
    parent,
    false /* !useCache */
  );
  const branch = cloneDeep(parentBranch);
  branch.name = nextFakeBranchName();
  console.timeEnd(tag);
  return branch;
};
const prepareBranchFromDatabaseHead = async (uid: string) => {
  const tag = `prepareBranchFromDatabaseHead(${uid})`;
  console.log(tag);
  console.time(tag);

  console.time("--fetch metadata");
  const database = databaseStore.getDatabaseByUID(uid);
  const metadata = await dbSchemaStore.getOrFetchDatabaseMetadata({
    database: database.name,
    skipCache: false,
    view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
  });
  console.timeEnd("--fetch metadata");

  console.time("--build branch object");
  // Here metadata is not used for editing, so we need not to clone a copy
  // for baseline
  const branch = Branch.fromPartial({
    name: nextFakeBranchName(),
    engine: database.instanceEntity.engine,
    baselineDatabase: database.name,
    baselineSchemaMetadata: metadata,
    schemaMetadata: metadata,
  });
  console.timeEnd("--build branch object");

  console.timeEnd(tag);
  return branch;
};

const state = ref<BranchPrepareState>();
const prepareBranch = async (
  _parentBranchName: string | undefined,
  _databaseId: string | undefined
) => {
  isPreparingBranch.value = true;

  const finish = (s: BranchPrepareState | undefined) => {
    const isOutdated =
      _parentBranchName !== parentBranchName.value ||
      _databaseId !== databaseId.value;
    if (isOutdated) {
      return;
    }

    state.value = s;
    isPreparingBranch.value = false;
  };

  if (_parentBranchName) {
    const branch = await prepareBranchFromParentBranch(_parentBranchName);
    return finish({
      branch,
      parent: _parentBranchName,
    });
  }
  if (_databaseId && _databaseId !== String(UNKNOWN_ID)) {
    const branch = await prepareBranchFromDatabaseHead(_databaseId);
    return finish({
      branch,
      parent: undefined,
    });
  }
  return finish(undefined);
};

watch(
  [debouncedParentBranchName, debouncedDatabaseId],
  ([parentBranchName, databaseId]) => {
    prepareBranch(parentBranchName, databaseId);
  }
);

const allowConfirm = computed(() => {
  return branchId.value && state.value && !isCreating.value;
});

const confirmText = computed(() => {
  return t("common.create");
});

const handleConfirm = async () => {
  if (!state.value) {
    return;
  }

  if (!validateBranchName(branchId.value)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Branch name valid characters: /^[a-zA-Z][a-zA-Z0-9-_/]+$/",
    });
    return;
  }

  const { branch, parent } = state.value;

  const { baselineDatabase } = branch;
  isCreating.value = true;
  if (!parent) {
    await branchStore.createBranch(
      project.value.name,
      branchId.value,
      Branch.fromPartial({
        baselineDatabase,
      })
    );
  } else {
    await branchStore.createBranch(
      project.value.name,
      branchId.value,
      Branch.fromPartial({
        parentBranch: parent!,
      })
    );
  }
  isCreating.value = false;
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.created-succeed"),
  });

  // Go to branch detail page after created.
  router.replace({
    name: "workspace.project.branch.detail",
    params: {
      projectSlug: projectV1Slug(project.value),
      branchName: branchId.value,
    },
  });
};
</script>
