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
    <slot>
      <slot name="image">
        <svg
          class="mx-auto h-12 w-12 text-gray-400 pointer-events-none"
          stroke="currentColor"
          fill="none"
          viewBox="0 0 48 48"
          aria-hidden="true"
        >
          <path
            d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </slot>
      <div class="text-sm text-gray-600 inline-flex pointer-events-none">
        <span
          class="relative cursor-pointer rounded-md font-medium text-indigo-600 hover:text-indigo-500 focus-within:outline-none focus-within:ring-2 focus-within:ring-offset-2 focus-within:ring-indigo-500"
        >
          {{ $t("settings.general.workspace.select-logo") }}
        </span>
        <p class="pl-1">
          {{ $t("settings.general.workspace.drag-logo") }}
        </p>
      </div>
      <p class="text-xs text-gray-500 pointer-events-none">
        {{
          $t("settings.general.workspace.logo-upload-tip", {
            extension: supportFileExtensions.join(", "),
            size: maxFileSizeInMiB,
          })
        }}
      </p>
    </slot>
  </div>
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { ref, reactive } from "vue";
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
