<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && emit('dismiss')"
  >
    <DrawerContent :title="$t('schema-designer.merge-branch')" :closable="true">
      <div
        class="space-y-3 w-[calc(100vw-24rem)] min-w-[64rem] max-w-[calc(100vw-8rem)] h-full overflow-x-auto"
      >
        <div class="w-full flex flex-row justify-end items-center gap-2">
          <NButton type="primary" @click="handleMergeBranch">
            {{ $t("common.merge") }}
          </NButton>
        </div>
        <div class="w-full flex flex-row justify-end items-center gap-2">
          <NCheckbox v-model:checked="state.deleteBranchAfterMerged">
            {{ $t("schema-designer.delete-branch-after-merged") }}
          </NCheckbox>
        </div>
        <div class="w-full pr-12 pt-4 pb-6 grid grid-cols-3">
          <div class="flex flex-row justify-end">
            <BranchSelector
              class="!w-4/5 text-center"
              :clearable="false"
              :project="getProjectName(project.name)"
              :branch="state.branchName"
              :filter="targetBranchFilter"
              @update:branch="(branch) => (state.branchName = branch ?? '')"
            />
          </div>
          <div class="flex flex-row justify-center">
            <MoveLeft :size="40" stroke-width="1" />
          </div>
          <div class="flex flex-row justify-start">
            <NInput
              class="!w-4/5 text-center"
              readonly
              :value="sourceBranch.branchId"
              size="large"
            />
          </div>
        </div>
        <div class="w-full grid grid-cols-2">
          <div class="col-span-1">
            <span>base</span>
          </div>
          <div class="col-span-1">
            <span>compare</span>
          </div>
        </div>
        <div class="w-full h-[calc(100%-14rem)] border relative">
          <DiffEditor
            v-if="ready"
            :key="state.branchName"
            class="h-full"
            :original="targetBranch.schema"
            :modified="state.editingSchema"
            @update:modified="state.editingSchema = $event"
          />
          <div
            v-else
            class="inset-0 absolute flex flex-col justify-center items-center"
          >
            <BBSpin />
          </div>
        </div>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { asyncComputed } from "@vueuse/core";
import { MoveLeft } from "lucide-vue-next";
import { NButton, NInput, NCheckbox, useDialog } from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import { pushNotification } from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import { getProjectName } from "@/store/modules/v1/common";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DiffEditor } from "../MonacoEditor";

interface LocalState {
  branchName: string;
  editingSchema: string;
  deleteBranchAfterMerged: boolean;
}

const props = defineProps<{
  project: ComposedProject;
  headBranchName: string;
  branchName: string;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "merged", branchName: string): void;
}>();

const state = reactive<LocalState>({
  branchName: "",
  editingSchema: "",
  deleteBranchAfterMerged: false,
});
const { t } = useI18n();
const dialog = useDialog();
const branchStore = useBranchStore();
const isLoadingSourceBranch = ref(false);
const isLoadingTargetBranch = ref(false);
const emptyBranch = () => Branch.fromPartial({});

const sourceBranch = asyncComputed(
  async () => {
    const name = props.headBranchName;
    if (!name) {
      return emptyBranch();
    }
    return await branchStore.fetchBranchByName(name, true /* useCache */);
  },
  emptyBranch(),
  {
    evaluating: isLoadingSourceBranch,
  }
);

const targetBranch = asyncComputed(
  async () => {
    const name = state.branchName;
    if (!name) {
      return emptyBranch();
    }
    return await branchStore.fetchBranchByName(name, true /* useCache */);
  },
  emptyBranch(),
  {
    evaluating: isLoadingTargetBranch,
  }
);

// ready when both source and target branch are loaded
const ready = computed(() => {
  return !isLoadingSourceBranch.value && !isLoadingTargetBranch.value;
});

const targetBranchFilter = (branch: Branch) => {
  return (
    branch.name !== props.headBranchName &&
    branch.engine === sourceBranch.value.engine
  );
};

const handleMergeBranch = async () => {
  try {
    await branchStore.mergeBranch({
      name: targetBranch.value.name,
      headBranch: sourceBranch.value.name,
      mergedSchema: "",
      etag: "",
    });
  } catch (error: any) {
    // If there is conflict, we need to show the conflict and let user resolve it.
    if (error.code === Status.ABORTED) {
      dialog.create({
        positiveText: t("schema-designer.diff-editor.resolve"),
        title: t("schema-designer.diff-editor.auto-merge-failed"),
        content: t("schema-designer.diff-editor.need-to-resolve-conflicts"),
        autoFocus: true,
        closable: true,
        maskClosable: true,
        closeOnEsc: true,
        onPositiveClick: async () => {
          // TODO(d): open resolve window.
          // Fetching the latest base branch.
          await branchStore.fetchBranchByName(
            state.branchName,
            false /* !useCache */
          );
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

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-designer.message.merge-to-main-successfully"),
  });

  emit("merged", state.branchName);
  if (state.deleteBranchAfterMerged) {
    await branchStore.deleteBranch(props.headBranchName);
  }
};

watch(
  [sourceBranch, targetBranch],
  () => {
    state.editingSchema = sourceBranch.value.schema;
  },
  { immediate: true }
);

watch(
  () => props.branchName,
  () => {
    state.branchName = props.branchName;
  },
  { immediate: true }
);
</script>
