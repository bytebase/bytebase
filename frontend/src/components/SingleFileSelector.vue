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
      <NEmpty v-if="showNoDataPlaceholder" class="py-4"></NEmpty>
      <div class="text-sm text-gray-600 inline-flex pointer-events-none">
        <span
          class="relative cursor-pointer rounded-md font-medium text-indigo-600 hover:text-indigo-500 focus-within:outline-hidden focus-within:ring-2 focus-within:ring-offset-2 focus-within:ring-indigo-500"
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
import { NEmpty } from "naive-ui";
import { reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

interface LocalState {
  dropAreaActive: boolean;
}

const props = defineProps<{
  maxFileSizeInMiB: number;
  disabled?: boolean;
  supportFileExtensions: string[];
  showNoDataPlaceholder: boolean;
}>();

const emit = defineEmits(["on-select"]);

const { t } = useI18n();

const state = reactive<LocalState>({
  dropAreaActive: false,
});

const uploader = ref<HTMLInputElement>();

const onUploaderClick = () => {
  uploader.value?.click();
};

const onFileChange = () => {
  selectFile(uploader.value?.files);
};

const onFileDrop = (e: DragEvent) => {
  e.preventDefault();
  state.dropAreaActive = false;

  if (props.disabled) {
    return;
  }

  const files = e.dataTransfer?.files;
  selectFile(files);
};

const selectFile = (files: FileList | null | undefined) => {
  if (!files || !files.length) {
    return;
  }

  const file = files.item(0);
  if (!file || !validFile(file)) {
    return;
  }
  emit("on-select", file);
  if (uploader.value) {
    uploader.value.value = "";
  }
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
