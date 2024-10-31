<template>
  <div class="flex flex-col items-start gap-y-4 relative">
    <div class="w-full flex flex-col gap-2">
      <p class="textlabel !text-base">{{ $t("common.title") }}</p>
      <NInput v-model:value="title" class="w-full" size="large" />
    </div>
    <FileToCreateTable />
    <div class="w-full flex justify-end">
      <ErrorTipsButton
        style="--n-padding: 0 10px"
        :errors="createButtonErrors"
        :button-props="{
          type: 'primary',
        }"
        @click="onCreateRelease"
      >
        <template #icon>
          <PlusIcon />
        </template>
        {{ $t("common.create") }}
      </ErrorTipsButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NInput } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { ErrorTipsButton } from "@/components/v2";
import { releaseServiceClient } from "@/grpcweb";
import { pushNotification, useSheetV1Store } from "@/store";
import {
  Release,
  Release_File,
  ReleaseFileType,
} from "@/types/proto/v1/release_service";
import FileToCreateTable from "./FileToCreateTable";
import { provideReleaseCreateContext } from "./context";

const router = useRouter();
const { t } = useI18n();
const { title, files, project } = provideReleaseCreateContext();
const sheetStore = useSheetV1Store();

const createButtonErrors = computed(() => {
  const errors: string[] = [];
  if (title.value.trim() === "") {
    errors.push("Title is required");
  }
  if (files.value.length === 0) {
    errors.push("No files to create");
  }
  const duplicatedVersions = new Set<string>();
  for (const file of files.value) {
    if (duplicatedVersions.has(file.version)) {
      errors.push(`Duplicated version: ${file.version}`);
      break;
    }
    duplicatedVersions.add(file.version);
  }
  return errors;
});

const onCreateRelease = async () => {
  const release = Release.fromPartial({
    title: title.value,
  });
  for (const file of files.value) {
    const sheet = await sheetStore.createSheet(project.value.name, {
      title: file.path,
      content: new TextEncoder().encode(file.statement),
    });
    release.files.push(
      Release_File.fromPartial({
        path: file.path,
        version: file.version,
        sheet: sheet.name,
        type: ReleaseFileType.VERSIONED,
      })
    );
  }
  const created = await releaseServiceClient.createRelease({
    parent: project.value.name,
    release,
  });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("release.messages.succeed-to-create-release"),
  });
  // Redirect to the created release detail page.
  router.replace(`/${created.name}`);
};
</script>
