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
</template>

<script setup lang="ts">
import { UploadIcon } from "lucide-vue-next";
import {
  NButton,
  NTooltip,
  NUpload,
  NUploadTrigger,
  type UploadFileInfo,
} from "naive-ui";
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useChangelistStore, useSheetV1Store } from "@/store";
import {
  Changelist_Change as Change,
  Changelist,
} from "@/types/proto/v1/changelist_service";
import { Engine } from "@/types/proto/v1/common";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { setSheetStatement } from "@/utils";
import { readUpload, type ParsedFile } from "../../import";
import { useChangelistDetailContext } from "../context";
import { fallbackVersionForChange } from "../../common";

const { t } = useI18n();
const { changelist, project, isUpdating } = useChangelistDetailContext();
const uploadFileList = ref<UploadFileInfo[]>([]);

const handleFileChange = async (options: { file: UploadFileInfo }) => {
  const cleanup = () => {
    isUpdating.value = false;
    uploadFileList.value = [];
  };

  try {
    isUpdating.value = true;
    const files = await readUpload(options.file);

    if (files.length === 0) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("changelist.import.no-file-to-upload"),
      });
      return cleanup();
    }

    await save(files);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    cleanup();
  }
};

const save = async (files: ParsedFile[]) => {
  const createdSheets = await Promise.all(
    files.map(async (f) => {
      const { name, content } = f;
      const sheet = Sheet.fromPartial({
        title: name,
        engine: Engine.ENGINE_UNSPECIFIED, // TODO(jim)
      });
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
      version: fallbackVersionForChange()
    })
  );
  const changelistPatch = Changelist.fromPartial({
    ...changelist.value,
    changes: [...changelist.value.changes, ...newChanges],
  });
  await useChangelistStore().patchChangelist(changelistPatch, ["changes"]);
};
</script>
