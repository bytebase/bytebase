<template>
  <NUpload
    v-model:file-list="uploadFileList"
    abstract
    accept="application/x-zip,.zip,application/sql,.sql"
    :multiple="false"
    @change="handleFileChange"
  >
    <NUploadTrigger #="{ handleClick }" abstract>
      <NTooltip>
        <template #trigger>
          <NButton icon style="--n-padding: 0 10px" @click="handleClick">
            <template #icon>
              <UploadIcon class="w-4 h-4" />
            </template>
          </NButton>
        </template>
        <template #default>
          <div class="whitespace-nowrap">
            {{ $t("changelist.import.upload-sql-or-zip-file") }}
          </div>
        </template>
      </NTooltip>
    </NUploadTrigger>
  </NUpload>

  <BBModal
    :show="state.showModal"
    :title="$t('sql-editor.select-encoding')"
    class="shadow-inner outline outline-gray-200"
    @close="closeModal"
  >
    <div class="w-80">
      <div class="w-full flex flex-row justify-start items-center gap-2">
        <span class="textinfolabel">{{ $t("common.encoding") }}</span>
        <NSelect
          v-model:value="state.encoding"
          class="!w-24"
          filterable
          :options="encodingOptions"
          :consistent-menu-width="false"
        />
      </div>
      <div>
        <NUpload :file-list="uploadFileList" disabled trigger-class="!hidden">
        </NUpload>
      </div>
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3">
      <NButton @click="closeModal">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton type="primary" @click="save">
        {{ $t("common.save") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { UploadIcon } from "lucide-vue-next";
import {
  NButton,
  NSelect,
  NTooltip,
  NUpload,
  NUploadTrigger,
  type UploadFileInfo,
} from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import BBModal from "@/bbkit/BBModal.vue";
import { pushNotification, useChangelistStore, useSheetV1Store } from "@/store";
import {
  Changelist_Change as Change,
  Changelist,
} from "@/types/proto/v1/changelist_service";
import { Engine } from "@/types/proto/v1/common";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { ENCODINGS, setSheetStatement, type Encoding } from "@/utils";
import { fallbackVersionForChange } from "../../common";
import { readUpload, type ParsedFile } from "../../import";
import { useChangelistDetailContext } from "../context";

interface LocalState {
  encoding: Encoding;
  files: ParsedFile[];
  showModal: boolean;
}

const { t } = useI18n();
const { changelist, project, isUpdating } = useChangelistDetailContext();
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
  isUpdating.value = false;
  uploadFileList.value = [];
  state.files = [];
};

const handleFileChange = async (options: { file: UploadFileInfo }) => {
  isUpdating.value = true;
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

const save = async () => {
  isUpdating.value = true;
  const createdSheets = await Promise.all(
    state.files.map(async (f) => {
      const { name, arrayBuffer } = f;
      const sheet = Sheet.fromPartial({
        title: name,
        engine: Engine.ENGINE_UNSPECIFIED, // TODO(jim)
      });
      const content = new TextDecoder(state.encoding).decode(arrayBuffer);
      setSheetStatement(sheet, content);
      const created = await useSheetV1Store().createSheet(
        project.value.name,
        sheet
      );
      return created;
    })
  );
  const newChanges = createdSheets.map((sheet) =>
    Change.fromPartial({
      sheet: sheet.name,
      version: fallbackVersionForChange(),
    })
  );
  const changelistPatch = Changelist.fromPartial({
    ...changelist.value,
    changes: [...changelist.value.changes, ...newChanges],
  });
  await useChangelistStore().patchChangelist(changelistPatch, ["changes"]);
  closeModal();
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const closeModal = () => {
  state.showModal = false;
  cleanup();
};
</script>
