<template>
  <BBModal
    v-if="state.visible"
    :title="title"
    :subtitle="subtitle"
    :show-close="closable"
    :esc-closable="closable"
    @close="goNegative"
  >
    <slot name="default"></slot>

    <div class="pt-4 border-t border-block-border flex justify-end space-x-3">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="goNegative"
      >
        {{ negativeText || $t("common.cancel") }}
      </button>
      <button
        type="button"
        class="btn-primary py-2 px-4"
        @click.prevent="goPositive"
      >
        {{ positiveText || $t("common.confirm") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { Defer, defer } from "@/utils";

defineProps({
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
  negativeText: {
    type: String,
    default: undefined,
  },
  positiveText: {
    type: String,
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

const goPositive = () => {
  state.visible = false;
  state.defer?.resolve(true);
};

const goNegative = () => {
  state.visible = false;
  state.defer?.resolve(false);
};

defineExpose({ open });
</script>
