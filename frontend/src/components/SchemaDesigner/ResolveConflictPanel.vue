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
            :original="baselineSchemaDesign.schema"
            :value="state.editingSchema"
            @change="state.editingSchema = $event"
          />
        </div>
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('dismiss')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" @click="handleResolveSchemaDesignDraft">
            {{ $t("schema-designer.diff-editor.resolve-and-merge-again") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { NDrawer, NDrawerContent } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import DiffEditor from "@/components/MonacoEditor/DiffEditor.vue";
import { useSheetV1Store } from "@/store";
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
  // Should be a schema design name of personal draft.
  schemaDesignName: string;
}>();

const emit = defineEmits(["dismiss", "try-merge"]);

const state = reactive<LocalState>({
  editingSchema: "",
  initialized: false,
});
const sheetStore = useSheetV1Store();
const schemaDesignStore = useSchemaDesignStore();

const schemaDesign = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(props.schemaDesignName || "");
});

const baselineSchemaDesign = computed(() => {
  return schemaDesignStore.getSchemaDesignByName(
    schemaDesign.value.baselineSheetName || ""
  );
});

const prepareLastestBaselineSchemaDesign = async () => {
  await schemaDesignStore.fetchSchemaDesignByName(
    schemaDesign.value.baselineSheetName
  );
};

onMounted(async () => {
  await prepareLastestBaselineSchemaDesign();
  state.editingSchema = schemaDesign.value.schema;
  state.initialized = true;
});

const handleResolveSchemaDesignDraft = async () => {
  const updateMask = ["schema", "baseline_sheet_name"];
  const [projectName] = getProjectAndSchemaDesignSheetId(
    baselineSchemaDesign.value.name
  );
  // Create a baseline sheet for the schema design.
  const baselineSheet = await sheetStore.createSheet(
    `projects/${projectName}`,
    {
      name: `baseline schema of ${baselineSchemaDesign.value.title}`,
      database: schemaDesign.value.baselineDatabase,
      content: new TextEncoder().encode(baselineSchemaDesign.value.schema),
      visibility: Sheet_Visibility.VISIBILITY_PROJECT,
      source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
      type: Sheet_Type.TYPE_SQL,
    }
  );

  // Update the schema design draft first.
  await schemaDesignStore.updateSchemaDesign(
    SchemaDesign.fromPartial({
      name: schemaDesign.value.name,
      engine: schemaDesign.value.engine,
      schema: state.editingSchema,
      baselineSheetName: baselineSheet.name,
    }),
    updateMask
  );

  // Try to merge the schema design draft again.
  emit("try-merge");
};
</script>
