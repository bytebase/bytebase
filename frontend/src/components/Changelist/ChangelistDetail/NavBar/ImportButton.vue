<template>
  <NUpload
    abstract
    accept="application/x-zip,.zip,application/sql,.sql"
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
import JSZip from "jszip";
import { orderBy } from "lodash-es";
import { UploadIcon } from "lucide-vue-next";
import {
  NButton,
  NTooltip,
  NUpload,
  NUploadTrigger,
  type UploadFileInfo,
} from "naive-ui";
import { useI18n } from "vue-i18n";
import { pushNotification, useChangelistStore, useSheetV1Store } from "@/store";
import {
  Changelist_Change as Change,
  Changelist,
} from "@/types/proto/v1/changelist_service";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { defer, setSheetStatement } from "@/utils";
import { useChangelistDetailContext } from "../context";

type ParsedFile = {
  name: string;
  content: string;
};

const { t } = useI18n();
const { changelist, project, isUpdating } = useChangelistDetailContext();

const unzip = async (file: File) => {
  const zip = await JSZip.loadAsync(file);
  const files = orderBy(zip.files, (f) => f.name, "asc").filter(
    (f) => !f.dir && f.name.toLowerCase().endsWith(".sql")
  );
  const results = await Promise.all(
    files.map<Promise<ParsedFile>>(async (f) => {
      const content = await f.async("string");
      return {
        name: f.name,
        content,
      };
    })
  );
  return results;
};

const readFile = (file: File) => {
  const d = defer<string>();
  const fr = new FileReader();
  fr.addEventListener("load", (e) => {
    const result = fr.result;
    if (typeof result === "string") {
      d.resolve(result);
      return;
    }
    d.reject(new Error("Failed to read file content."));
  });
  fr.addEventListener("error", (e) => {
    d.reject(fr.error ?? new Error("Failed to read file content."));
  });
  fr.readAsText(file);
  return d.promise;
};

const handleFileChange = async (options: { file: UploadFileInfo }) => {
  const fileInfo = options.file;
  if (!fileInfo.file) {
    return;
  }

  const cleanup = () => {
    isUpdating.value = false;
  };

  try {
    isUpdating.value = true;

    const files: ParsedFile[] = [];
    if (fileInfo.name.toLowerCase().endsWith(".sql")) {
      const content = await readFile(fileInfo.file);
      files.push({
        name: fileInfo.file.name,
        content,
      });
    } else {
      const results = await unzip(fileInfo.file);
      files.push(...results);
    }

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
    })
  );
  const changelistPatch = Changelist.fromPartial({
    ...changelist.value,
    changes: [...changelist.value.changes, ...newChanges],
  });
  await useChangelistStore().patchChangelist(changelistPatch, ["changes"]);
};
</script>
