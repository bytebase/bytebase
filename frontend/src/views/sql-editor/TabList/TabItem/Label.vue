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
import { create } from "@bufbuild/protobuf";
import { NEllipsis } from "naive-ui";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSQLEditorTabStore, useWorkSheetStore } from "@/store";
import type { SQLEditorTab } from "@/types";
import { WorksheetSchema } from "@/types/proto-es/v1/worksheet_service_pb";
import { useTabListContext } from "../context";

type LocalState = {
  editing: boolean;
  title: string;
};

const props = defineProps<{
  tab: SQLEditorTab;
}>();

const state = reactive<LocalState>({
  editing: false,
  title: props.tab.title,
});

const tabStore = useSQLEditorTabStore();
const worksheetV1Store = useWorkSheetStore();
const inputRef = ref<HTMLInputElement>();
const { events } = useTabListContext();

const readonly = computed(() => {
  const viewState = props.tab.viewState;
  return viewState.view !== "CODE";
});

const isCurrentTab = computed(() => props.tab.id === tabStore.currentTabId);

useEmitteryEventListener(events, "rename-tab", ({ tab }) => {
  if (tab.id === props.tab.id) {
    tabStore.setCurrentTabId(tab.id);
    beginEdit();
  }
});

const beginEdit = () => {
  if (readonly.value) {
    return;
  }
  state.editing = true;
  state.title = props.tab.title;
  nextTick(() => {
    inputRef.value?.select();
  });
};

const confirmEdit = () => {
  const { tab } = props;

  const title = state.title.trim();
  if (title === "") {
    return cancelEdit();
  }

  tabStore.updateTab(tab.id, {
    title,
  });

  if (tab.worksheet) {
    worksheetV1Store.patchWorksheet(
      create(WorksheetSchema, {
        name: tab.worksheet,
        title,
      }),
      ["title"]
    );
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
  position: relative;
  min-width: 6rem;
  max-width: 16rem;
  overflow: hidden;
  text-overflow: ellipsis;
}

.label :deep(.name) {
  width: 100%;
  font-size: 0.875rem;
  line-height: 1.25rem;
}
.edit {
  border: 0;
  border-bottom-width: 1px;
  position: absolute;
  inset: 0;
  padding: 0;
  font-size: 0.875rem;
  line-height: 1.25rem;
}

.label.new :deep(.name) {
  font-style: italic;
}
</style>
