<template>
  <div class="flex flex-col items-start gap-y-4 relative">
    <div class="w-full flex flex-col gap-2">
      <p class="textlabel !text-base">{{ $t("common.title") }}</p>
      <NInput v-model:value="title" class="w-full" size="large" />
    </div>
    <ReleaseFileTable />
    <div class="w-full flex justify-end">
      <NButton type="primary" :disabled="!allowCreate" @click="onCreateRelease">
        <template #icon>
          <PlusIcon />
        </template>
        {{ $t("common.create") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton, NInput } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { releaseServiceClient } from "@/grpcweb";
import { pushNotification, useSheetV1Store } from "@/store";
import { Release, Release_File } from "@/types/proto/v1/release_service";
import ReleaseFileTable from "./ReleaseFileTable/ReleaseFileTable.vue";
import { provideReleaseCreateContext } from "./context";

const router = useRouter();
const { t } = useI18n();
const { title, files, project } = provideReleaseCreateContext();
const sheetStore = useSheetV1Store();

const allowCreate = computed(() => {
  if (title.value.trim() === "") {
    return false;
  }
  if (files.value.length === 0) {
    return false;
  }
  // TODO(steven): check file's version and statement in frontend.
  return true;
});

const onCreateRelease = async () => {
  const release = Release.fromPartial({
    title: title.value,
  });
  for (const file of files.value) {
    const sheet = await sheetStore.createSheet(project.value.name, {
      title: file.name,
      content: new TextEncoder().encode(file.statement),
    });
    release.files.push(
      Release_File.fromPartial({
        name: file.name,
        version: file.version,
        sheet: sheet.name,
        type: file.type,
      })
    );
  }
  const created = await releaseServiceClient.createRelease({
    parent: project.value.name,
    release,
  });

  pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: t("release.messages.succeed-to-create-release"),
  });
  // Redirect to the created release detail page.
  router.replace(`/${created.name}`);
};
</script>
