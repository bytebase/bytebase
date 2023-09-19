<template>
  <textarea
    ref="textareaRef"
    v-model="state.value"
    class="text-main focus:ring-control focus:border-control sm:text-sm resize-none whitespace-pre-wrap border-control-border rounded-md disabled:cursor-not-allowed"
  />
</template>

<script lang="ts" setup>
import { useEventListener } from "@vueuse/core";
import { debounce } from "lodash-es";
import { onMounted, reactive, ref, watch } from "vue";
import { useVModel } from "@/composables/useVModel";
import { sizeToFit } from "@/utils";

type LocalState = {
  value: string | undefined;
};

const props = withDefaults(
  defineProps<{
    value?: string;
    padding?: number;
    maxHeight?: number;
    minHeight?: number;
  }>(),
  {
    value: undefined,
    padding: 2,
    maxHeight: -1,
    minHeight: -1,
  }
);

const emit = defineEmits<{
  (name: "update:value", value: string | undefined): void;
}>();

const state = reactive<LocalState>({
  value: props.value,
});
const textareaRef = ref<HTMLTextAreaElement>();

useVModel(props, state, emit, "value");

const resize = () => {
  sizeToFit(textareaRef.value, props.padding, props.maxHeight);
};

const debouncedResize = debounce(resize, 50);

onMounted(() => {
  resize();
});

useEventListener("resize", debouncedResize);

watch(() => state.value, debouncedResize);
</script>
