<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div class="space-y-4">
    <div>
      <div v-if="getWebhookLink !== ''" class="mb-2">
        <label class="textlabel mt-2">
          <i18n-t keypath="repository.our-webhook-link">
            <template #webhookLink>
              <a class="normal-link" :href="getWebhookLink" target="_blank">
                {{ getWebhookLink }}
              </a>
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
      </div>
      <BBTextField
        id="repository"
        name="repository"
        class="mt-2 w-full"
        :disabled="true"
        :value="repositoryInfo.fullPath"
      />
      <ResourceIdField
        v-model:value="repositoryConfig.resourceId"
        class="max-w-full flex-nowrap mt-1.5"
        editing-class="mt-4"
        resource-type="vcs-connector"
        :resource-title="repositoryInfo.title"
        :validate="validateResourceId"
        :readonly="!create || !allowEdit"
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
        v-model:value="repositoryConfig.branch"
        name="branch"
        class="mt-2 w-full"
        placeholder="e.g. main"
        :disabled="!allowEdit"
        :required="true"
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
import { Status } from "nice-grpc-common";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useVCSConnectorStore } from "@/store";
import type { RepositoryConfig } from "@/types";
import type { ResourceId, ValidatedMessage } from "@/types";
import type { Project } from "@/types/proto/v1/project_service";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";
import type { VCSRepository } from "@/types/proto/v1/vcs_provider_service";
import { getErrorCode } from "@/utils/grpcweb";

const props = withDefaults(
  defineProps<{
    allowEdit?: boolean;
    create?: boolean;
    vcsType: VCSProvider_Type;
    vcsName: string;
    repositoryInfo: VCSRepository;
    repositoryConfig: RepositoryConfig;
    project: Project;
  }>(),
  {
    allowEdit: true,
    create: false,
  }
);

const { t } = useI18n();
const vcsConnectorStore = useVCSConnectorStore();

const getWebhookLink = computed(() => {
  switch (props.vcsType) {
    case VCSProvider_Type.AZURE_DEVOPS: {
      const parts = props.repositoryInfo.id.split("/");
      if (parts.length !== 3) {
        return "";
      }
      const [organization, project, _] = parts;
      return `https://dev.azure.com/${organization}/${project}/_settings/serviceHooks`;
    }
    case VCSProvider_Type.GITHUB:
      return `${props.repositoryInfo.webUrl}/settings/hooks`;
  }
  return "";
});

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  try {
    const instance = await vcsConnectorStore.getOrFetchConnector(
      props.project.name,
      resourceId
    );
    if (instance) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t("resource.instance"),
          }),
        },
      ];
    }
  } catch (error) {
    if (getErrorCode(error) !== Status.NOT_FOUND) {
      throw error;
    }
  }
  return [];
};
</script>
