<template>
  <div class="w-full flex flex-col gap-4">
    <div class="w-full flex flex-row items-center gap-4">
      <NInputGroup class="!w-auto">
        <NInputGroupLabel>
          {{ $t("common.version") }}
          <span class="text-red-600">*</span>
        </NInputGroupLabel>
        <NInput v-model:value="state.version" class="!w-40" />
      </NInputGroup>
      <NInputGroup>
        <NInputGroupLabel>
          {{ $t("database.revision.filename") }}
        </NInputGroupLabel>
        <NInput v-model:value="state.name" class="!w-56" />
      </NInputGroup>
    </div>

    <div class="w-full flex flex-col gap-y-2">
      <p class="w-auto textinfolabel">
        {{ $t("common.statement") }}
        <span class="text-red-600">*</span>
      </p>
      <MonacoEditor
        v-model:content="state.statement"
        class="h-auto min-h-[120px] max-h-[300px] border rounded-md text-sm overflow-auto"
        :auto-height="{ min: 120, max: 300 }"
        :auto-focus="false"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput, NInputGroup, NInputGroupLabel } from "naive-ui";
import { reactive, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import type { FileToCreate } from "../context";

const props = defineProps<{
  file: FileToCreate;
}>();

const emit = defineEmits<{
  (event: "update", value: FileToCreate): void;
}>();

const state = reactive({ ...props.file });

watch(
  () => state,
  (newState) => {
    emit("update", newState);
  },
  { deep: true }
);
</script>
