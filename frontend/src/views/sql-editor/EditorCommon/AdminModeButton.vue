<template>
  <NButton
    v-if="showButton"
    :size="props.size"
    type="warning"
    :ghost="true"
    :disabled="isDisconnected"
    @click.stop="enterAdminMode"
  >
    <template #icon>
      <WrenchIcon class="w-4 h-4" />
    </template>
    <span v-if="!hideText">{{ $t("sql-editor.admin-mode.self") }}</span>
  </NButton>
</template>

<script lang="ts" setup>
import { WrenchIcon } from "lucide-vue-next";
import { type ButtonProps, NButton } from "naive-ui";
import { storeToRefs } from "pinia";
import type { PropType } from "vue";
import { computed } from "vue";
import { useSQLEditorStore, useSQLEditorTabStore } from "@/store";

const emit = defineEmits<{
  (e: "enter"): void;
}>();

const props = defineProps({
  size: {
    type: String as PropType<ButtonProps["size"]>,
    default: "medium",
  },
  hideText: {
    type: Boolean,
    default: false,
  },
});

const tabStore = useSQLEditorTabStore();
const editorStore = useSQLEditorStore();

const { currentTab, isDisconnected } = storeToRefs(tabStore);

const showButton = computed(() => {
  if (!editorStore.allowAdmin) return false;
  const mode = currentTab.value?.mode;
  return mode === "WORKSHEET";
});

const enterAdminMode = async () => {
  const tab = currentTab.value;
  if (!tab) {
    return;
  }

  tabStore.updateCurrentTab({
    mode: "ADMIN",
  });

  emit("enter");
};
</script>
