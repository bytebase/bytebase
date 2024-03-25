<template>
  <div class="label" :class="[tab.status.toLowerCase()]">
    <NEllipsis
      class="name"
      :tooltip="{
        placement: 'top',
        delay: 250,
      }"
      :class="state.editing && 'invisible'"
      @dblclick="beginEdit"
    >
      {{ state.title }}
    </NEllipsis>

    <input
      v-if="state.editing"
      ref="inputRef"
      v-model="state.title"
      type="text"
      class="edit"
      @blur="confirmEdit"
      @keyup.enter="(e) => (e.target as HTMLInputElement).blur()"
      @keyup.esc="cancelEdit"
    />
  </div>
</template>
<script lang="ts" setup>
import { NEllipsis } from "naive-ui";
import { computed, nextTick, PropType, reactive, ref, watch } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSQLEditorTabStore, useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { useTabListContext } from "../context";

type LocalState = {
  editing: boolean;
  title: string;
};

const props = defineProps({
  tab: {
    type: Object as PropType<SQLEditorTab>,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
});

const state = reactive<LocalState>({
  editing: false,
  title: props.tab.title,
});

const tabStore = useSQLEditorTabStore();
const worksheetV1Store = useWorkSheetStore();
const inputRef = ref<HTMLInputElement>();
const { events } = useTabListContext();

const isCurrentTab = computed(() => props.tab.id === tabStore.currentTabId);

useEmitteryEventListener(events, "rename-tab", ({ tab }) => {
  if (tab.id === props.tab.id) {
    tabStore.setCurrentTabId(tab.id);
    beginEdit();
  }
});

const beginEdit = () => {
  state.editing = true;
  state.title = props.tab.title;
  nextTick(() => {
    inputRef.value?.focus();
  });
};

const confirmEdit = () => {
  const { tab } = props;

  const title = state.title.trim();
  if (title === "") {
    return cancelEdit();
  }

  tab.title = title;
  tab.status = "DIRTY";
  if (tab.sheet) {
    worksheetV1Store
      .patchSheet(
        {
          name: tab.sheet,
          title,
        },
        ["title"]
      )
      .then(() => {
        tab.status = "CLEAN";
      });
  }

  state.editing = false;
};

const cancelEdit = () => {
  state.editing = false;
  state.title = props.tab.title;
};

watch(
  () => props.tab.title,
  (title) => {
    state.title = title;
  }
);
watch(isCurrentTab, (value) => {
  if (!value) {
    cancelEdit();
  }
});
</script>

<style scoped lang="postcss">
.label {
  @apply relative flex items-center whitespace-nowrap min-w-[6rem] max-w-[12rem] truncate;
}

.label :deep(.name) {
  @apply h-6 w-full flex items-center text-sm;
}
.edit {
  @apply border-0 border-b absolute inset-0 p-0 text-sm;
}

.label.new :deep(.name) {
  @apply italic;
}
</style>
