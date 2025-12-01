<template>
  <div
    ref="container"
    class="droppable-textarea relative"
    :class="{
      reading: state.reading,
      'drop-over': isOverDropZone,
    }"
  >
    <NInput
      ref="inputRef"
      v-model:value="state.value"
      :placeholder="placeholder"
      type="textarea"
      v-bind="$attrs"
      :disabled="disabled"
    />

    <div
      v-if="!state.value"
      class="absolute bottom-2 left-[50%] -translate-x-1/2 flex flex-col items-center justify-center border border-control-border hover:border-accent border-dashed text-xs text-control-placeholder hover:text-accent p-2 rounded-md"
    >
      Or drag and drop files here.
      <input
        v-if="!disabled"
        type="file"
        class="absolute inset-0 opacity-0 cursor-pointer"
        title=""
        @change="handleFileChange"
      />
    </div>

    <div
      v-if="isOverDropZone || state.reading"
      class="absolute inset-0 pointer-events-none flex flex-col items-center justify-center bg-white/50 border border-accent border-dashed text-accent"
      :style="{
        borderRadius,
      }"
    >
      <heroicons:arrow-up-tray v-if="isOverDropZone" class="w-8 h-8 color" />
      <BBSpin v-if="state.reading" />
    </div>
  </div>
</template>

<script lang="ts">
export default {
  inheritAttrs: false,
};
</script>

<script lang="ts" setup>
import { useDropZone, useMutationObserver } from "@vueuse/core";
import { head } from "lodash-es";
import { NInput } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { pushNotification } from "@/store";
import { readFileAsArrayBuffer } from "@/utils";

type LocalState = {
  value: string | undefined;
  reading: boolean;
};

const props = withDefaults(
  defineProps<{
    value: string | undefined;
    placeholder?: string;
    maxFileSize?: number; // in MB
    disabled?: boolean;
  }>(),
  {
    placeholder: undefined,
    maxFileSize: 1,
    disabled: false,
  }
);

const emit = defineEmits<{
  (name: "update:value", value: string | undefined): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  value: props.value,
  reading: false,
});
const container = ref<HTMLDivElement>();
const inputRef = ref<InstanceType<typeof NInput>>();
const inputWrapperRef = computed(
  () => inputRef.value?.wrapperElRef as HTMLDivElement | undefined
);
const borderRadius = ref("");

watch(
  () => props.value,
  (value) => (state.value = value)
);

watch(
  () => state.value,
  (value) => emit("update:value", value)
);

const onDrop = async (files: File[] | FileList | null) => {
  // called when files are dropped on zone
  const file = head(files);
  if (file) {
    const { maxFileSize } = props;
    if (maxFileSize > 0 && file.size > maxFileSize * 1024 * 1024) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("common.file-selector.size-limit", { size: maxFileSize }),
      });
      return;
    }

    state.reading = true;
    try {
      const { arrayBuffer } = await readFileAsArrayBuffer(file);
      const decoder = new TextDecoder("utf-8");
      const statement = decoder.decode(arrayBuffer);
      emit("update:value", statement);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: `Read file error`,
        description: String(error),
      });
    }
    state.reading = false;
  }
};

const handleFileChange = (e: Event) => {
  const target = e.target as HTMLInputElement;
  onDrop(target.files);
};

const { isOverDropZone } = useDropZone(container, onDrop);

const updateBorderRadius = (element: HTMLElement) => {
  borderRadius.value = getComputedStyle(element).borderRadius;
};
useMutationObserver(
  inputWrapperRef,
  (records) => {
    updateBorderRadius(records[0].target as HTMLElement);
  },
  {
    attributeFilter: ["style", "class"],
    attributes: true,
  }
);
onMounted(() => {
  const wrapper = inputWrapperRef.value;
  if (wrapper) {
    updateBorderRadius(wrapper);
  }
});
</script>

<style lang="postcss" scoped>
.droppable-textarea.reading {
  pointer-events: none;
}
.droppable-textarea.drop-over :deep(.n-input__state-border),
.droppable-textarea.reading :deep(.n-input__state-border) {
  border-style: dashed;
}
</style>
