<template>
  <NUpload
    v-model:file-list="uploadFileList"
    abstract
    accept="application/x-zip,.zip,application/sql,.sql"
    :multiple="false"
    v-bind="$attrs"
    @change="handleFileChange"
  >
    <NUploadTrigger #="{ handleClick }" abstract>
      <slot name="trigger" @click="handleClick" />
    </NUploadTrigger>
  </NUpload>

  <BBModal
    :show="state.showModal"
    :title="$t('sql-editor.select-encoding')"
    @close="closeModal"
  >
    <div class="w-96">
      <div class="w-full flex flex-row justify-start items-center gap-2">
        <span class="textlabel">{{ $t("common.encoding") }}</span>
        <NSelect
          v-model:value="state.encoding"
          size="small"
          filterable
          :options="encodingOptions"
          :consistent-menu-width="false"
        />
      </div>
      <NUpload :file-list="uploadFileList" disabled trigger-class="!hidden">
      </NUpload>
    </div>
    <div class="w-full flex items-center justify-end mt-4 space-x-2">
      <NButton quaternary @click="closeModal">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton type="primary" @click="onConfirm">
        {{ $t("common.confirm") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import {
  NButton,
  NSelect,
  NUpload,
  NUploadTrigger,
  type UploadFileInfo,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import BBModal from "@/bbkit/BBModal.vue";
import { readUpload } from "@/components/Changelist/import";
import { pushNotification } from "@/store";
import { ENCODINGS, type Encoding } from "@/utils";

type ParsedFile = {
  name: string;
  arrayBuffer: ArrayBuffer;
};

interface LocalState {
  encoding: Encoding;
  files: ParsedFile[];
  showModal: boolean;
}

const emit = defineEmits<{
  (event: "update", value: { filename: string; statement: string }[]): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  encoding: "utf-8",
  files: [],
  showModal: false,
});
const uploadFileList = ref<UploadFileInfo[]>([]);

const encodingOptions = computed(() =>
  ENCODINGS.map((encoding) => ({
    label: encoding,
    value: encoding,
  }))
);

const cleanup = () => {
  uploadFileList.value = [];
  state.files = [];
};

const handleFileChange = async (options: { file: UploadFileInfo }) => {
  const files = await readUpload(options.file);
  if (files.length === 0) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("changelist.import.no-file-to-upload"),
    });
    uploadFileList.value = [];
  }
  state.files = files;
  state.showModal = true;
};

const onConfirm = async () => {
  emit(
    "update",
    state.files.map((f) => {
      return {
        filename: f.name,
        statement: new TextDecoder(state.encoding).decode(f.arrayBuffer),
      };
    })
  );
  closeModal();
};

const closeModal = () => {
  state.showModal = false;
  cleanup();
};
</script>
