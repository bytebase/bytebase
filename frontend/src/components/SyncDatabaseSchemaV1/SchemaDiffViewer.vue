<template>
  <div class="w-full h-full flex flex-col gap-2">
    <div class="w-full flex flex-row justify-between items-center">
      <span>{{ title }}</span>
      <div class="flex gap-x-2 shrink-0">
        <NButton size="small" @click="handleNavigatorClick('previous')">
          <template #icon>
            <ArrowUpIcon class="w-5 h-auto" />
          </template>
        </NButton>
        <NButton size="small" @click="handleNavigatorClick('next')">
          <template #icon>
            <ArrowDownIcon class="w-5 h-auto" />
          </template>
        </NButton>
        <NButton
          v-if="showFullscreen"
          size="small"
          @click="showFullscreenModal = true"
        >
          <template #icon>
            <Maximize2Icon class="w-5 h-auto" />
          </template>
        </NButton>
      </div>
    </div>
    <div class="w-full flex-1 overflow-y-scroll border">
      <DiffEditor
        ref="diffEditorRef"
        class="h-full"
        :original="normalizedOriginal"
        :modified="normalizedModified"
        :options="{ ignoreTrimWhitespace: true }"
        :readonly="true"
      />
    </div>
  </div>

  <SchemaDiffViewerModal
    v-if="showFullscreenModal"
    :title="title"
    :original="normalizedOriginal"
    :modified="normalizedModified"
    @close="showFullscreenModal = false"
  />
</template>

<script lang="ts" setup>
import { ArrowDownIcon, ArrowUpIcon, Maximize2Icon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { DiffEditor } from "@/components/MonacoEditor";
import SchemaDiffViewerModal from "./SchemaDiffViewerModal.vue";

const props = defineProps<{
  title: string;
  original: string;
  modified: string;
  showFullscreen?: boolean;
}>();

const normalizeLineEndings = (content: string) =>
  content.replace(/\r\n?/g, "\n");

const normalizedOriginal = computed(() => normalizeLineEndings(props.original));
const normalizedModified = computed(() => normalizeLineEndings(props.modified));

const showFullscreenModal = ref(false);
const diffEditorRef = ref<InstanceType<typeof DiffEditor>>();

const handleNavigatorClick = (target: "next" | "previous") => {
  diffEditorRef?.value?.diffEditor?.editor?.goToDiff(target);
};
</script>
