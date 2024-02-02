<template>
  <div class="h-full flex flex-col gap-y-1 relative">
    <NTabs v-model:value="tab">
      <template v-if="branch" #prefix>
        <span class="text-control-placeholder text-sm">
          {{ branch.branchId }}
        </span>
      </template>
      <NTab name="visualized-schema">
        {{ $t("branch.visualized-schema") }}
      </NTab>
      <NTab name="schema-text">
        {{ $t("branch.schema-text") }}
      </NTab>
    </NTabs>
    <div
      v-show="tab === 'visualized-schema'"
      class="flex-1 relative text-sm overflow-y-hidden"
    >
      <SchemaEditorLite
        ref="schemaEditorRef"
        :key="branch?.name"
        resource-type="branch"
        :project="project"
        :readonly="true"
        :branch="branch ?? Branch.fromPartial({})"
        :loading="loading"
      />
    </div>
    <div
      v-show="tab === 'schema-text'"
      class="w-full flex-1 relative text-sm border rounded overflow-clip"
    >
      <MonacoEditor
        :readonly="true"
        :content="branch?.schema ?? ''"
        class="h-full"
      />
    </div>
    <MaskSpinner v-if="loading" />
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";

type TabValue = "visualized-schema" | "schema-text";

defineProps<{
  project: ComposedProject;
  branch: Branch | undefined;
  loading?: boolean;
}>();

const tab = ref<TabValue>("visualized-schema");
</script>
