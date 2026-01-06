<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-x-1 text-sm font-medium">
        <PackageIcon class="w-4 h-4" />
        <span :class="compact ? '' : 'text-base'">{{ releaseTitle }}</span>
      </div>
      <NButton
        v-if="release && isValidReleaseName(release.name)"
        size="tiny"
        tag="a"
        text
        :href="`/${release.name}`"
        target="_blank"
        icon-placement="right"
      >
        {{ $t("common.view") }}
        <template #icon>
          <ExternalLinkIcon class="w-3 h-3" />
        </template>
      </NButton>
    </div>

    <div
      v-if="release && isValidReleaseName(release.name)"
      class="border rounded-md bg-gray-50"
      :class="compact ? 'px-3 py-2' : 'px-4 py-3'"
    >
      <div class="flex flex-col" :class="compact ? 'gap-y-2' : 'gap-y-3'">
        <div class="flex items-start justify-between">
          <div>
            <h3 class="text-sm font-medium text-gray-900">
              {{ release.title }}
            </h3>
            <p v-if="release.name" class="text-xs text-gray-500 mt-0.5">
              {{ release.name }}
            </p>
          </div>
        </div>

        <div
          v-if="release.files && release.files.length > 0"
          class="flex flex-col gap-y-1"
        >
          <div class="flex items-center justify-between">
            <h4 class="text-xs font-medium text-gray-700">
              {{ $t("release.files") }} ({{ release.files.length }})
            </h4>
            <NButton
              v-if="release.files.length > maxDisplayedFiles"
              size="tiny"
              quaternary
              tag="a"
              :href="`/${release.name}`"
              target="_blank"
            >
              {{ $t("release.view-all-files") }}
            </NButton>
          </div>
          <div
            class="w-full grid gap-1"
            :class="compact ? 'grid-cols-1 sm:grid-cols-2' : 'grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2'"
          >
            <div
              v-for="file in displayedFiles"
              :key="file.path"
              class="w-full flex items-center justify-between text-xs bg-white rounded-sm"
              :class="compact ? 'p-1.5' : 'p-2'"
            >
              <div class="flex-1 min-w-0 mr-2">
                <div class="font-medium truncate">{{ file.path }}</div>
                <div class="text-gray-500">
                  {{ file.version }} •
                  {{ getChangeTypeText(file.enableGhost) }}
                </div>
              </div>
              <div
                v-if="release.type"
                class="inline-flex items-center rounded-sm text-xs shrink-0 bg-blue-100 text-blue-800"
                :class="compact ? 'px-1 py-0.5' : 'px-1.5 py-0.5'"
              >
                {{ getReleaseFileTypeText(release.type) }}
              </div>
            </div>
            <div
              v-if="release.files.length > maxDisplayedFiles"
              class="text-xs text-gray-500 text-center py-1"
            >
              {{
                $t("release.and-n-more-files", {
                  count: release.files.length - maxDisplayedFiles,
                })
              }}
            </div>
          </div>
        </div>

        <div v-if="release.vcsSource" class="flex flex-col gap-y-0.5">
          <h4 class="text-xs font-medium text-gray-700">
            {{ $t("release.vcs-source") }}
          </h4>
          <div class="text-xs">
            <span class="text-gray-500">{{ getVCSTypeText(release.vcsSource.vcsType) }}:</span>
            <a
              v-if="release.vcsSource.url"
              :href="release.vcsSource.url"
              target="_blank"
              class="ml-1 text-blue-600 hover:text-blue-800"
            >
              {{ release.vcsSource.url }}
            </a>
          </div>
        </div>

        <div class="text-xs text-gray-500">
          {{
            dayjs(getDateForPbTimestampProtoEs(release.createTime)).format(
              "YYYY-MM-DD HH:mm:ss"
            )
          }}
        </div>
      </div>
    </div>

    <div v-else-if="!releaseReady" class="border rounded-md p-2 bg-gray-50">
      <div class="flex items-center gap-x-2">
        <BBSpin />
        <span class="text-xs text-gray-600">
          {{ $t("common.loading") }}
        </span>
      </div>
    </div>

    <div v-else class="border rounded-md p-2 bg-red-50">
      <div class="text-xs text-red-600">
        {{ $t("release.not-found") }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import dayjs from "dayjs";
import { ExternalLinkIcon, PackageIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { useReleaseByName } from "@/store";
import { getDateForPbTimestampProtoEs, isValidReleaseName } from "@/types";
import { VCSType } from "@/types/proto-es/v1/common_pb";
import { Release_Type } from "@/types/proto-es/v1/release_service_pb";

const props = withDefaults(
  defineProps<{
    releaseName: string;
    compact?: boolean;
  }>(),
  {
    compact: false,
  }
);

const { t } = useI18n();
const { release, ready: releaseReady } = useReleaseByName(
  computed(() => props.releaseName)
);

const maxDisplayedFiles = computed(() => (props.compact ? 4 : 6));

const releaseTitle = computed(() => {
  if (release.value && release.value.title) {
    return release.value.title;
  }
  if (props.releaseName) {
    const parts = props.releaseName.split("/");
    return parts[parts.length - 1] || props.releaseName;
  }
  return t("release.title");
});

const displayedFiles = computed(() => {
  if (!release.value?.files) return [];
  return release.value.files.slice(0, maxDisplayedFiles.value);
});

const getChangeTypeText = (enableGhost: boolean) => {
  return enableGhost
    ? t("release.change-type.ddl-ghost")
    : t("release.change-type.ddl");
};

const getReleaseFileTypeText = (fileType: Release_Type) => {
  switch (fileType) {
    case Release_Type.VERSIONED:
      return t("release.file-type.versioned");
    case Release_Type.DECLARATIVE:
      return t("release.file-type.declarative");
    case Release_Type.TYPE_UNSPECIFIED:
      return t("release.file-type.unspecified");
    default:
      fileType satisfies never;
      return t("release.file-type.unspecified");
  }
};

const getVCSTypeText = (vcsType: VCSType) => {
  switch (vcsType) {
    case VCSType.GITHUB:
      return "GitHub";
    case VCSType.GITLAB:
      return "GitLab";
    case VCSType.BITBUCKET:
      return "Bitbucket";
    case VCSType.AZURE_DEVOPS:
      return "Azure DevOps";
    default:
      return t("release.vcs-type.unspecified");
  }
};
</script>
