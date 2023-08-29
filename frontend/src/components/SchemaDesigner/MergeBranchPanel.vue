<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    :close-on-esc="true"
    @update:show="(show: boolean) => !show && emit('dismiss')"
  >
    <NDrawerContent
      :title="$t('schema-designer.diff-editor.self')"
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
        <div class="pt-4 pb-6 w-full flex flex-row justify-center items-center">
          <NInput
            class="!w-40 text-center"
            readonly
            :value="sourceBranch.title"
          />
          <div class="mx-16">
            <MoveLeft :size="32" stroke-width="1" />
          </div>
          <NInput
            class="!w-40 text-center"
            readonly
            :value="targetBranch.title"
          />
        </div>
        <div class="w-full grid grid-cols-2">
          <div class="col-span-1">
            <span>{{ $t("schema-designer.diff-editor.latest-schema") }}</span>
          </div>
          <div class="col-span-1">
            <span>{{ $t("schema-designer.diff-editor.editing-schema") }}</span>
          </div>
        </div>
        <div class="w-full h-[calc(100%-8rem)] border">
          <DiffEditor
            v-if="state.initialized"
            class="h-full"
            :original="sourceBranch.schema"
            :value="state.editingSchema"
            @change="state.editingSchema = $event"
          />
        </div>
      </div>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { MoveLeft } from "lucide-vue-next";
import { NButton, NDrawer, NDrawerContent, NInput } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import DiffEditor from "@/components/MonacoEditor/DiffEditor.vue";
import { pushNotification, useSheetV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import { getProjectAndSchemaDesignSheetId } from "@/store/modules/v1/common";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import {
  Sheet_Source,
  Sheet_Type,
  Sheet_Visibility,
} from "@/types/proto/v1/sheet_service";

interface LocalState {
  editingSchema: string;
  initialized: boolean;
}

const props = defineProps<{
  sourceBranchName: string;
  targetBranchName: string;
}>();

const emit = defineEmits(["dismiss", "try-merge"]);

const state = reactive<LocalState>({
  editingSchema: "",
  initialized: false,
});
const { t } = useI18n();
const sheetStore = useSheetV1Store();
const schemaDesignStore = useSchemaDesignStore();

const sourceBranch = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(props.sourceBranchName || "");
});

const targetBranch = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(props.targetBranchName || "");
});

onMounted(async () => {
  state.editingSchema = targetBranch.value.schema;
  state.initialized = true;
});

const handleSaveDraft = async (ignoreNotify?: boolean) => {
  const updateMask = ["schema", "baseline_sheet_name"];
  const [projectName] = getProjectAndSchemaDesignSheetId(
    targetBranch.value.name
  );
  // Create a baseline sheet for the schema design.
  const baselineSheet = await sheetStore.createSheet(
    `projects/${projectName}`,
    {
      name: `baseline schema of ${sourceBranch.value.title}`,
      database: targetBranch.value.baselineDatabase,
      content: new TextEncoder().encode(sourceBranch.value.schema),
      visibility: Sheet_Visibility.VISIBILITY_PROJECT,
      source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
      type: Sheet_Type.TYPE_SQL,
    }
  );

  // Update the schema design draft first.
  await schemaDesignStore.updateSchemaDesign(
    SchemaDesign.fromPartial({
      name: targetBranch.value.name,
      engine: targetBranch.value.engine,
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

  // Try to merge the schema design draft again.
  emit("try-merge");
};
</script>
