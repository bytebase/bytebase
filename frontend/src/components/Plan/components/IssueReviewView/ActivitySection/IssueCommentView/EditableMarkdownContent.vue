<template>
  <div>
    <p v-if="!isEditing && !content">
      <i class="text-gray-400 italic">{{ placeholder }}</i>
    </p>
    <MarkdownEditor
      v-else
      :mode="isEditing ? 'editor' : 'preview'"
      :content="isEditing ? editContent : content"
      :project="project"
      :maxlength="maxlength"
      :max-height="maxHeight"
      @change="(val: string) => emit('update:editContent', val)"
      @submit="emit('save')"
    />
    <div
      v-if="isEditing"
      class="flex gap-x-2 mt-2 items-center justify-end"
    >
      <NButton quaternary size="small" @click.prevent="emit('cancel')">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton
        size="small"
        :disabled="!allowSave"
        :loading="isSaving"
        @click.prevent="emit('save')"
      >
        {{ $t("common.save") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import MarkdownEditor from "@/components/MarkdownEditor";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

withDefaults(
  defineProps<{
    content: string;
    editContent: string;
    project: Project;
    isEditing: boolean;
    allowSave: boolean;
    isSaving?: boolean;
    placeholder?: string;
    maxlength?: number;
    maxHeight?: number;
  }>(),
  {
    isSaving: false,
    placeholder: "",
    maxlength: 65536,
    maxHeight: Number.MAX_SAFE_INTEGER,
  }
);

const emit = defineEmits<{
  (e: "update:editContent", value: string): void;
  (e: "save"): void;
  (e: "cancel"): void;
}>();
</script>
