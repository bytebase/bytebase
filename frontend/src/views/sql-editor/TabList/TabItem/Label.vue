<template>
  <div class="label">
    <span
      class="name"
      :class="state.editing && 'invisible'"
      @dblclick="beginEdit"
    >
      {{ state.name }}
    </span>

    <input
      v-if="state.editing"
      ref="inputRef"
      v-model="state.name"
      type="text"
      class="edit"
      @blur="confirmEdit"
      @keyup.enter="confirmEdit"
      @keyup.esc="cancelEdit"
    />
  </div>
</template>
<script lang="ts" setup>
import { computed, nextTick, PropType, reactive, ref, watch } from "vue";
import { useSheetV1Store, useTabStore } from "@/store";
import type { TabInfo } from "@/types";

type LocalState = {
  editing: boolean;
  name: string;
};

const props = defineProps({
  tab: {
    type: Object as PropType<TabInfo>,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
});

const state = reactive<LocalState>({
  editing: false,
  name: props.tab.name,
});

const tabStore = useTabStore();
const sheetV1Store = useSheetV1Store();
const inputRef = ref<HTMLInputElement>();

const isCurrentTab = computed(() => props.tab.id === tabStore.currentTabId);

const beginEdit = () => {
  state.editing = true;
  state.name = props.tab.name;
  nextTick(() => {
    inputRef.value?.focus();
  });
};

const confirmEdit = () => {
  const { tab } = props;

  const name = state.name.trim();
  if (name === "") {
    return cancelEdit();
  }

  tab.name = name;
  if (tab.sheetName) {
    sheetV1Store.patchSheet({
      name: tab.sheetName,
      title: name,
    });
  }

  state.editing = false;
};

const cancelEdit = () => {
  state.editing = false;
  state.name = props.tab.name;
};

watch(
  () => props.tab.name,
  (name) => {
    state.name = name;
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

.name {
  @apply h-6 w-full flex items-center text-sm;
}
.edit {
  @apply border-0 border-b absolute inset-0 p-0 text-sm;
}

.temp .name {
  @apply italic;
}
</style>
