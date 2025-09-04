<template>
  <Drawer :show="showCreatePanel" @close="showCreatePanel = false">
    <DrawerContent
      :title="$t('changelist.new')"
      class="w-[40rem] max-w-[100vw] relative"
    >
      <template #default>
        <div
          class="grid items-center gap-y-4 gap-x-4"
          style="grid-template-columns: minmax(6rem, auto) 1fr"
        >
          <div v-if="!disableProjectSelect" class="contents">
            <div class="textlabel">
              {{ $t("common.project") }}
              <span class="ml-0.5 text-error">*</span>
            </div>
            <div>
              <ProjectSelect
                v-model:project-name="projectName"
                :include-all="false"
                style="width: 14rem"
              />
            </div>
          </div>
          <div class="contents">
            <div class="textlabel">
              {{ $t("changelist.name") }}
              <span class="ml-0.5 text-error">*</span>
            </div>
            <div>
              <NInput
                v-model:value="title"
                :placeholder="$t('changelist.name-placeholder')"
                style="width: 14rem"
              />
            </div>
          </div>
          <div class="contents">
            <div class="col-span-2">
              <ResourceIdField
                ref="resourceIdField"
                v-model:value="resourceId"
                resource-type="changelist"
                :resource-title="title"
                :fetch-resource="
                  (resourceId: ResourceId) =>
                    changelistStore.getOrFetchChangelistByName(
                      `${projectName}/changelists/${resourceId}`,
                      true /* silent */
                    )
                "
              />
            </div>
          </div>
          <div class="contents file-upload">
            <div class="col-span-2 flex flex-col gap-1">
              <NUpload
                v-model:file-list="uploadFileList"
                abstract
                accept="application/x-zip,.zip,application/sql,.sql"
                :multiple="false"
                @change="uploadFileList = [$event.file]"
              >
                <div
                  class="w-full flex flex-row justify-start items-center gap-2"
                >
                  <NUploadTrigger #="{ handleClick }" abstract>
                    <NButton
                      icon
                      style="--n-padding: 0 10px"
                      class="self-start"
                      :loading="isParsingUploadFile"
                      @click="handleClick"
                    >
                      <template #icon>
                        <UploadIcon class="w-4 h-4" />
                      </template>
                      {{
                        $t("changelist.import.optional-upload-sql-or-zip-file")
                      }}
                    </NButton>
                  </NUploadTrigger>
                  <NSelect
                    v-if="uploadFileList.length > 0"
                    v-model:value="state.encoding"
                    class="!w-24"
                    filterable
                    :options="encodingOptions"
                    :consistent-menu-width="false"
                  />
                </div>
                <div class="flex flex-col gap-1">
                  <NUploadFileList />
                </div>
              </NUpload>
              <div class="flex flex-col gap-1 pl-7 text-xs">
                <div v-for="(file, i) in files" :key="`${file.name}-${i}`">
                  {{ file.name }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div
          v-if="isLoading"
          v-zindexable="{ enabled: true }"
          class="absolute bg-white/50 inset-0 flex flex-col items-center justify-center"
        >
          <BBSpin />
        </div>
      </template>

      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton @click="showCreatePanel = false">{{
            $t("common.cancel")
          }}</NButton>
          <NTooltip :disabled="errors.length === 0">
            <template #trigger>
              <NButton
                type="primary"
                tag="div"
                :disabled="errors.length > 0"
                @click="doCreate"
              >
                {{ $t("common.add") }}
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="errors" />
            </template>
          </NTooltip>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { create as createProto } from "@bufbuild/protobuf";
import { asyncComputed } from "@vueuse/core";
import { UploadIcon } from "lucide-vue-next";
import {
  NButton,
  NInput,
  NTooltip,
  NUpload,
  NUploadFileList,
  NUploadTrigger,
  type UploadFileInfo,
  NSelect,
} from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import ErrorList from "@/components/misc/ErrorList.vue";
import {
  Drawer,
  DrawerContent,
  ProjectSelect,
  ResourceIdField,
} from "@/components/v2";
import { pushNotification, useChangelistStore, useSheetV1Store } from "@/store";
import type { ResourceId } from "@/types";
import {
  ChangelistSchema,
  Changelist_ChangeSchema as ChangeSchema,
  CreateChangelistRequestSchema,
} from "@/types/proto-es/v1/changelist_service_pb";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import {
  ENCODINGS,
  extractChangelistResourceName,
  setSheetStatement,
  type Encoding,
} from "@/utils";
import { readUpload, type ParsedFile } from "../import";
import { useChangelistDashboardContext } from "./context";

interface LocalState {
  encoding: Encoding;
}

const props = defineProps<{
  project: Project;
  disableProjectSelect?: boolean;
}>();

const router = useRouter();
const { t } = useI18n();
const { showCreatePanel, events } = useChangelistDashboardContext();
const state = reactive<LocalState>({
  encoding: "utf-8",
});

const title = ref("");
const projectName = ref<string>(props.project.name);
const isLoading = ref(false);
const resourceId = ref("");
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();
const changelistStore = useChangelistStore();

const encodingOptions = computed(() =>
  ENCODINGS.map((encoding) => ({
    label: encoding,
    value: encoding,
  }))
);

const errors = asyncComputed(() => {
  const errors: string[] = [];
  if (!title.value.trim()) {
    errors.push(t("changelist.error.name-is-required"));
  }
  if (resourceIdField.value && !resourceIdField.value.isValidated) {
    errors.push(t("changelist.error.invalid-resource-id"));
  }

  return errors;
}, []);

const uploadFileList = ref<UploadFileInfo[]>([]);
const files = ref<ParsedFile[]>([]);
const isParsingUploadFile = ref(false);

watch(uploadFileList, async (fileList) => {
  const file = fileList[0];
  if (!file) {
    files.value = [];
    return;
  }
  const cleanup = () => {
    isParsingUploadFile.value = false;
  };

  try {
    files.value = [];
    isParsingUploadFile.value = true;
    files.value = await readUpload(file);

    if (files.value.length === 0) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: t("changelist.import.no-file-to-upload"),
      });
      return cleanup();
    }

    if (!title.value) {
      title.value = file.name;
    }
  } finally {
    cleanup();
  }
});

const doCreate = async () => {
  if (errors.value.length > 0) return;
  if (!resourceIdField.value) return;

  isLoading.value = true;
  try {
    const createdSheets = await Promise.all(
      files.value.map(async (f) => {
        const { name, arrayBuffer } = f;
        const sheet = createProto(SheetSchema, {
          title: name,
          engine: Engine.ENGINE_UNSPECIFIED,
        });
        const content = new TextDecoder(state.encoding).decode(arrayBuffer);
        setSheetStatement(sheet, content);
        const created = await useSheetV1Store().createSheet(
          projectName.value,
          sheet
        );
        return created;
      })
    );
    const changes = createdSheets.map((sheet) =>
      createProto(ChangeSchema, {
        sheet: sheet.name,
      })
    );

    const created = await useChangelistStore().createChangelist(
      createProto(CreateChangelistRequestSchema, {
        parent: projectName.value,
        changelist: createProto(ChangelistSchema, {
          description: title.value,
          changes,
        }),
        changelistId: resourceId.value,
      })
    );
    showCreatePanel.value = false;
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });

    router.push(
      `/${projectName.value}/changelists/${extractChangelistResourceName(
        created.name
      )}`
    );
    events.emit("refresh");
  } finally {
    isLoading.value = false;
  }
};

const reset = () => {
  title.value = "";
  projectName.value = props.project?.name;
};

watch(showCreatePanel, (show) => {
  if (show) {
    reset();
  }
});
</script>

<style scoped lang="postcss">
.file-upload :deep(.n-upload-file-list .n-upload-file .n-upload-file-info) {
  @apply items-center;
}
.file-upload
  :deep(
    .n-upload-file-list
      .n-upload-file
      .n-upload-file-info
      .n-upload-file-info__thumbnail
  ) {
  @apply flex items-center justify-center mr-0.5;
}
</style>
