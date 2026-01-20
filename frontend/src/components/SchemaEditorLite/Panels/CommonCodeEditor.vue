<template>
  <div class="h-full flex flex-col gap-1 overflow-y-hidden pt-1">
    <div class="flex justify-between items-center text-sm">
      <slot name="header-suffix" />
    </div>
    <div class="flex-1 relative px-2 overflow-y-hidden">
      <MonacoEditor
        :content="state.code"
        :readonly="!editable"
        :auto-complete-context="{
          instance: extractDatabaseResourceName(db.name).instance,
          database: db.name,
          scene: 'all',
        }"
        class="border w-full h-full rounded-sm"
        @update:content="handleUpdateCode"
      />
    </div>
    <slot name="preview" />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import type { EditStatus } from "..";

type LocalState = {
  code: string;
};

const props = defineProps<{
  db: Database;
  code: string;
  readonly: boolean;
  status: EditStatus;
}>();

const emit = defineEmits<{
  (e: "update:code", code: string): void;
}>();

const state = reactive<LocalState>({
  code: props.code,
});

const editable = computed(() => {
  if (props.readonly) {
    return false;
  }
  return true;
});

const handleUpdateCode = (code: string) => {
  state.code = code;
  emit("update:code", code);
};

watch(
  () => props.code,
  (code) => {
    state.code = code;
  }
);
</script>
