<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div class="space-y-4">
    <div>
      <div v-if="getWebhookLink !== ''" class="mb-2">
        <label class="textlabel mt-2">
          <i18n-t keypath="repository.our-webhook-link">
            <template #webhookLink>
              <a class="normal-link" :href="getWebhookLink" target="_blank">{{
                getWebhookLink
              }}</a>
            </template>
          </i18n-t>
        </label>
      </div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="gitprovider" class="textlabel">
          {{ $t("repository.git-provider") }}
        </label>
        <VCSIcon :type="vcsType" />
      </div>
      <BBTextField
        id="gitprovider"
        name="gitprovider"
        class="mt-2 w-full"
        :disabled="true"
        :value="vcsName"
      />
    </div>
    <div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="repository" class="textlabel">
          {{ $t("common.repository") }}
        </label>
        <div
          v-if="!create && allowEdit"
          class="ml-1 normal-link text-sm"
          @click.prevent="$emit('change-repository')"
        >
          {{ $t("common.change") }}
        </div>
      </div>
      <BBTextField
        id="repository"
        name="repository"
        class="mt-2 w-full"
        :disabled="true"
        :value="repositoryInfo.fullPath"
      />
    </div>
    <div>
      <div class="textlabel">
        {{ $t("common.branch") }} <span class="text-red-600">*</span>
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.branch-observe-file-change") }}
      </div>
      <BBTextField
        id="branch"
        v-model:value="repositoryConfig.branchFilter"
        name="branch"
        class="mt-2 w-full"
        placeholder="e.g. main"
        :disabled="!allowEdit"
      />
    </div>
    <div>
      <div class="textlabel">{{ $t("repository.base-directory") }}</div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.base-directory-description") }}
      </div>
      <BBTextField
        id="basedirectory"
        v-model:value="repositoryConfig.baseDirectory"
        name="basedirectory"
        class="mt-2 w-full"
        :disabled="!allowEdit"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { ExternalRepositoryInfo, RepositoryConfig } from "@/types";
import type { Project } from "@/types/proto/v1/project_service";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";

defineEmits<{
  (event: "change-repository"): void;
}>();

const props = withDefaults(
  defineProps<{
    allowEdit?: boolean;
    create?: boolean;
    vcsType: VCSProvider_Type;
    vcsName: string;
    repositoryInfo: ExternalRepositoryInfo;
    repositoryConfig: RepositoryConfig;
    project: Project;
  }>(),
  {
    allowEdit: true,
    create: false,
  }
);

const getWebhookLink = computed(() => {
  if (props.vcsType === VCSProvider_Type.AZURE_DEVOPS) {
    const parts = props.repositoryInfo.externalId.split("/");
    if (parts.length !== 3) {
      return "";
    }
    const [organization, project, _] = parts;
    return `https://dev.azure.com/${organization}/${project}/_settings/serviceHooks`;
  }
  return "";
});
</script>
