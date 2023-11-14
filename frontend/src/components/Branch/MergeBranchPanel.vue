<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    :close-on-esc="true"
    @update:show="(show: boolean) => !show && emit('dismiss')"
  >
    <NDrawerContent
      :title="$t('schema-designer.merge-branch')"
      :closable="true"
    >
      <div
        class="space-y-3 w-[calc(100vw-24rem)] min-w-[64rem] max-w-[calc(100vw-8rem)] h-full overflow-x-auto"
      >
        <div class="w-full flex flex-row justify-end items-center gap-2">
          <NButton @click="() => handleSaveDraft()">{{
            $t("schema-designer.save-draft")
          }}</NButton>
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
              :branch="state.targetBranchName"
              :filter="targetBranchFilter"
              @update:branch="
                (branch) => (state.targetBranchName = branch ?? '')
              "
            />
          </div>
          <div class="flex flex-row justify-center">
            <MoveLeft :size="40" stroke-width="1" />
          </div>
          <div class="flex flex-row justify-start">
            <NInput
              class="!w-4/5 text-center"
              readonly
              :value="sourceBranch.title"
              size="large"
            />
          </div>
        </div>
        <div class="w-full grid grid-cols-2">
          <div class="col-span-1">
            <span>{{ $t("schema-designer.diff-editor.latest-schema") }}</span>
          </div>
          <div class="col-span-1">
            <span>{{ $t("schema-designer.diff-editor.editing-schema") }}</span>
          </div>
        </div>
        <div class="w-full h-[calc(100%-14rem)] border relative">
          <DiffEditor
            v-if="ready"
            :key="state.targetBranchName"
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
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { asyncComputed } from "@vueuse/core";
import { MoveLeft } from "lucide-vue-next";
import {
  NButton,
  NDrawer,
  NDrawerContent,
  NInput,
  NCheckbox,
  useDialog,
} from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useSheetV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import {
  getProjectAndSchemaDesignSheetId,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import {
  Sheet_Source,
  Sheet_Type,
  Sheet_Visibility,
} from "@/types/proto/v1/sheet_service";
import { DiffEditor } from "../MonacoEditor";

interface LocalState {
  targetBranchName: string;
  editingSchema: string;
  deleteBranchAfterMerged: boolean;
}

const props = defineProps<{
  sourceBranchName: string;
  targetBranchName: string;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "merged", branchName: string): void;
}>();

const state = reactive<LocalState>({
  targetBranchName: "",
  editingSchema: "",
  deleteBranchAfterMerged: true,
});
const { t } = useI18n();
const dialog = useDialog();
const sheetStore = useSheetV1Store();
const schemaDesignStore = useSchemaDesignStore();
const isLoadingSourceBranch = ref(false);
const isLoadingTargetBranch = ref(false);
const emptyBranch = () => SchemaDesign.fromPartial({});

const sourceBranch = asyncComputed(
  async () => {
    const name = props.sourceBranchName;
    if (!name) {
      return emptyBranch();
    }
    return await schemaDesignStore.fetchSchemaDesignByName(
      name,
      true /* useCache */
    );
  },
  emptyBranch(),
  {
    evaluating: isLoadingSourceBranch,
  }
);

const targetBranch = asyncComputed(
  async () => {
    const name = state.targetBranchName;
    if (!name) {
      return emptyBranch();
    }
    return await schemaDesignStore.fetchSchemaDesignByName(
      name,
      true /* useCache */
    );
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

const targetBranchFilter = (branch: SchemaDesign) => {
  return (
    branch.name !== props.sourceBranchName &&
    branch.engine === sourceBranch.value.engine
  );
};

const handleSaveDraft = async (ignoreNotify?: boolean) => {
  const updateMask = ["schema", "baseline_sheet_name"];
  const [projectName] = getProjectAndSchemaDesignSheetId(
    sourceBranch.value.name
  );
  // Create a baseline sheet for the schema design.
  const baselineSheet = await sheetStore.createSheet(
    `${projectNamePrefix}${projectName}`,
    {
      title: `baseline schema of ${targetBranch.value.title}`,
      database: targetBranch.value.baselineDatabase,
      content: new TextEncoder().encode(targetBranch.value.schema),
      visibility: Sheet_Visibility.VISIBILITY_PROJECT,
      source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
      type: Sheet_Type.TYPE_SQL,
    }
  );

  // Update the schema design draft first.
  await schemaDesignStore.updateSchemaDesign(
    SchemaDesign.fromPartial({
      name: sourceBranch.value.name,
      engine: sourceBranch.value.engine,
      baselineDatabase: sourceBranch.value.baselineDatabase,
      schema: state.editingSchema,
      baselineSheetName: baselineSheet.name,
    }),
    updateMask
  );

  if (!ignoreNotify) {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("schema-designer.message.updated-succeed"),
    });
    emit("dismiss");
  }
};

const handleMergeBranch = async () => {
  await handleSaveDraft(true);

  try {
    await schemaDesignStore.mergeSchemaDesign({
      name: sourceBranch.value.name,
      targetName: targetBranch.value.name,
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
          // Do nothing.
        },
        onPositiveClick: async () => {
          // Fetching the latest target branch.
          await schemaDesignStore.fetchSchemaDesignByName(
            state.targetBranchName,
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

  emit("merged", state.targetBranchName);
  if (state.deleteBranchAfterMerged) {
    await schemaDesignStore.deleteSchemaDesign(props.sourceBranchName);
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
  () => props.targetBranchName,
  () => {
    state.targetBranchName = props.targetBranchName;
  },
  { immediate: true }
);
</script>
