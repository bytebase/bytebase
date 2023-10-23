<template>
  <div
    ref="container"
    class="droppable-textarea"
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
    />

    <div
      v-if="!state.value"
      class="absolute bottom-2 left-[50%] -translate-x-1/2 flex flex-col items-center justify-center border border-control-border hover:border-accent border-dashed text-xs text-control-placeholder hover:text-accent p-2 rounded-md"
    >
      Or drag and drop files here.
      <input
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
import { computed, reactive, ref, watch } from "vue";
import { useDropZone, useMutationObserver } from "@vueuse/core";
import { head } from "lodash-es";
import { useI18n } from "vue-i18n";

import { pushNotification } from "@/store";
import { BBSpin } from "@/bbkit";
import { onMounted } from "vue";
import { NInput } from "naive-ui";

type LocalState = {
  value: string | undefined;
  reading: boolean;
};

const props = withDefaults(
  defineProps<{
    value: string | undefined;
    placeholder?: string;
    maxFileSize?: number; // in MB
  }>(),
  {
    placeholder: undefined,
    maxFileSize: 1,
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

const onDrop = (files: File[] | FileList | null) => {
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
    const fr = new FileReader();
    fr.addEventListener("load", (e) => {
      emit("update:value", fr.result as string);
      state.reading = false;
    });
    fr.addEventListener("error", () => {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: `Read file error`,
        description: String(fr.error),
      });
      state.reading = false;
      return;
    });
    state.reading = true;
    fr.readAsText(file);
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
.droppable-textarea {
  @apply relative;
}
.droppable-textarea.reading {
  @apply pointer-events-none;
}
.droppable-textarea.drop-over :deep(.n-input__state-border),
.droppable-textarea.reading :deep(.n-input__state-border) {
  border-style: dashed;
}
</style>
