<template>
  <div v-if="viewMode !== 'NONE'" class="px-4 py-2 flex flex-col gap-y-2">
    <EditorView ref="editorViewRef" :advices="advices" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ref } from "vue";
import { nextTick } from "vue";
import { useRouter } from "vue-router";
import { EMPTY_ID } from "@/types";
import type { Advice } from "@/types/proto/api/v1alpha/sql_service";
import { usePlanContext } from "../../logic";
import EditorView from "./EditorView";

defineProps<{
  advices?: Advice[];
}>();

const { selectedSpec } = usePlanContext();

const editorViewRef = ref<InstanceType<typeof EditorView>>();
const router = useRouter();

type ViewMode = "NONE" | "EDITOR";

const viewMode = computed((): ViewMode => {
  if (selectedSpec.value.id !== String(EMPTY_ID)) {
    return "EDITOR";
  }

  return "NONE";
});

router.afterEach((to) => {
  if (to.hash) {
    scrollToLineByHash(to.hash);
  }
});

const scrollToLineByHash = (hash: string) => {
  const match = hash.match(/^#L(\d+)$/);
  if (!match) return;
  const lineNumber = parseInt(match[1], 10);
  nextTick(() => {
    editorViewRef.value?.editor?.monacoEditor?.editor?.codeEditor?.revealLineNearTop(
      lineNumber
    );
  });
};
</script>
