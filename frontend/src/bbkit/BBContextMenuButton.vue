<template>
  <button
    class="select-none h-[38px] inline-flex items-center border rounded-md disabled:opacity-50 disabled:cursor-not-allowed text-sm leading-5 font-medium overflow-hidden focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
    :data-button-type="state.currentAction.type ?? 'NORMAL'"
    :class="actionButtonClass.wrapper"
    :disabled="disabled"
  >
    <div
      class="flex items-center px-4 py-2"
      :class="[actionButtonClass.btn, disabled && 'pointer-events-none']"
      @click.stop="handleClickButton"
    >
      <slot name="default" :action="state.currentAction">
        {{ state.currentAction.text }}
      </slot>
    </div>

    <div
      v-if="actionList.length > 1"
      class="flex h-full w-0 border-l"
      :class="actionButtonClass.divider"
    />

    <NPopover
      v-if="actionList.length > 1"
      ref="popover"
      :disabled="disabled"
      placement="bottom-end"
      trigger="click"
      raw
      style="box-shadow: none"
      :show-arrow="false"
    >
      <template #trigger>
        <div
          class="flex items-center px-2 h-full"
          :class="[actionButtonClass.btn, disabled && 'pointer-events-none']"
        >
          <heroicons-outline:chevron-down />
        </div>
      </template>

      <div
        class="flex flex-col divide-y divide-gray-200 border border-gray-200 rounded-md shadow-md overflow-hidden"
      >
        <slot
          v-for="(action, index) in actionList"
          :key="action.key"
          name="action-item"
          :action="action"
          :index="index"
          :selected="isCurrentAction(action)"
          :click-handler="() => handleClickActionList(action)"
        >
          <div
            class="flex items-center justify-between gap-x-4 py-2 pl-4 pr-2 bg-white cursor-pointer text-sm font-medium text-control hover:bg-main-hover hover:text-white"
            @click.stop="handleClickActionList(action)"
          >
            <span>{{ action.text }}</span>
            <span :class="!isCurrentAction(action) && 'invisible'">
              <heroicons-outline:check />
            </span>
          </div>
        </slot>
      </div>
    </NPopover>
  </button>
</template>

<script lang="ts" setup>
import { useLocalStorage } from "@vueuse/core";
import { NPopover } from "naive-ui";
import { computed, PropType, reactive, shallowRef, watch } from "vue";
import { BBButtonType } from "./types";

export type ButtonAction<T = unknown> = {
  key: string;
  text: string;
  type: BBButtonType;
  params: T;
};

type LocalState = {
  currentAction: ButtonAction;
};

const STORE_PREFIX = "bb.button-with-context-menu";

const props = defineProps({
  preferenceKey: {
    type: String,
    default: "",
  },
  actionList: {
    type: Array as PropType<ButtonAction[]>,
    required: true,
  },
  defaultActionKey: {
    type: String,
    default: "",
  },
  disabled: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (event: "click", action: ButtonAction): void;
}>();

const popover = shallowRef<InstanceType<typeof NPopover>>();

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
const getDefaultAction = (): ButtonAction => {
  const storage = getStorage();
  if (storage && storage.value) {
    const storedDefaultAction = props.actionList.find(
      (action) => action.key === storage.value
    );
    if (storedDefaultAction) {
      return storedDefaultAction;
    }
  }
  if (props.defaultActionKey) {
    const defaultAction = props.actionList.find(
      (action) => action.key === props.defaultActionKey
    );
    if (defaultAction) {
      return defaultAction;
    }
  }

  return props.actionList[0];
};

const state = reactive<LocalState>({
  currentAction: getDefaultAction(),
});

const actionButtonClass = computed(() => {
  const { type } = state.currentAction;
  if (type in BUTTON_CLASS_MAP) {
    return BUTTON_CLASS_MAP[type];
  }
  return BUTTON_CLASS_MAP.NORMAL;
});

const isCurrentAction = (action: ButtonAction) => {
  return action.key === state.currentAction.key;
};

const handleClickActionList = (action: ButtonAction) => {
  const reset = () => {
    popover.value?.setShow(false);
  };

  if (isCurrentAction(action)) return reset();

  const storage = getStorage();
  if (storage) {
    // Save the selected action as the default action
    storage.value = action.key;
  }

  state.currentAction = action;

  reset();
};

const handleClickButton = () => {
  if (props.disabled) return;
  emit("click", state.currentAction);
};

watch(
  () => props.actionList,
  () => {
    state.currentAction = getDefaultAction();
  }
);

const BUTTON_CLASS_MAP: Record<
  BBButtonType,
  {
    wrapper: string;
    btn: string;
    divider: string;
  }
> = {
  NORMAL: {
    wrapper: "border-control-border text-control disabled:bg-control-bg",
    btn: "bg-white hover:bg-control-bg-hover",
    divider: "border-control-border",
  },
  PRIMARY: {
    wrapper: "border-transparent text-accent-text disabled:bg-accent",
    btn: "bg-accent hover:bg-accent-hover",
    divider: "border-indigo-500",
  },
  SECONDARY: {
    wrapper: "border-control-border text-main  disabled:bg-control-bg",
    btn: "bg-white hover:bg-control-bg-hover hover:text-control-hover",
    divider: "border-control-border",
  },
  SUCCESS: {
    wrapper: "border-transparent text-accent-text disabled:bg-success",
    btn: "bg-success hover:bg-success-hover",
    divider: "border-green-500",
  },
  DANGER: {
    wrapper: "border-transparent text-accent-text disabled:bg-error",
    btn: "bg-error hover:bg-error-hover",
    divider: "border-red-500",
  },
};
</script>
