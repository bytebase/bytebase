<template>
  <NPopconfirm
    v-bind="popconfirmProps"
    :disabled="!tooltip"
    @positive-click="handleConfirm"
  >
    <template #trigger>
      <NButton
        :loading="loading"
        style="--n-icon-margin: 4px; --n-icon-size: 14px"
        v-bind="buttonAttrs"
      >
        <slot />
      </NButton>
    </template>

    <div v-if="tooltip" :class="tooltipClass">{{ tooltip }}</div>
  </NPopconfirm>
</template>

<script lang="ts">
import { defineComponent } from "vue";

defineComponent({
  inheritAttrs: false,
});
</script>

<script lang="ts" setup>
import { computed, ref, useAttrs } from "vue";
import { omit } from "lodash-es";
import { VueClass } from "@/utils";
import {
  type ButtonProps,
  type PopconfirmProps,
  NButton,
  NPopconfirm,
} from "naive-ui";

export interface SpinnerButtonProps extends ButtonProps {
  onConfirm: () => Promise<any>;
  tooltip?: string;
  tooltipClass?: VueClass;
  popconfirmProps?: PopconfirmProps;
}
const props = defineProps<SpinnerButtonProps>();

const attrs = useAttrs();
const buttonAttrs = computed(() =>
  omit(attrs, "tooltip", "tooltipClass", "popconfirmProps")
);

const loading = ref(false);

const handleConfirm = async () => {
  if (loading.value) return;

  loading.value = true;
  try {
    await props.onConfirm();
  } finally {
    loading.value = false;
  }
};
</script>
