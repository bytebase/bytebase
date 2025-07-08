<template>
  <UploadFilesButton @update="onUploadFiles">
    <template #trigger="{ onClick }">
      <NTooltip>
        <template #trigger>
          <NButton icon style="--n-padding: 0 10px" @click="onClick">
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
    </template>
  </UploadFilesButton>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { UploadIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { useI18n } from "vue-i18n";
import UploadFilesButton from "@/components/UploadFilesButton.vue";
import { pushNotification, useChangelistStore, useSheetV1Store } from "@/store";
import {
  Changelist_ChangeSchema,
  ChangelistSchema,
} from "@/types/proto-es/v1/changelist_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { setSheetStatement } from "@/utils";
import { useChangelistDetailContext } from "../context";

const { t } = useI18n();
const { changelist, project } = useChangelistDetailContext();

const onUploadFiles = async (
  statementMap: { filename: string; statement: string }[]
) => {
  const createdSheets = await Promise.all(
    statementMap.map(async (m) => {
      const sheet = create(SheetSchema, {
        title: m.filename,
      });
      setSheetStatement(sheet, m.statement);
      const created = await useSheetV1Store().createSheet(
        project.value.name,
        sheet
      );
      return created;
    })
  );
  const newChanges = createdSheets.map((sheet) =>
    create(Changelist_ChangeSchema, {
      sheet: sheet.name,
    })
  );
  const changelistPatch = create(ChangelistSchema, {
    ...changelist.value,
    changes: [...changelist.value.changes, ...newChanges],
  });
  await useChangelistStore().patchChangelist(changelistPatch, ["changes"]);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
