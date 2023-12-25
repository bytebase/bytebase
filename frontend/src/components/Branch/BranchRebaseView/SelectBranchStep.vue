<template>
  <div
    class="w-full grid grid-cols-3 items-end text-sm gap-x-2"
    style="grid-template-columns: 1fr auto 1fr"
  >
    <BranchSelector
      class="!full text-center"
      :clearable="false"
      :project="project"
      :branch="headBranch?.name"
      :filter="headBranchFilter"
      @update:branch="$emit('update:head-branch-name', $event)"
    />
    <div class="flex flex-row justify-center px-2 h-[34px]">
      <MoveLeftIcon :size="40" stroke-width="1" />
    </div>
    <div class="flex flex-col">
      <NRadioGroup
        :value="sourceType"
        class="space-x-2"
        @update:value="$emit('update:source-type', $event as RebaseSourceType)"
      >
        <NRadio value="BRANCH">分支</NRadio>
        <NRadio value="DATABASE">数据库</NRadio>
      </NRadioGroup>
      <BranchSelector
        v-if="sourceType === 'BRANCH'"
        class="!w-full text-center"
        :clearable="false"
        :project="project"
        :branch="sourceBranch?.name"
        :filter="sourceBranchFilter"
        @update:branch="$emit('update:source-branch-name', $event)"
      />
      <DatabaseSelect
        v-if="sourceType === 'DATABASE'"
        :database="sourceDatabase?.uid"
        :project="project.uid"
        :allowed-engine-type-list="headBranch ? [headBranch.engine] : undefined"
        style="width: 100%"
      />
    </div>
  </div>
  <div class="w-full flex-1 flex flex-col relative text-sm gap-y-1">
    <div class="flex flex-row items-center gap-x-1">
      <span v-if="!headBranch || !sourceBranch">
        {{ $t("branch.merge-rebase.select-branches-to-rebase") }}
      </span>
      <template v-else>
        <template v-if="isValidating">
          <BBSpin class="!w-4 !h-4" />
          <span>{{ $t("branch.merge-rebase.validating-branch") }}</span>
        </template>
        <template v-else-if="validationState">
          <template v-if="validationState.branch">
            <CheckIcon class="w-4 h-4 text-success" />
            <span>{{ $t("branch.merge-rebase.able-to-rebase") }}</span>
          </template>
          <template v-else>
            <XCircleIcon class="w-4 h-4 text-error" />
            <span>
              {{ $t("branch.merge-rebase.cannot-automatically-rebase") }}
            </span>
          </template>
        </template>
      </template>
    </div>
    <NTabs v-model:value="tab">
      <template v-if="headBranch" #prefix>
        <div class="text-control-placeholder">
          {{
            $t("branch.merge-rebase.changes-of-branch", {
              branch: headBranch.branchId,
            })
          }}
        </div>
      </template>
      <NTab name="schema-editor">
        {{ $t("schema-editor.self") }}
      </NTab>
      <NTab name="raw-schema-text">
        {{ $t("schema-editor.raw-sql") }}
      </NTab>
    </NTabs>
    <div
      v-show="tab === 'schema-editor'"
      class="w-full flex-1 relative text-sm"
    >
      <SchemaEditorLite
        ref="schemaEditorRef"
        :key="headBranch?.name"
        resource-type="branch"
        :project="project"
        :readonly="true"
        :loading="isLoadingHeadBranch"
        :branch="headBranch ?? Branch.fromPartial({})"
      />
      <MaskSpinner v-if="isLoadingHeadBranch" />
    </div>
    <div
      v-show="tab === 'raw-schema-text'"
      class="w-full flex-1 relative text-sm border rounded overflow-clip"
    >
      <DiffEditor
        :readonly="true"
        :original="headBranch?.baselineSchema"
        :modified="headBranch?.schema"
        class="h-full"
      />

      <MaskSpinner v-if="isLoadingHeadBranch || isLoadingSourceBranch" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { CheckIcon, XCircleIcon, MoveLeftIcon } from "lucide-vue-next";
import { NRadioGroup, NTab, NTabs } from "naive-ui";
import { ref, watch } from "vue";
import { DiffEditor } from "@/components/MonacoEditor";
import SchemaEditorLite from "@/components/SchemaEditorLite";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { DatabaseSelect } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { ComposedDatabase, ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { RebaseBranchValidationState, RebaseSourceType } from "./types";

type TabValue = "schema-editor" | "raw-schema-text";

const props = defineProps<{
  project: ComposedProject;
  sourceType: RebaseSourceType;
  headBranch: Branch | undefined;
  sourceBranch: Branch | undefined;
  sourceDatabase: ComposedDatabase | undefined;
  isLoadingSourceBranch?: boolean;
  isLoadingHeadBranch?: boolean;
  isValidating?: boolean;
  validationState: RebaseBranchValidationState | undefined;
}>();

defineEmits<{
  (event: "update:source-type", type: RebaseSourceType): void;
  (event: "update:head-branch-name", branch: string | undefined): void;
  (event: "update:source-branch-name", branch: string | undefined): void;
  (event: "update:source-database-uid", uid: string): void;
}>();

const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const tab = ref<TabValue>("schema-editor");

const sourceBranchFilter = (branch: Branch) => {
  const { headBranch } = props;
  if (!headBranch) {
    return true;
  }
  return branch.engine === headBranch.engine && branch.name !== headBranch.name;
};
const headBranchFilter = (branch: Branch) => {
  const { sourceBranch } = props;
  if (!sourceBranch) {
    return true;
  }
  return (
    branch.engine === sourceBranch.engine && branch.name !== sourceBranch.name
  );
};

// re-calculate diff for coloring when branch changed
watch(
  [() => props.headBranch, schemaEditorRef],
  ([head, editor]) => {
    if (head && editor) {
      const db = useDatabaseV1Store().getDatabaseByName(head.baselineDatabase);
      editor.rebuildMetadataEdit(
        db,
        head.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({}),
        head.schemaMetadata ?? DatabaseMetadata.fromPartial({})
      );
    }
  },
  {
    immediate: true,
  }
);
</script>
