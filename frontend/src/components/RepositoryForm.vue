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
    <div>
      <div class="textlabel">
        {{ $t("project.settings.schema-change-type") }}
        <span class="text-red-600">*</span>
        <LearnMoreLink
          url="https://www.bytebase.com/docs/change-database/state-based-migration?source=console"
          class="ml-1"
        />
      </div>
    </div>
    <div>
      <div class="textlabel">
        {{ $t("repository.file-path-template") }}
        <span class="text-red-600">*</span>
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.file-path-template-description") }}
        <LearnMoreLink
          url="https://www.bytebase.com/docs/vcs-integration/name-and-organize-schema-files?source=console#file-path-template"
          class="ml-1"
        />
      </div>
      <BBTextField
        id="filepathtemplate"
        v-model:value="repositoryConfig.filePathTemplate"
        name="filepathtemplate"
        class="mt-2 w-full"
        :disabled="!allowEdit"
      />
      <div class="mt-2 textinfolabel capitalize">
        <span class="text-red-600">*</span>
        {{ $t("common.required-placeholder") }}:
        {{ FILE_REQUIRED_PLACEHOLDER }};
        <template v-if="fileOptionalPlaceholder.length > 0">
          {{ $t("common.optional-placeholder") }}:
          {{ fileOptionalPlaceholder.join(", ") }};
        </template>
        {{ $t("common.optional-directory-wildcard") }}:
        {{ FILE_OPTIONAL_DIRECTORY_WILDCARD }}
      </div>
      <div class="mt-2 textinfolabel">
        • {{ $t("repository.file-path-example-schema-migration") }}:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "ddl"
          )
        }}
      </div>
      <div class="mt-2 textinfolabel">
        • {{ $t("repository.file-path-example-data-migration") }}:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "dml"
          )
        }}
      </div>
    </div>
  </div>
  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import type { ExternalRepositoryInfo, RepositoryConfig } from "@/types";
import type { Project } from "@/types/proto/v1/project_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";

const FILE_REQUIRED_PLACEHOLDER = "{{DB_NAME}}, {{VERSION}}, {{TYPE}}";
const FILE_OPTIONAL_DIRECTORY_WILDCARD = "*, **";
const SINGLE_ASTERISK_REGEX = /\/\*\//g;
const DOUBLE_ASTERISKS_REGEX = /\/\*\*\//g;

interface LocalState {
  showFeatureModal: boolean;
  showInstanceAssignmentDrawer: boolean;
}

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

const state = reactive<LocalState>({
  showFeatureModal: false,
  showInstanceAssignmentDrawer: false,
});

const isTenantProject = computed(() => {
  return props.project.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const sampleFilePath = (
  baseDirectory: string,
  filePathTemplate: string,
  type: string
): string => {
  type Item = {
    placeholder: string;
    sampleText: string;
  };
  const placeholderList: Item[] = [
    {
      placeholder: "{{VERSION}}",
      sampleText: "202101131000",
    },
    {
      placeholder: "{{DB_NAME}}",
      sampleText: "db1",
    },
    {
      placeholder: "{{TYPE}}",
      sampleText: type,
    },
    {
      placeholder: "{{ENV_ID}}",
      sampleText: "env1",
    },
    {
      placeholder: "{{ENV_NAME}}", // for legacy support
      sampleText: "env1",
    },
    {
      placeholder: "{{DESCRIPTION}}",
      sampleText: "create_tablefoo_for_bar",
    },
  ];
  let result = `${baseDirectory}/${filePathTemplate}`;
  // To replace the wildcard.
  result = result.replace(SINGLE_ASTERISK_REGEX, "/foo/");
  result = result.replace(DOUBLE_ASTERISKS_REGEX, "/foo/bar/");
  for (const item of placeholderList) {
    const re = new RegExp(item.placeholder, "g");
    result = result.replace(re, item.sampleText);
  }
  return result;
};

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

const fileOptionalPlaceholder = computed(() => {
  const tags = [] as string[];
  // Only allows {{ENV_ID}} to be an optional placeholder for non-tenant mode projects
  if (!isTenantProject.value) tags.push("{{ENV_ID}}");
  tags.push("{{DESCRIPTION}}");
  return tags;
});
</script>
