<template>
  <div class="h-full flex flex-col gap-y-1 relative">
    <NTabs v-model:value="tab">
      <template #prefix>
        <slot name="tab-prefix" />
      </template>
      <NTab name="schema-text">
        {{ $t("branch.schema-text") }}
      </NTab>
      <NTab name="visualized-schema">
        {{ $t("branch.visualized-schema") }}
      </NTab>

      <template #suffix>
        <slot name="tab-suffix" />
      </template>
    </NTabs>
    <div v-show="tab === 'schema-text'" class="flex-1 flex flex-col text-sm">
      <div class="flex items-center justify-between text-control-placeholder">
        <div class="flex-1">
          <slot name="baseline-title">
            {{ $t("branch.baseline") }}
          </slot>
        </div>
        <div class="flex-1">
          <slot name="head-title">
            {{ $t("branch.head") }}
          </slot>
        </div>
      </div>
      <div class="flex-1 w-full relative text-sm border rounded overflow-clip">
        <DiffEditor
          :readonly="true"
          :original="virtualBranch?.baselineSchema ?? ''"
          :modified="virtualBranch?.schema ?? ''"
          class="h-full"
        />
      </div>
    </div>
    <div
      v-show="tab === 'visualized-schema'"
      class="flex-1 relative text-sm overflow-y-hidden"
    >
      <SchemaEditorLite
        ref="schemaEditorRef"
        :key="virtualBranch?.name"
        resource-type="branch"
        :project="project"
        :readonly="true"
        :branch="virtualBranch ?? Branch.fromPartial({})"
        :loading="combinedLoading"
      />
    </div>
    <MaskSpinner v-if="combinedLoading" />
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { v1 as uuidv1 } from "uuid";
import { computed, ref, watch } from "vue";
import { DiffEditor } from "@/components/MonacoEditor";
import SchemaEditorLite from "@/components/SchemaEditorLite";
import { useDatabaseV1Store } from "@/store";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { DatabaseMetadata } from "@/types/proto/v1/database_service";

type TabValue = "schema-text" | "visualized-schema";

const props = defineProps<{
  project: ComposedProject;
  base: Branch | undefined;
  head: Branch | undefined;
  isBaseLoading?: boolean;
  isHeadLoading?: boolean;
}>();

const tab = ref<TabValue>("schema-text");
const schemaEditorRef = ref<InstanceType<typeof SchemaEditorLite>>();
const combinedLoading = computed(() => {
  return props.isBaseLoading || props.isHeadLoading;
});

const emptyBranch = () => {
  return Branch.fromPartial({});
};

const virtualBranch = computed(() => {
  if (combinedLoading.value) {
    return emptyBranch();
  }
  const { project, base, head } = props;
  if (!base || !head) {
    return emptyBranch();
  }
  return Branch.fromPartial({
    name: `${project.name}/branches/${uuidv1()}`,
    baselineDatabase: base.baselineDatabase,
    baselineSchema: base.schema,
    baselineSchemaMetadata: cloneDeep(base.schemaMetadata),
    schema: head.schema,
    schemaMetadata: cloneDeep(head.schemaMetadata),
  });
});

// re-calculate diff for coloring when branch changed
watch(
  [virtualBranch, schemaEditorRef],
  ([branch, editor]) => {
    if (branch && editor) {
      const db = useDatabaseV1Store().getDatabaseByName(
        branch.baselineDatabase
      );
      editor.rebuildMetadataEdit(
        db,
        branch.baselineSchemaMetadata ?? DatabaseMetadata.fromPartial({}),
        branch.schemaMetadata ?? DatabaseMetadata.fromPartial({})
      );
    }
  },
  {
    immediate: true,
  }
);
</script>
