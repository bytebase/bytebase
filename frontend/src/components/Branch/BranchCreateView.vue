<template>
  <div
    class="w-full h-full flex flex-col gap-y-3 overflow-y-hidden overflow-x-auto"
  >
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-40 items-center text-sm">{{
        $t("database.branch-name")
      }}</span>
      <NInput
        v-model:value="branchTitle"
        type="text"
        class="!w-60 text-sm"
        :placeholder="'feature/add-billing'"
      />
      <span class="ml-8 mr-4 flex items-center text-sm">{{
        $t("schema-designer.parent-branch")
      }}</span>
      <BranchSelector
        class="!w-60"
        clearable
        :branch="state.parentBranchName"
        :project="state.projectId"
        @update:branch="(branch) => (state.parentBranchName = branch ?? '')"
      />
    </div>
    <NDivider class="!my-0" />
    <div class="w-full flex flex-row justify-start items-center">
      <span class="flex w-full items-center text-sm font-medium">{{
        state.parentBranchName
          ? $t("schema-designer.baseline-version-from-parent")
          : $t("schema-designer.baseline-version")
      }}</span>
    </div>
    <BaselineSchemaSelector
      :project-id="state.projectId"
      :database-id="state.baselineSchema.databaseId"
      :database-metadata="state.baselineSchema.databaseMetadata"
      :readonly="disallowToChangeBaseline"
      @update="handleBaselineSchemaChange"
    />
    <div class="w-full flex-1 overflow-y-hidden">
      <SchemaEditorV1
        :key="refreshId"
        :loading="state.loading"
        :project="project"
        :resource-type="'branch'"
        :branches="branches"
        :readonly="true"
      />
    </div>
    <div class="w-full flex items-center justify-end">
      <NButton
        type="primary"
        :disabled="!allowConfirm"
        :loading="state.isCreating"
        @click.prevent="handleConfirm"
      >
        {{ confirmText }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { uniqueId } from "lodash-es";
import { NButton, NDivider, NInput } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import SchemaEditorV1 from "@/components/SchemaEditorV1/index.vue";
import {
  pushNotification,
  useDatabaseV1Store,
  useProjectV1Store,
  useSchemaEditorV1Store,
} from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import {
  databaseNamePrefix,
  getProjectAndBranchId,
} from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { projectV1Slug } from "@/utils";
import BaselineSchemaSelector from "./BaselineSchemaSelector.vue";
import { validateBranchName } from "./utils";

interface BaselineSchema {
  // The uid of database.
  databaseId?: string;
  databaseMetadata?: DatabaseMetadata;
}

interface LocalState {
  projectId?: string;
  loading: boolean;
  baselineSchema: BaselineSchema;
  branch: Branch;
  parentBranchName?: string;
  isCreating: boolean;
}

const props = defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
});

const { t } = useI18n();
const router = useRouter();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const branchStore = useBranchStore();
const state = reactive<LocalState>({
  projectId: props.projectId,
  loading: false,
  baselineSchema: {},
  branch: Branch.fromPartial({}),
  isCreating: false,
});
const branchTitle = ref<string>("");
const refreshId = ref<string>("");

const project = computed(() => {
  const project = projectStore.getProjectByUID(state.projectId || "");
  return project;
});

const disallowToChangeBaseline = computed(() => {
  return !!state.parentBranchName;
});

// Avoid to create array or object literals in template to improve performance
const branches = computed(() => [state.branch]);

watch(
  () => [state.branch.baselineDatabase, state.branch.baselineSchema],
  () => {
    refreshId.value = uniqueId();
  }
);

watch(
  () => state.parentBranchName,
  async () => {
    if (!state.parentBranchName) {
      state.baselineSchema = {};
      state.branch = Branch.fromPartial({});
      return;
    }

    const branch = await branchStore.fetchBranchByName(
      state.parentBranchName,
      false /* !useCache */
    );
    const database = await databaseStore.getOrFetchDatabaseByName(
      branch.baselineDatabase
    );
    state.baselineSchema.databaseId = database.uid;
    state.baselineSchema.databaseMetadata = undefined;
    state.branch = branch;
    refreshId.value = uniqueId();
  }
);

const prepareSchemaDesign = async () => {
  if (
    state.baselineSchema.databaseId &&
    state.baselineSchema.databaseMetadata
  ) {
    const database = databaseStore.getDatabaseByUID(
      state.baselineSchema.databaseId
    );
    return Branch.fromPartial({
      engine: database.instanceEntity.engine,
      baselineSchemaMetadata: state.baselineSchema.databaseMetadata,
      schemaMetadata: state.baselineSchema.databaseMetadata,
    });
  }
  return Branch.fromPartial({});
};

const allowConfirm = computed(() => {
  return (
    branchTitle.value && state.baselineSchema.databaseId && !state.isCreating
  );
});

const confirmText = computed(() => {
  return t("common.create");
});

const handleBaselineSchemaChange = async (baselineSchema: BaselineSchema) => {
  if (state.parentBranchName) {
    return;
  }

  state.baselineSchema = baselineSchema;
  console.time("prepareSchemaDesign");
  state.loading = true;
  state.branch = await prepareSchemaDesign();
  state.loading = false;
  console.timeEnd("prepareSchemaDesign");
};

const handleConfirm = async () => {
  if (!state.branch) {
    return;
  }

  if (!validateBranchName(branchTitle.value)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Branch name valid characters: /^[a-zA-Z][a-zA-Z0-9-_/]+$/",
    });
    return;
  }

  const database = useDatabaseV1Store().getDatabaseByUID(
    state.baselineSchema.databaseId || ""
  );
  const schemaEditorV1Store = useSchemaEditorV1Store();
  const branchSchema = schemaEditorV1Store.resourceMap["branch"].get(
    state.branch.name
  );
  if (!branchSchema) {
    return;
  }

  state.isCreating = true;
  const baselineDatabase = `${database.instanceEntity.name}/${databaseNamePrefix}${database.databaseName}`;
  let createdSchemaDesign;
  if (!state.parentBranchName) {
    createdSchemaDesign = await branchStore.createBranch(
      project.value.name,
      branchTitle.value,
      Branch.fromPartial({
        baselineDatabase: baselineDatabase,
      })
    );
  } else {
    const parentBranch = await branchStore.fetchBranchByName(
      state.parentBranchName,
      false /* useCache */
    );
    createdSchemaDesign = await branchStore.createBranchDraft(
      project.value.name,
      branchTitle.value,
      parentBranch.name
    );
  }
  state.isCreating = false;
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.created-succeed"),
  });

  // Go to branch detail page after created.
  const [_, branchId] = getProjectAndBranchId(createdSchemaDesign.name);
  router.replace({
    name: "workspace.project.branch.detail",
    params: {
      projectSlug: projectV1Slug(project.value),
      branchName: branchId,
    },
  });
};
</script>
