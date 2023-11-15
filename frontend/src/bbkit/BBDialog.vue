<template>
  <BBModal
    v-if="state.visible"
    :title="title"
    :subtitle="subtitle"
    :show-close="closable"
    :close-on-esc="closable"
    @close="onNegativeClick"
  >
    <slot name="default"></slot>

    <div class="pt-4 border-t border-block-border flex justify-end space-x-3">
      <button
        v-if="showNegativeBtn"
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="onNegativeClick"
      >
        {{ negativeText || $t("common.cancel") }}
      </button>
      <button
        type="button"
        class="btn-primary py-2 px-4"
        data-label="bb-modal-confirm-button"
        @click.prevent="onPositiveClick"
      >
        {{ positiveText || $t("common.confirm") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { PropType, reactive } from "vue";
import { Defer, defer } from "@/utils";

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  subtitle: {
    type: String,
    default: "",
  },
  closable: {
    type: Boolean,
    default: false,
  },
  showNegativeBtn: {
    type: Boolean,
    default: true,
  },
  negativeText: {
    type: String,
    default: undefined,
  },
  positiveText: {
    type: String,
    default: undefined,
  },
  onBeforePositiveClick: {
    type: Function as PropType<() => boolean>,
    default: undefined,
  },
  onBeforeNegativeClick: {
    type: Function as PropType<() => boolean>,
    default: undefined,
  },
});

const state = reactive({
  visible: false,
  defer: undefined as Defer<boolean> | undefined,
});

const open = () => {
  if (state.defer) {
    state.defer.reject(new Error("duplicated call"));
  }

  state.defer = defer<boolean>();
  state.visible = true;

  return state.defer.promise;
};

const onPositiveClick = () => {
  if (props.onBeforePositiveClick) {
    if (!props.onBeforePositiveClick()) {
      return;
    }
  }

  state.visible = false;
  state.defer?.resolve(true);
};

const onNegativeClick = () => {
  if (props.onBeforeNegativeClick) {
    if (!props.onBeforeNegativeClick()) {
      return;
    }
  }

  state.visible = false;
  state.defer?.resolve(false);
};

defineExpose({ open });
</script>
