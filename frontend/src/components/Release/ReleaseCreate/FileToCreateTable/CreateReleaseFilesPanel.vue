<template>
  <Drawer v-bind="$attrs" @close="emit('close')">
    <DrawerContent class="w-176" :title="$t('release.actions.new-file')">
      <template #default>
        <div class="w-full flex flex-col gap-4">
          <div v-for="f in state.files" :key="f.id" class="w-full relative">
            <ReleaseFileDetailPanel
              :class="isCreating && 'border p-4 rounded-lg'"
              :file="f"
              @update="
                (f) =>
                  (state.files = state.files.map((file) =>
                    file.id === f.id ? f : file
                  ))
              "
            />
            <NButton
              v-if="isCreating && state.files.length > 1"
              class="!absolute top-2 right-2"
              text
              @click="onDeleteFile(f)"
            >
              <template #icon>
                <XIcon class="hover:opacity-80" />
              </template>
            </NButton>
          </div>
        </div>
        <div
          v-if="isCreating"
          class="w-full flex flex-row justify-end mt-4 gap-2"
        >
          <UploadFilesButton @update="onUploadFiles">
            <template #trigger="{ onClick }">
              <NButton size="small" @click="onClick">
                <template #icon>
                  <UploadIcon class="w-4 h-4" />
                </template>
                {{ $t("common.upload") }}
              </NButton>
            </template>
          </UploadFilesButton>
          <NButton size="small" @click="onAddFile">
            <template #icon>
              <PlusIcon class="w-4 h-4" />
            </template>
            {{ $t("common.new") }}
          </NButton>
        </div>
      </template>

      <template #footer>
        <div class="w-full flex flex-row justify-between items-center">
          <div>
            <NButton
              v-if="!isCreating"
              type="error"
              text
              @click="onDeletePropsFile"
            >
              <template #icon>
                <TrashIcon class="w-4 h-auto" />
              </template>
              {{ $t("common.delete") }}
            </NButton>
          </div>

          <div class="flex flex-row justify-end items-center">
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
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { uniqueId } from "lodash-es";
import { TrashIcon, XIcon, PlusIcon, UploadIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { Drawer, DrawerContent, ErrorTipsButton } from "@/components/v2";
import { useReleaseCreateContext, type FileToCreate } from "../context";
import ReleaseFileDetailPanel from "./ReleaseFileDetailPanel.vue";
import UploadFilesButton from "./UploadFilesButton.vue";

interface LocalState {
  files: FileToCreate[];
}

const props = defineProps<{
  file?: FileToCreate;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const context = useReleaseCreateContext();
const state = reactive<LocalState>({
  files: [],
});

const isCreating = computed(() => props.file === undefined);

const nextButtonErrors = computed(() => {
  const errors: string[] = [];
  if (state.files.length === 0) {
    errors.push("No new files");
  } else {
    // Check all file's version and statement.
    for (const file of state.files) {
      if (!file.version) {
        errors.push("File version is required");
      }
      if (!file.statement) {
        errors.push("File statement is required");
      }
    }
  }
  return errors;
});

const onAddFile = () => {
  state.files.push({
    id: uniqueId("file"),
    version: "",
    path: "",
    statement: "",
  });
};

const onDeleteFile = (file: FileToCreate) => {
  state.files = state.files.filter((f) => f.id !== file.id);
};

const onDeletePropsFile = () => {
  if (props.file) {
    context.files.value = context.files.value.filter(
      (file) => file.id !== props.file?.id
    );
    emit("close");
  }
};

const onUploadFiles = (
  statementMap: { filename: string; statement: string }[]
) => {
  const files: FileToCreate[] = statementMap.map(({ filename, statement }) => ({
    id: uniqueId(),
    version: "",
    path: filename,
    statement,
  }));
  state.files = [...state.files, ...files];
};

const onConfirm = () => {
  if (!isCreating.value) {
    context.files.value = context.files.value.map((file) =>
      file.id === props.file?.id ? state.files[0] : file
    );
  } else {
    context.files.value = [...context.files.value, ...state.files];
  }
  emit("close");
};

watch(
  () => props.file,
  () => {
    state.files = [];
    if (props.file) {
      state.files.push(props.file);
    } else {
      onAddFile();
    }
  },
  {
    immediate: true,
  }
);
</script>
