<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div class="space-y-4">
    <div>
      <div v-if="!create && getWebhookLink !== ''" class="mb-2">
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
      <div class="textlabel flex items-center gap-x-1">
        {{ $t("database-group.self") }}
        <FeatureBadge :feature="'bb.feature.database-grouping'" />
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.database-group-description") }}
        <span
          v-if="hasDatabaseGroupPermission"
          class="cursor-pointer normal-link"
          @click="showDatabaseGroupPanel = true"
        >
          {{ $t("database-group.create") }}
        </span>
      </div>
      <DatabaseGroupSelect
        :selected="repositoryConfig.databaseGroup"
        class="mt-2"
        style="width: 100%"
        :project="project.name"
        :select-first-as-default="false"
        @update:selected="(val) => (repositoryConfig.databaseGroup = val ?? '')"
      />
    </div>
    <div>
      <div class="textlabel">
        {{ $t("common.branch") }}
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

  <DatabaseGroupPanel
    :show="showDatabaseGroupPanel"
    :project="project"
    :redirect-to-detail-page="false"
    @close="showDatabaseGroupPanel = false"
  />
</template>

<script lang="ts" setup>
import { Status } from "nice-grpc-common";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBTextField } from "@/bbkit";
import { DatabaseGroupSelect } from "@/components/v2/Select";
import { useVCSConnectorStore, hasFeature } from "@/store";
import type {
  RepositoryConfig,
  ComposedProject,
  ResourceId,
  ValidatedMessage,
} from "@/types";
import { VCSType } from "@/types/proto/v1/common";
import type { VCSRepository } from "@/types/proto/v1/vcs_provider_service";
import { hasProjectPermissionV2 } from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";
import DatabaseGroupPanel from "./DatabaseGroup/DatabaseGroupPanel.vue";
import FeatureBadge from "./FeatureGuard/FeatureBadge.vue";
import { VCSIcon } from "./VCS";
import { ResourceIdField } from "./v2";

const props = withDefaults(
  defineProps<{
    allowEdit?: boolean;
    create?: boolean;
    vcsType: VCSType;
    vcsName: string;
    repositoryInfo: VCSRepository;
    repositoryConfig: RepositoryConfig;
    project: ComposedProject;
  }>(),
  {
    allowEdit: true,
    create: false,
  }
);

const showDatabaseGroupPanel = ref<boolean>(false);
const { t } = useI18n();
const vcsConnectorStore = useVCSConnectorStore();

const hasDatabaseGroupPermission = computed(
  () =>
    hasFeature("bb.feature.database-grouping") &&
    hasProjectPermissionV2(props.project, "bb.projects.update")
);

const getWebhookLink = computed(() => {
  switch (props.vcsType) {
    case VCSType.AZURE_DEVOPS: {
      const parts = props.repositoryInfo.id.split("/");
      if (parts.length !== 3) {
        return "";
      }
      const [organization, project, _] = parts;
      return `https://dev.azure.com/${organization}/${project}/_settings/serviceHooks`;
    }
    case VCSType.GITHUB:
      return `${props.repositoryInfo.webUrl}/settings/hooks`;
    case VCSType.BITBUCKET:
      return `${props.repositoryInfo.webUrl}/admin/webhooks`;
    case VCSType.GITLAB:
      return `${props.repositoryInfo.webUrl}/-/hooks`;
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
