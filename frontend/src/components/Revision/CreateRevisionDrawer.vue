<template>
  <Drawer
    v-model:show="show"
    :mask-closable="true"
    placement="right"
    :default-width="1024"
    :resizable="true"
    :width="undefined"
    class="max-w-[90vw]!"
  >
    <DrawerContent :title="$t('database.revision.import-revision')" closable>
      <div class="flex flex-col gap-y-4">
        <!-- Steps indicator -->
        <NSteps :current="currentStep">
          <NStep :title="$t('database.revision.select-source')" />
          <NStep
            v-if="selectedSource === 'release'"
            :title="$t('database.revision.select-release')"
          />
          <NStep
            v-if="selectedSource === 'release'"
            :title="$t('database.revision.select-files')"
          />
          <NStep
            v-if="selectedSource === 'local'"
            :title="$t('database.revision.upload-files')"
          />
        </NSteps>

        <!-- Step content -->
        <div class="flex-1">
          <!-- Step 1: Select Source -->
          <template v-if="currentStep === Step.SELECT_SOURCE">
            <NRadioGroup
              v-model:value="selectedSource"
              size="large"
              class="flex flex-col gap-y-4 w-full"
            >
              <div
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                :class="{
                  'border-blue-500 bg-blue-50': selectedSource === 'release',
                }"
              >
                <NRadio value="release" class="w-full">
                  <div class="flex items-start gap-x-3 w-full">
                    <PackageIcon
                      class="w-6 h-6 mt-1 shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center gap-x-2">
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
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
                :class="{
                  'border-blue-500 bg-blue-50': selectedSource === 'local',
                }"
              >
                <NRadio value="local" class="w-full">
                  <div class="flex items-start gap-x-3 w-full">
                    <FolderOpenIcon
                      class="w-6 h-6 mt-1 shrink-0"
                      :stroke-width="1.5"
                    />
                    <div class="flex-1">
                      <div class="flex items-center gap-x-2">
                        <span class="text-lg font-medium text-gray-900">
                          {{ $t("database.revision.from-local-files") }}
                        </span>
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

          <!-- Step 2: Upload Local Files (for local source) -->
          <template
            v-else-if="
              currentStep === Step.UPLOAD_FILES && selectedSource === 'local'
            "
          >
            <div class="flex flex-col gap-y-4">
              <div class="text-sm text-gray-600">
                {{ $t("database.revision.upload-files-description") }}
              </div>

              <!-- File upload area (only show when no files selected) -->
              <div
                v-if="localFiles.length === 0"
                class="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center hover:border-gray-400 transition-colors cursor-pointer"
                @click="triggerFileInput"
                @dragover.prevent
                @drop.prevent="handleFileDrop"
              >
                <input
                  ref="fileInput"
                  type="file"
                  multiple
                  accept=".sql,.txt,.md,text/plain,text/markdown,text/x-sql"
                  class="hidden"
                  @change="handleFileSelect"
                />
                <FolderOpenIcon class="w-12 h-12 mx-auto text-gray-400 mb-3" />
                <p class="text-sm text-gray-600">
                  {{ $t("database.revision.drag-drop-or-click") }}
                </p>
                <p class="text-xs text-gray-500 mt-1">
                  {{ $t("database.revision.supported-formats") }}
                </p>
              </div>

              <!-- Hidden file input for when files are already selected -->
              <input
                v-else
                ref="fileInput"
                type="file"
                multiple
                accept=".sql,.txt,.md,text/plain,text/markdown,text/x-sql"
                class="hidden"
                @change="handleFileSelect"
              />

              <!-- Uploaded files list -->
              <div v-if="localFiles.length > 0" class="flex flex-col gap-y-3">
                <div class="flex items-center justify-between">
                  <h4 class="font-medium text-control">
                    {{ $t("database.revision.uploaded-files") }}
                    <span
                      v-if="localFiles.length > 1"
                      class="text-control-light"
                    >
                      ({{ localFiles.length }})
                    </span>
                  </h4>
                  <NButton size="small" @click="triggerFileInput">
                    <template #icon>
                      <PlusIcon class="w-4 h-4" />
                    </template>
                    {{ $t("database.revision.add-more-files") }}
                  </NButton>
                </div>
                <div
                  v-for="(file, index) in localFiles"
                  :key="index"
                  class="border border-gray-200 rounded-lg p-4 flex flex-col gap-y-3"
                >
                  <div class="flex items-start justify-between">
                    <div class="flex-1 flex flex-col gap-y-3">
                      <div class="flex items-center gap-2">
                        <FileIcon class="w-4 h-4 text-gray-500" />
                        <span class="text-sm font-medium">{{ file.name }}</span>
                        <span class="text-xs text-gray-500"
                          >({{ formatFileSize(file.size) }})</span
                        >
                      </div>

                      <!-- Version and Type inputs -->
                      <div class="grid grid-cols-4 gap-3">
                        <div class="col-span-3">
                          <label class="block text-xs text-gray-500 mb-1">
                            {{ $t("common.version") }} *
                          </label>
                          <NInput
                            v-model:value="file.version"
                            size="small"
                            placeholder="e.g., 1.0.0"
                            :status="getVersionStatus(file.version)"
                          />
                          <p
                            v-if="
                              file.version && !validateVersion(file.version)
                            "
                            class="text-xs text-red-600 mt-1"
                          >
                            {{ $t("database.revision.invalid-version-format") }}
                          </p>
                          <p
                            v-else-if="
                              file.version && isVersionDuplicate(file.version)
                            "
                            class="text-xs text-red-600 mt-1"
                          >
                            {{ $t("database.revision.version-already-exists") }}
                          </p>
                        </div>
                        <div class="col-span-1">
                          <label class="block text-xs text-gray-500 mb-1">
                            {{ $t("database.revision.revision-type") }}
                          </label>
                          <NSelect
                            v-model:value="file.type"
                            size="small"
                            :options="revisionTypeOptions"
                          />
                        </div>
                      </div>

                      <!-- File content preview -->
                      <div v-if="file.content" class="mt-2">
                        <label class="block text-xs text-gray-500 mb-1">
                          {{ $t("database.revision.content-preview") }}
                        </label>
                        <div
                          class="bg-gray-50 rounded-sm p-2 text-xs font-mono text-gray-700 max-h-32 overflow-auto"
                        >
                          <pre
                            >{{ file.content.substring(0, 500)
                            }}{{ file.content.length > 500 ? "..." : "" }}</pre
                          >
                        </div>
                      </div>
                    </div>
                    <NButton
                      quaternary
                      size="tiny"
                      @click="removeLocalFile(index)"
                    >
                      <XIcon class="w-4 h-4" />
                    </NButton>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <!-- Step 2: Select Release (for release source) -->
          <template
            v-else-if="
              currentStep === Step.SELECT_RELEASE &&
              selectedSource === 'release'
            "
          >
            <div class="flex flex-col gap-y-4">
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
            <div class="flex flex-col gap-y-4">
              <div class="text-sm text-gray-600">
                {{ $t("database.revision.select-files-description") }}
              </div>

              <!-- Selectable files section -->
              <div
                v-if="selectableFiles.length > 0"
                class="flex flex-col gap-y-2"
              >
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
                  :release-type="selectedRelease!.type"
                  :show-selection="true"
                  :selected-files="selectedFiles"
                  @update:selected-files="handleFileSelection"
                />
              </div>

              <!-- Files with existing versions section -->
              <div
                v-if="filesWithExistingVersions.length > 0"
                class="flex flex-col gap-y-2 mt-6"
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
                  :release-type="selectedRelease!.type"
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
import {
  FileIcon,
  FolderOpenIcon,
  PackageIcon,
  PlusIcon,
  XIcon,
} from "lucide-vue-next";
import {
  NButton,
  NInput,
  NRadio,
  NRadioGroup,
  NSelect,
  NStep,
  NSteps,
} from "naive-ui";
import type { Ref } from "vue";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import ReleaseDataTable from "@/components/Release/ReleaseDataTable.vue";
import ReleaseFileTable from "@/components/Release/ReleaseDetail/ReleaseFileTable/ReleaseFileTable.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { revisionServiceClientConnect } from "@/connect";
import {
  pushNotification,
  useCurrentProjectV1,
  useReleaseStore,
  useRevisionStore,
  useSheetV1Store,
} from "@/store";
import {
  type Release,
  type Release_File,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";
import {
  BatchCreateRevisionsRequestSchema,
  type CreateRevisionRequest,
  CreateRevisionRequestSchema,
  type Revision,
  Revision_Type,
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
  UPLOAD_FILES = 4, // Alternative step 2 for local files path
}

interface LocalFile {
  name: string;
  size: number;
  content: string;
  version: string;
  type: Revision_Type;
}

const { project } = useCurrentProjectV1();
const { t } = useI18n();
const releaseStore = useReleaseStore();
const revisionStore = useRevisionStore();
const sheetStore = useSheetV1Store();
const show = defineModel<boolean>("show", { default: false });

const selectedSource: Ref<"release" | "local"> = ref("release");
const isCreating = ref(false);
const currentStep = ref(Step.SELECT_SOURCE);
const isLoadingReleases = ref(false);
const releaseList = ref<Release[]>([]);
const selectedRelease = ref<Release | null>(null);
const selectedFiles = ref<Release_File[]>([]);
const existingRevisionVersions = ref<Set<string>>(new Set());

// Local files state
const fileInput = ref<HTMLInputElement>();
const localFiles = ref<LocalFile[]>([]);

const revisionTypeOptions = [
  {
    label: t("database.revision.type-versioned"),
    value: Revision_Type.VERSIONED,
  },
  {
    label: t("database.revision.type-declarative"),
    value: Revision_Type.DECLARATIVE,
  },
];

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
  if (selectedSource.value === "local") {
    return currentStep.value === Step.UPLOAD_FILES;
  }
  return currentStep.value === Step.SELECT_FILES;
});

const canSubmit = computed(() => {
  if (selectedSource.value === "local") {
    return (
      localFiles.value.length > 0 &&
      localFiles.value.every(
        (f) =>
          validateVersion(f.version) &&
          !isVersionDuplicate(f.version) &&
          f.content
      )
    );
  }
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

// Check if a version already exists (for local files)
const isVersionDuplicate = (version: string): boolean => {
  return existingRevisionVersions.value.has(version);
};

// Map release type to revision type
const mapReleaseTypeToRevisionType = (
  releaseType: Release_Type
): Revision_Type => {
  switch (releaseType) {
    case Release_Type.VERSIONED:
      return Revision_Type.VERSIONED;
    case Release_Type.DECLARATIVE:
      return Revision_Type.DECLARATIVE;
    default:
      return Revision_Type.TYPE_UNSPECIFIED;
  }
};

// Version validation - must match backend format (numeric parts separated by dots)
const validateVersion = (version: string): boolean => {
  if (!version) return false;
  // Version must be numeric parts separated by dots (e.g., "1", "1.0", "1.0.0")
  const versionRegex = /^(\d+)(\.(\d+))*$/;
  return versionRegex.test(version);
};

// Get version input status for validation display
const getVersionStatus = (version: string): "error" | undefined => {
  if (!version) return "error";
  if (!validateVersion(version)) return "error";
  if (isVersionDuplicate(version)) return "error";
  return undefined;
};

// File handling functions
const formatFileSize = (bytes: number): string => {
  if (bytes < 1024) return bytes + " B";
  const kb = bytes / 1024;
  if (kb < 1024) return kb.toFixed(1) + " KB";
  const mb = kb / 1024;
  return mb.toFixed(1) + " MB";
};

const triggerFileInput = () => {
  fileInput.value?.click();
};

const readFileContent = (file: File): Promise<string> => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = (e) => resolve(e.target?.result as string);
    reader.onerror = reject;
    reader.readAsText(file);
  });
};

const handleFileSelect = async (event: Event) => {
  const input = event.target as HTMLInputElement;
  if (input.files) {
    await processFiles(Array.from(input.files));
  }
  input.value = ""; // Reset input
};

const handleFileDrop = async (event: DragEvent) => {
  if (event.dataTransfer?.files) {
    await processFiles(Array.from(event.dataTransfer.files));
  }
};

const processFiles = async (files: File[]) => {
  for (const file of files) {
    // Check if file is acceptable
    const extension = file.name.split(".").pop()?.toLowerCase();
    if (!["sql", "txt", "md"].includes(extension || "")) {
      pushNotification({
        module: "bytebase",
        style: "WARN",
        title: `Unsupported file type ${extension} for ${file.name}`,
      });
      continue;
    }

    // Check for duplicates
    if (localFiles.value.some((f) => f.name === file.name)) {
      continue;
    }

    try {
      const content = await readFileContent(file);

      // Auto-generate version from filename if possible
      let version = "";

      // Try different patterns to extract version
      // Priority order: most specific to least specific
      const patterns = [
        // Match semantic versioning: v1.0.0, V1.2.3, 1.0.0
        /[Vv]?(\d+\.\d+\.\d+)/,
        // Match two-part version: v1.0, V2.1, 1.0
        /[Vv]?(\d+\.\d+)/,
        // Match version with underscores: v1_0_0, V2_1_3
        /[Vv]?(\d+)[_](\d+)(?:[_](\d+))?/,
        // Match simple numeric version: v001, V123, 001, 123
        /[Vv]?(\d{3,})/,
        // Match any number sequence: v1, V2, 1, 2
        /[Vv]?(\d+)/,
      ];

      for (const pattern of patterns) {
        const match = file.name.match(pattern);
        if (match) {
          if (pattern.source.includes("_")) {
            // Handle underscore-separated versions
            const parts = [match[1], match[2], match[3]].filter(Boolean);
            version = parts.join(".");
          } else {
            // For other patterns, use the first captured group
            version = match[1];
          }
          break;
        }
      }

      // If we found a 3-digit number like 001, format it as 0.0.1
      if (version && /^\d{3}$/.test(version)) {
        version = version.split("").join(".");
      }

      localFiles.value.push({
        name: file.name,
        size: file.size,
        content,
        version,
        type: Revision_Type.VERSIONED,
      });
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `Failed to read file ${file.name}`,
      });
    }
  }
};

const removeLocalFile = (index: number) => {
  localFiles.value.splice(index, 1);
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
    localFiles.value = [];
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
    if (selectedSource.value === "local") {
      currentStep.value = Step.UPLOAD_FILES;
    } else {
      currentStep.value = Step.SELECT_RELEASE;
      await loadReleases();
    }
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
  if (
    currentStep.value === Step.SELECT_RELEASE ||
    currentStep.value === Step.UPLOAD_FILES
  ) {
    currentStep.value = Step.SELECT_SOURCE;
  } else if (currentStep.value === Step.SELECT_FILES) {
    currentStep.value = Step.SELECT_RELEASE;
  }
};

const handleConfirm = async () => {
  if (!canSubmit.value) return;

  isCreating.value = true;
  try {
    let requests: CreateRevisionRequest[] = [];

    if (selectedSource.value === "local") {
      // For local files, create sheets first
      for (const file of localFiles.value) {
        try {
          // Create a sheet for each file
          const sheet = await sheetStore.createSheet(project.value.name, {
            content: new TextEncoder().encode(file.content),
          });

          // Create revision request with the new sheet
          requests.push(
            createProto(CreateRevisionRequestSchema, {
              parent: props.database,
              revision: {
                version: file.version,
                sheet: sheet.name,
                type: file.type,
              },
            })
          );
        } catch (error) {
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: `Failed to create sheet for ${file.name}`,
          });
          throw error;
        }
      }
    } else if (selectedRelease.value) {
      // Create revision requests using the existing sheet from release files
      requests = selectedFiles.value.map((file) =>
        createProto(CreateRevisionRequestSchema, {
          parent: props.database,
          revision: {
            release: selectedRelease.value!.name,
            version: file.version,
            file: `${selectedRelease.value!.name}/files/${file.path}`,
            sheet: file.sheet,
            type: mapReleaseTypeToRevisionType(selectedRelease.value!.type),
          },
        })
      );
    }
    if (requests.length === 0) {
      return;
    }

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
      title: "Revisions Created",
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
