<template>
  <div
    class="transition-all hover:bg-gray-100"
    :class="[
      state.dropAreaActive ? 'bg-gray-300 opacity-100' : '',
      disabled ? 'cursor-not-allowed' : 'cursor-pointer',
    ]"
    @click="onUploaderClick"
    @drop="onFileDrop"
    @dragover.prevent
    @dragenter="state.dropAreaActive = true"
    @dragleave="state.dropAreaActive = false"
  >
    <input
      ref="uploader"
      name="uploader"
      type="file"
      :accept="supportFileExtensions.join(',')"
      class="sr-only hidden"
      :disabled="disabled"
      @input="onFileChange"
    />
    <slot></slot>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

interface LocalState {
  dropAreaActive: boolean;
}

const props = defineProps({
  maxFileSizeInMiB: {
    required: true,
    type: Number,
  },
  disabled: {
    required: false,
    type: Boolean,
    default: false,
  },
  supportFileExtensions: {
    required: true,
    type: Object as PropType<string[]>,
  },
});

const emit = defineEmits(["on-select"]);

const { t } = useI18n();

const state = reactive<LocalState>({
  dropAreaActive: false,
});

const uploader = ref<HTMLInputElement | null>(null);

const onUploaderClick = () => {
  uploader.value?.click();
};

const onFileChange = () => {
  const files: File[] = (uploader.value as any).files;
  selectFile(files);
};

const onFileDrop = (e: any) => {
  e.preventDefault();
  state.dropAreaActive = false;

  if (props.disabled) {
    return;
  }

  const files: File[] = e.dataTransfer.files;
  selectFile(files);
};

const selectFile = (files: File[]) => {
  if (!files.length) {
    return;
  }

  const file = files[0];
  if (!validFile(file)) {
    return;
  }
  emit("on-select", file);
};

const validFile = (file: File): boolean => {
  const extension = file.name.toLowerCase().split(".").slice(-1)[0];
  if (!props.supportFileExtensions.includes(`.${extension}`)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.file-selector.type-limit", {
        extension: props.supportFileExtensions.join(", "),
      }),
    });
    return false;
  }

  if (file.size > props.maxFileSizeInMiB * 1024 * 1024) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.file-selector.size-limit", {
        size: props.maxFileSizeInMiB,
      }),
    });
    return false;
  }

  return true;
};
</script>
