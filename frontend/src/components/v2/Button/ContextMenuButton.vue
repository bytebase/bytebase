<template>
  <NButtonGroup v-if="currentAction" size="small">
    <NButton
      v-bind="currentAction.props"
      @click="$emit('click', currentAction)"
    >
      <template #icon>
        <slot name="icon" :action="currentAction" />
      </template>
      <slot name="default" :action="currentAction">
        {{ currentAction.text }}
      </slot>
    </NButton>

    <NPopselect
      v-if="options.length > 1"
      :value="currentActionKey"
      :options="options"
      placement="bottom-end"
      trigger="click"
      @update:value="changeActionKey"
    >
      <NButton
        :style="dropdownButtonStyle(currentAction)"
        v-bind="currentAction.props"
      >
        <heroicons-outline:chevron-down />
      </NButton>
    </NPopselect>
  </NButtonGroup>
</template>

<script setup lang="ts">
import { useLocalStorage } from "@vueuse/core";
import { head } from "lodash-es";
import { NButton, NButtonGroup, NPopselect, SelectOption } from "naive-ui";
import { ref, computed } from "vue";
import { ContextMenuButtonAction } from "./types";

const STORE_PREFIX = "bb.context-menu-button";

const props = defineProps<{
  preferenceKey: string;
  actionList: ContextMenuButtonAction[];
  defaultActionKey: string;
}>();

defineEmits<{
  (event: "click", action: ContextMenuButtonAction): void;
}>();

const getStorage = () => {
  const { preferenceKey } = props;
  if (!preferenceKey) return undefined;
  // e.g key = "bb.button-with-context-menu.task-status-transition"
  const key = `${STORE_PREFIX}.${preferenceKey}`;
  return useLocalStorage(key, props.defaultActionKey, {
    listenToStorageChanges: false,
  });
};

// Load user stored default action from localStorage if possible.
// The stored default action may not be in the actionList this time.
// Fallback to the first action if not found.
// But we won't mutate the stored value.
const getDefaultActionKey = (): string | undefined => {
  const storage = getStorage();
  if (storage && storage.value) {
    return storage.value;
  }
  if (props.defaultActionKey) {
    return props.defaultActionKey;
  }

  return props.actionList[0]?.key;
};

const currentActionKey = ref(getDefaultActionKey());
const currentAction = computed(() => {
  const key = currentActionKey.value;
  if (!key) return undefined;
  return (
    props.actionList.find((action) => action.key === key) ??
    head(props.actionList)
  );
});

const options = computed(() => {
  return props.actionList.map<SelectOption>((action) => ({
    label: action.text,
    value: action.key,
  }));
});

const changeActionKey = (key: string) => {
  const action = props.actionList.find((action) => action.key === key);
  if (action) {
    const storage = getStorage();
    if (storage) {
      storage.value = action.key;
    }
  }
  currentActionKey.value = key;
};

const dropdownButtonStyle = (action: ContextMenuButtonAction) => {
  const style: Record<string, any> = {
    "--n-padding": "0 6px",
  };
  if (action.props?.type === "primary") {
    style["--n-padding"] = "0 8px";
    style["border-left"] = "0.5px solid var(--color-accent-hover)";
  }
  return style;
};
</script>
