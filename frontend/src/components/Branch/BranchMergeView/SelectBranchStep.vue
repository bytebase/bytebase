<template>
  <div
    class="w-full grid grid-cols-4 items-center text-sm gap-x-2"
    style="grid-template-columns: 1fr auto 1fr auto"
  >
    <BranchSelector
      class="!w-full text-center"
      :clearable="false"
      :project="project"
      :branch="targetBranch?.name"
      :filter="targetBranchFilter"
      @update:branch="$emit('update:target-branch-name', $event)"
    />
    <div class="flex flex-row justify-center px-2">
      <MoveLeftIcon :size="40" stroke-width="1" />
    </div>
    <BranchSelector
      class="!full text-center"
      :clearable="false"
      :project="project"
      :branch="headBranch?.name"
      :filter="headBranchFilter"
      @update:branch="$emit('update:head-branch-name', $event)"
    />
    <NCheckbox
      :checked="deleteBranchAfterMerged"
      @update:checked="
        $emit('update:delete-branch-after-merged', $event as boolean)
      "
    >
      {{ $t("branch.merge-rebase.delete-branch-after-merged") }}
    </NCheckbox>
  </div>
  <div class="w-full flex-1 flex flex-col relative text-sm gap-y-1">
    <div class="flex flex-row items-center gap-x-1">
      <span v-if="!headBranch || !targetBranch">
        {{ $t("branch.merge-rebase.select-branches-to-merge") }}
      </span>
      <template v-else>
        <template v-if="isValidating">
          <BBSpin class="!w-4 !h-4" />
          <span>{{ $t("branch.merge-rebase.validating-branch") }}</span>
        </template>
        <template v-else-if="validationState">
          <template v-if="validationState.status === Status.OK">
            <CheckIcon class="w-4 h-4 text-success" />
            <span>{{ $t("branch.merge-rebase.able-to-merge") }}</span>
          </template>
          <template v-else>
            <XCircleIcon class="w-4 h-4 text-error" />
            <i18n-t
              keypath="branch.merge-rebase.cannot-automatically-merge"
              tag="span"
              class="text-error"
            >
              <template #rebase_branch>
                <router-link
                  :to="rebaseLink(headBranch, targetBranch)"
                  class="normal-link"
                >
                  {{ $t("branch.merge-rebase.go-rebase") }}
                </router-link>
              </template>
            </i18n-t>
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

      <MaskSpinner v-if="isLoadingHeadBranch || isLoadingTargetBranch" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { CheckIcon, XCircleIcon, MoveLeftIcon } from "lucide-vue-next";
import { NCheckbox, NTab, NTabs } from "naive-ui";
import { Status } from "nice-grpc-common";
import { ref, watch } from "vue";
import { DiffEditor } from "@/components/MonacoEditor";
import SchemaEditorLite from "@/components/SchemaEditorLite";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { useDatabaseV1Store } from "@/store";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";
import { projectSlugV1 } from "@/utils";
import { MergeBranchValidationState } from "./types";

type TabValue = "schema-editor" | "raw-schema-text";

const props = defineProps<{
  project: ComposedProject;
  targetBranch: Branch | undefined;
  headBranch: Branch | undefined;
  isLoadingTargetBranch?: boolean;
  isLoadingHeadBranch?: boolean;
  isValidating?: boolean;
  validationState: MergeBranchValidationState | undefined;
  deleteBranchAfterMerged: boolean;
}>();

defineEmits<{
  (event: "update:head-branch-name", branch: string | undefined): void;
  (event: "update:target-branch-name", branch: string | undefined): void;
  (event: "update:delete-branch-after-merged", on: boolean): void;
}>();

const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const tab = ref<TabValue>("schema-editor");

const targetBranchFilter = (branch: Branch) => {
  const { headBranch } = props;
  if (!headBranch) {
    return true;
  }
  return branch.engine === headBranch.engine && branch.name !== headBranch.name;
};
const headBranchFilter = (branch: Branch) => {
  const { targetBranch } = props;
  if (!targetBranch) {
    return true;
  }
  return (
    branch.engine === targetBranch.engine && branch.name !== targetBranch.name
  );
};

const rebaseLink = (headBranch: Branch, targetBranch: Branch) => {
  return {
    name: "workspace.project.branch.rebase",
    params: {
      projectSlug: projectSlugV1(props.project),
      branchName: headBranch.branchId,
    },
    query: {
      source: targetBranch.branchId,
    },
  };
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
