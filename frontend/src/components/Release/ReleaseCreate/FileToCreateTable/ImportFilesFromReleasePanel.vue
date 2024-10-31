<template>
  <Drawer v-bind="$attrs" @close="emit('close')">
    <DrawerContent
      class="w-192"
      :title="$t('release.actions.import-from-release')"
    >
      <template #default>
        <div class="w-full mb-4 flex flex-col gap-2">
          <p class="textlabel">
            {{ $t("release.select") }}
          </p>
          <NSelect
            v-model:value="state.selectedReleaseName"
            :options="releaseOptions"
            :consistent-menu-width="false"
          />
        </div>

        <div v-if="selectedRelease" class="w-full flex flex-col gap-2">
          <p class="textlabel">
            {{ $t("release.files") }}
          </p>
          <ReleaseFileTable
            :files="selectedRelease.files"
            :show-selection="true"
            :row-clickable="false"
            @update:selected-files="state.selectedFiles = $event"
          />
        </div>
      </template>

      <template #footer>
        <div class="w-full flex flex-row justify-end items-center">
          <ErrorTipsButton
            style="--n-padding: 0 10px"
            :errors="nextButtonErrors"
            :button-props="{
              type: 'primary',
            }"
            @click="onConfirm"
          >
            {{ $t("common.next") }}
          </ErrorTipsButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { uniqueId } from "lodash-es";
import { NSelect } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { Drawer, DrawerContent, ErrorTipsButton } from "@/components/v2";
import { useReleaseStore, useSheetV1Store } from "@/store";
import { Release_File } from "@/types/proto/v1/release_service";
import ReleaseFileTable from "../../ReleaseDetail/ReleaseFileTable/ReleaseFileTable.vue";
import { useReleaseCreateContext, type FileToCreate } from "../context";

interface LocalState {
  selectedReleaseName?: string;
  selectedFiles?: Release_File[];
}

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { project, files } = useReleaseCreateContext();
const releaseStore = useReleaseStore();
const sheetStore = useSheetV1Store();
const state = reactive<LocalState>({});

onMounted(async () => {
  await releaseStore.fetchReleasesByProject(project.value.name);
});

const releaseOptions = computed(() => {
  return releaseStore
    .getReleasesByProject(project.value.name)
    .map((release) => ({
      label: release.title,
      value: release.name,
    }));
});

const selectedRelease = computed(() => {
  if (!state.selectedReleaseName) return undefined;
  return releaseStore.getReleaseByName(state.selectedReleaseName);
});

const nextButtonErrors = computed(() => {
  const errors: string[] = [];
  if (!state.selectedFiles || state.selectedFiles?.length === 0) {
    errors.push("No new files");
  }
  return errors;
});

const onConfirm = async () => {
  const filesToCreate: FileToCreate[] = [];
  for (const file of state.selectedFiles!) {
    const sheet = await sheetStore.getOrFetchSheetByName(file.sheet, "FULL");
    if (!sheet) continue;
    filesToCreate.push({
      id: uniqueId(),
      version: file.version,
      path: file.path,
      statement: new TextDecoder().decode(sheet.content),
    });
  }
  if (filesToCreate.length > 0) {
    files.value.push(...filesToCreate);
    emit("close");
  }
};
</script>
