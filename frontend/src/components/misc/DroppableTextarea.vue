<template>
  <div
    ref="container"
    class="relative"
    :class="[state.reading && 'pointer-events-none']"
  >
    <textarea
      ref="textareaRef"
      v-model="state.value"
      class="textarea rounded-[3px]"
      :placeholder="placeholder"
      v-bind="$attrs"
    />

    <div
      v-if="!state.value"
      class="absolute bottom-2 left-[50%] -translate-x-1/2 flex flex-col items-center justify-center border border-control-border hover:border-control-hover border-dashed text-xs text-control-placeholder hover:text-control-hover p-2 rounded-md"
    >
      {{ borderRadius }}
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
      class="absolute inset-0 pointer-events-none flex flex-col items-center justify-center bg-white/50 border border-accent border-dashed"
      :style="{
        borderRadius,
      }"
    >
      <heroicons-outline:arrow-up-tray v-if="isOverDropZone" class="w-8 h-8" />
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
import { reactive, ref, watch } from "vue";
import { useDropZone, useMutationObserver } from "@vueuse/core";
import { head } from "lodash-es";
import { useI18n } from "vue-i18n";

import { pushNotification } from "@/store";
import { BBSpin } from "@/bbkit";
import { onMounted } from "vue";

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
const textareaRef = ref<HTMLTextAreaElement>();

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

const updateBorderRadius = (textarea: HTMLTextAreaElement) => {
  borderRadius.value = getComputedStyle(textarea).borderRadius;
};
useMutationObserver(
  textareaRef,
  (records) => {
    updateBorderRadius(records[0].target as HTMLTextAreaElement);
  },
  {
    attributeFilter: ["style", "class"],
    attributes: true,
  }
);
onMounted(() => {
  const textarea = textareaRef.value;
  if (textarea) {
    updateBorderRadius(textarea);
  }
});
</script>
