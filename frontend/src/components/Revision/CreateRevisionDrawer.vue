<template>
  <Drawer
    v-model:show="show"
    :mask-closable="true"
    placement="right"
    :default-width="1024"
    :resizable="true"
    :width="undefined"
    class="!max-w-[90vw]"
  >
    <DrawerContent :title="$t('database.revision.import-revision')" closable>
      <div class="flex flex-col gap-y-4">
        <!-- Steps indicator -->
        <NSteps :current="currentStep">
          <NStep :title="$t('database.revision.select-source')" />
          <NStep :title="$t('database.revision.select-release')" />
          <NStep :title="$t('database.revision.select-files')" />
        </NSteps>

        <!-- Step content -->
        <div class="flex-1">
          <!-- Step 1: Select Source -->
          <template v-if="currentStep === Step.SELECT_SOURCE">
            <NRadioGroup
              v-model:value="selectedSource"
              size="large"
              class="space-y-4 w-full"
            >
              <div
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                :class="{
                  'border-blue-500 bg-blue-50': selectedSource === 'release',
                }"
              >
                <NRadio value="release" class="w-full">
                  <div class="flex items-start space-x-3 w-full">
                    <PackageIcon
                      class="w-6 h-6 mt-1 flex-shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center space-x-2">
                        <span class="text-lg font-medium text-gray-900">
                          {{ $t("database.revision.from-release") }}
                        </span>
                      </div>
                      <p class="text-sm text-gray-600 mt-1">
                        {{ $t("database.revision.from-release-description") }}
                      </p>
                    </div>
                  </div>
                </NRadio>
              </div>
              <div
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors opacity-50 cursor-not-allowed"
              >
                <NRadio value="local" disabled class="w-full">
                  <div class="flex items-start space-x-3 w-full">
                    <FolderOpenIcon
                      class="w-6 h-6 mt-1 flex-shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center space-x-2">
                        <span class="text-lg font-medium text-gray-900">
                          {{ $t("database.revision.from-local-files") }}
                        </span>
                        <NBadge type="info">
                          {{ "Coming soon" }}
                        </NBadge>
                      </div>
                      <p class="text-sm text-gray-600 mt-1">
                        {{
                          $t("database.revision.from-local-files-description")
                        }}
                      </p>
                    </div>
                  </div>
                </NRadio>
              </div>
            </NRadioGroup>
          </template>

          <!-- Step 2: Select Release -->
          <template v-else-if="currentStep === Step.SELECT_RELEASE">
            <div class="space-y-4">
              <div class="text-sm text-gray-600">
                {{ $t("database.revision.select-release-description") }}
              </div>
              <ReleaseDataTable
                v-if="releaseList.length > 0"
                :release-list="releaseList"
                :show-selection="true"
                :single-select="true"
                :loading="isLoadingReleases"
                :selected-row-keys="
                  selectedRelease ? [selectedRelease.name] : []
                "
                bordered
                @update:checked-row-keys="handleReleaseSelection"
              />
              <div v-else-if="!isLoadingReleases" class="text-center py-8">
                <PackageIcon class="w-12 h-12 mx-auto text-gray-400 mb-4" />
                <p class="text-gray-500">
                  {{ $t("database.revision.no-releases-found") }}
                </p>
              </div>
            </div>
          </template>

          <!-- Step 3: Select Files -->
          <template v-else-if="currentStep === Step.SELECT_FILES">
            <div class="space-y-4">
              <div class="text-sm text-gray-600">
                {{ $t("database.revision.select-files-description") }}
              </div>

              <!-- Selectable files section -->
              <div v-if="selectableFiles.length > 0" class="space-y-2">
                <div>
                  <h4 class="font-medium text-control">
                    {{ $t("database.revision.available-files") }}
                    <span v-if="selectableFiles.length > 1">
                      ({{ selectableFiles.length }})
                    </span>
                  </h4>
                  <p class="text-sm text-control-light mt-1">
                    {{ $t("database.revision.available-files-description") }}
                  </p>
                </div>
                <ReleaseFileTable
                  :files="selectableFiles"
                  :show-selection="true"
                  :selected-files="selectedFiles"
                  @update:selected-files="handleFileSelection"
                />
              </div>

              <!-- Files with existing versions section -->
              <div
                v-if="filesWithExistingVersions.length > 0"
                class="space-y-2 mt-6"
              >
                <div>
                  <h4 class="text font-medium text-control">
                    {{ $t("database.revision.files-already-imported") }}
                    <span
                      v-if="filesWithExistingVersions.length > 1"
                      class="text-control-light"
                      >({{ filesWithExistingVersions.length }})</span
                    >
                  </h4>
                  <p class="text-sm text-control-light mt-1">
                    {{
                      $t("database.revision.files-already-imported-description")
                    }}
                  </p>
                </div>
                <ReleaseFileTable
                  :files="filesWithExistingVersions"
                  :show-selection="false"
                  :row-clickable="false"
                />
              </div>

              <!-- No files message -->
              <div
                v-if="!selectedRelease || selectedRelease.files.length === 0"
                class="text-center py-8"
              >
                <PackageIcon class="w-12 h-12 mx-auto text-gray-400 mb-4" />
                <p class="text-gray-500">
                  {{ $t("database.revision.no-files-found") }}
                </p>
              </div>
            </div>
          </template>
        </div>
      </div>
      <template #footer>
        <div class="w-full flex items-center justify-between">
          <div></div>
          <div class="flex items-center gap-x-3">
            <NButton
              quaternary
              v-if="currentStep === Step.SELECT_SOURCE"
              @click="handleCancel"
            >
              {{ $t("common.close") }}
            </NButton>
            <NButton
              v-if="currentStep > Step.SELECT_SOURCE"
              quaternary
              @click="handlePrevStep"
            >
              {{ $t("common.back") }}
            </NButton>
            <NButton
              v-if="!isLastStep"
              type="primary"
              :disabled="!canProceedToNextStep"
              @click="handleNextStep"
            >
              {{ $t("common.next") }}
            </NButton>
            <NButton
              v-else
              type="primary"
              :disabled="!canSubmit"
              :loading="isCreating"
              @click="handleConfirm"
            >
              {{ $t("common.confirm") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { create as createProto } from "@bufbuild/protobuf";
import { PackageIcon, FolderOpenIcon } from "lucide-vue-next";
import { NButton, NRadio, NRadioGroup, NSteps, NStep, NBadge } from "naive-ui";
import type { Ref } from "vue";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import ReleaseDataTable from "@/components/Release/ReleaseDataTable.vue";
import ReleaseFileTable from "@/components/Release/ReleaseDetail/ReleaseFileTable/ReleaseFileTable.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { revisionServiceClientConnect } from "@/grpcweb";
import {
  useCurrentProjectV1,
  useReleaseStore,
  useRevisionStore,
  pushNotification,
} from "@/store";
import type { ComposedRelease } from "@/types";
import {
  Release_File_Type,
  type Release_File,
} from "@/types/proto-es/v1/release_service_pb";
import {
  BatchCreateRevisionsRequestSchema,
  CreateRevisionRequestSchema,
  Revision_Type,
  type Revision,
} from "@/types/proto-es/v1/revision_service_pb";

const props = defineProps<{
  database: string;
}>();

const emit = defineEmits<{
  (event: "created", revisions: Revision[]): void;
}>();

enum Step {
  SELECT_SOURCE = 1,
  SELECT_RELEASE = 2,
  SELECT_FILES = 3,
}

const { project } = useCurrentProjectV1();
const { t } = useI18n();
const releaseStore = useReleaseStore();
const revisionStore = useRevisionStore();
const show = defineModel<boolean>("show", { default: false });

const selectedSource: Ref<"release" | "local"> = ref("release");
const isCreating = ref(false);
const currentStep = ref(Step.SELECT_SOURCE);
const isLoadingReleases = ref(false);
const releaseList = ref<ComposedRelease[]>([]);
const selectedRelease = ref<ComposedRelease | null>(null);
const selectedFiles = ref<Release_File[]>([]);
const existingRevisionVersions = ref<Set<string>>(new Set());

const canProceedToNextStep = computed(() => {
  if (currentStep.value === Step.SELECT_SOURCE) {
    return !!selectedSource.value;
  }
  if (currentStep.value === Step.SELECT_RELEASE) {
    return !!selectedRelease.value;
  }
  return false;
});

const isLastStep = computed(() => {
  return currentStep.value === Step.SELECT_FILES;
});

const canSubmit = computed(() => {
  return selectedFiles.value.length > 0;
});

// Split files into selectable and non-selectable groups
const selectableFiles = computed(() => {
  if (!selectedRelease.value) return [];
  return selectedRelease.value.files.filter((file) => isFileSelectable(file));
});

const filesWithExistingVersions = computed(() => {
  if (!selectedRelease.value) return [];
  return selectedRelease.value.files.filter((file) => !isFileSelectable(file));
});

// Load releases when entering step 2
const loadReleases = async () => {
  isLoadingReleases.value = true;
  try {
    await releaseStore.fetchReleasesByProject(project.value.name, {
      pageSize: 100,
    });
    releaseList.value = releaseStore.getReleasesByProject(project.value.name);
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: "Failed to load releases",
    });
  } finally {
    isLoadingReleases.value = false;
  }
};

// Load existing revisions to check for duplicates
const loadExistingRevisions = async () => {
  try {
    const response = await revisionStore.fetchRevisionsByDatabase(
      props.database,
      { pageSize: 1000 }
    );
    const versions = new Set<string>();
    response.revisions.forEach((revision) => {
      if (revision.version) {
        versions.add(revision.version);
      }
    });
    existingRevisionVersions.value = versions;
  } catch (error) {
    console.error("Failed to load existing revisions:", error);
  }
};

// Check if a file can be selected (no existing revision with same version)
const isFileSelectable = (file: Release_File): boolean => {
  return !existingRevisionVersions.value.has(file.version);
};

// Map release file type to revision type
const mapFileTypeToRevisionType = (
  fileType: Release_File_Type
): Revision_Type => {
  switch (fileType) {
    case Release_File_Type.VERSIONED:
      return Revision_Type.VERSIONED;
    case Release_File_Type.DECLARATIVE:
      return Revision_Type.DECLARATIVE;
    default:
      return Revision_Type.TYPE_UNSPECIFIED;
  }
};

// Reset state when drawer opens
watch(show, async (newVal) => {
  if (newVal) {
    currentStep.value = Step.SELECT_SOURCE;
    selectedSource.value = "release";
    isCreating.value = false;
    releaseList.value = [];
    selectedRelease.value = null;
    selectedFiles.value = [];
    existingRevisionVersions.value = new Set();
    // Load existing revisions to check for duplicates
    await loadExistingRevisions();
  }
});

const handleReleaseSelection = (selectedKeys: string[]) => {
  if (selectedKeys.length > 0) {
    const releaseName = selectedKeys[0];
    selectedRelease.value =
      releaseList.value.find((r) => r.name === releaseName) || null;
  } else {
    selectedRelease.value = null;
  }
};

const handleFileSelection = (files: Release_File[]) => {
  selectedFiles.value = files;
};

const handleCancel = () => {
  show.value = false;
};

const handleNextStep = async () => {
  if (currentStep.value === Step.SELECT_SOURCE) {
    currentStep.value = Step.SELECT_RELEASE;
    await loadReleases();
  } else if (
    currentStep.value === Step.SELECT_RELEASE &&
    selectedRelease.value
  ) {
    currentStep.value = Step.SELECT_FILES;
    // Reset file selection when moving to file selection step
    selectedFiles.value = [];
  }
};

const handlePrevStep = () => {
  if (currentStep.value === Step.SELECT_RELEASE) {
    currentStep.value = Step.SELECT_SOURCE;
  } else if (currentStep.value === Step.SELECT_FILES) {
    currentStep.value = Step.SELECT_RELEASE;
  }
};

const handleConfirm = async () => {
  if (!canSubmit.value || !selectedRelease.value) return;

  isCreating.value = true;
  try {
    // Create revision requests using the existing sheet from release files
    const requests = selectedFiles.value.map((file) =>
      createProto(CreateRevisionRequestSchema, {
        parent: props.database,
        revision: {
          release: selectedRelease.value!.name,
          version: file.version,
          file: `${selectedRelease.value!.name}/files/${file.id}`,
          sheet: file.sheet,
          type: mapFileTypeToRevisionType(file.type),
        },
      })
    );

    const batchRequest = createProto(BatchCreateRevisionsRequestSchema, {
      parent: props.database,
      requests,
    });

    const response =
      await revisionServiceClientConnect.batchCreateRevisions(batchRequest);

    emit("created", response.revisions);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: "Imported revisions successfully",
    });

    show.value = false;
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: error instanceof Error ? error.message : String(error),
    });
  } finally {
    isCreating.value = false;
  }
};
</script>
