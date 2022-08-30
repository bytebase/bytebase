<template>
  <div class="space-y-4">
    <div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="gitprovider" class="textlabel">
          {{ $t("repository.git-provider") }}
        </label>
        <template v-if="vcsType.startsWith('GITLAB')">
          <img class="h-4 w-auto" src="../assets/gitlab-logo.svg" />
        </template>
      </div>
      <input
        id="gitprovider"
        name="gitprovider"
        type="text"
        class="textfield mt-1 w-full"
        disabled="true"
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
      <input
        id="repository"
        name="repository"
        type="text"
        class="textfield mt-1 w-full"
        disabled="true"
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
      <input
        id="branch"
        v-model="repositoryConfig.branchFilter"
        name="branch"
        type="text"
        class="textfield mt-2 w-full"
        placeholder="e.g. master"
        :disabled="!allowEdit"
      />
    </div>
    <div>
      <div class="textlabel">{{ $t("repository.base-directory") }}</div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.base-directory-description") }}
      </div>
      <input
        id="basedirectory"
        v-model="repositoryConfig.baseDirectory"
        name="basedirectory"
        type="text"
        class="textfield mt-2 w-full"
        :disabled="!allowEdit"
      />
    </div>
    <div>
      <div class="textlabel">
        {{ $t("repository.file-path-template") }}
        <span class="text-red-600">*</span>
        <a
          href="https://bytebase.com/docs/vcs-integration/name-and-organize-schema-files#file-path-template?source=console"
          target="__blank"
          class="font-normal normal-link"
        >
          {{ $t("common.config-guide") }}</a
        >
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.file-path-template-description") }}
      </div>
      <input
        id="filepathtemplate"
        v-model="repositoryConfig.filePathTemplate"
        name="filepathtemplate"
        type="text"
        class="textfield mt-2 w-full"
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
            "migrate"
          )
        }}
      </div>
      <div class="mt-2 textinfolabel">
        • {{ $t("repository.file-path-example-data-migration") }}:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "data"
          )
        }}
      </div>
    </div>
    <div>
      <div class="textlabel">
        {{ $t("repository.schema-path-template") }}
        <a
          href="https://bytebase.com/docs/vcs-integration/name-and-organize-schema-files#schema-path-template?source=console"
          target="__blank"
          class="font-normal normal-link"
        >
          {{ $t("common.config-guide") }}</a
        >
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.schema-writeback-description") }}
        <span class="font-medium text-main">{{
          $t("repository.schema-writeback-protected-branch")
        }}</span>
      </div>
      <input
        id="schemapathtemplate"
        v-model="repositoryConfig.schemaPathTemplate"
        name="schemapathtemplate"
        type="text"
        class="textfield mt-2 w-full"
        :disabled="!allowEdit"
      />
      <div class="mt-2 textinfolabel">
        <span class="text-red-600">*</span> {{ $t("repository.if-specified") }},
        {{ $t("common.required-placeholder") }}:
        {{ SCHEMA_REQUIRED_PLACEHOLDER }};
        <template v-if="schemaOptionalTagPlaceholder.length > 0">
          {{ $t("common.optional-placeholder") }}:
          {{ schemaOptionalTagPlaceholder.join(", ") }}
        </template>
      </div>
      <div
        v-if="repositoryConfig.schemaPathTemplate"
        class="mt-2 textinfolabel"
      >
        • {{ $t("repository.schema-path-example") }}:
        {{
          sampleSchemaPath(
            repositoryConfig.baseDirectory,
            repositoryConfig.schemaPathTemplate
          )
        }}
      </div>
    </div>
    <div>
      <div class="textlabel">{{ $t("repository.sheet-path-template") }}</div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.sheet-path-template-description") }}
      </div>
      <input
        id="sheetpathtemplate"
        v-model="repositoryConfig.sheetPathTemplate"
        name="sheetpathtemplate"
        type="text"
        class="textfield mt-2 w-full"
        :disabled="!allowEdit"
      />
      <div class="mt-2 textinfolabel capitalize">
        <span class="text-red-600">*</span>
        {{ $t("common.required-placeholder") }}: {{ "\{\{NAME\}\}" }};
        <template v-if="schemaOptionalTagPlaceholder.length > 0">
          {{ $t("common.optional-placeholder") }}: {{ "\{\{ENV_NAME\}\}" }},
          {{ "\{\{DB_NAME\}\}" }}
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, PropType, defineComponent, computed } from "vue";
import {
  ExternalRepositoryInfo,
  Project,
  RepositoryConfig,
  VCSType,
} from "@/types";

const FILE_REQUIRED_PLACEHOLDER = "{{DB_NAME}}, {{VERSION}}, {{TYPE}}";
const SCHEMA_REQUIRED_PLACEHOLDER = "{{DB_NAME}}";
const FILE_OPTIONAL_DIRECTORY_WILDCARD = "*, **";
const SINGLE_ASTERISK_REGEX = /\/\*\//g;
const DOUBLE_ASTERISKS_REGEX = /\/\*\*\//g;

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
  name: "RepositoryForm",
  props: {
    allowEdit: {
      default: true,
      type: Boolean,
    },
    create: {
      type: Boolean,
      default: false,
    },
    vcsType: {
      required: true,
      type: String as PropType<VCSType>,
    },
    vcsName: {
      required: true,
      type: String,
    },
    repositoryInfo: {
      required: true,
      type: Object as PropType<ExternalRepositoryInfo>,
    },
    repositoryConfig: {
      required: true,
      type: Object as PropType<RepositoryConfig>,
    },
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  emits: ["change-repository"],
  setup(props) {
    const state = reactive<LocalState>({});

    const isTenantProject = computed(() => {
      return props.project.tenantMode === "TENANT";
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
          placeholder: "{{ENV_NAME}}",
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

    const sampleSchemaPath = (
      baseDirectory: string,
      schemaPathTemplate: string
    ): string => {
      type Item = {
        placeholder: string;
        sampleText: string;
      };

      const placeholderList: Item[] = [
        {
          placeholder: "{{DB_NAME}}",
          sampleText: "db1",
        },
      ];

      let result = `${baseDirectory}/${schemaPathTemplate}`;
      for (const item of placeholderList) {
        const re = new RegExp(item.placeholder, "g");
        result = result.replace(re, item.sampleText);
      }
      return result;
    };

    const fileOptionalPlaceholder = computed(() => {
      const tags = [] as string[];
      // Only allows {{ENV_NAME}} to be an optional placeholder for non-tenant mode projects
      if (!isTenantProject.value) tags.push("{{ENV_NAME}}");
      tags.push("{{DESCRIPTION}}");
      return tags;
    });

    const schemaOptionalTagPlaceholder = computed(() => {
      const tags = [] as string[];
      // Only allows {{ENV_NAME}} to be an optional placeholder for non-tenant mode projects
      if (!isTenantProject.value) tags.push("{{ENV_NAME}}");
      return tags;
    });

    return {
      FILE_REQUIRED_PLACEHOLDER,
      fileOptionalPlaceholder,
      FILE_OPTIONAL_DIRECTORY_WILDCARD,
      SCHEMA_REQUIRED_PLACEHOLDER,
      schemaOptionalTagPlaceholder,
      state,
      sampleFilePath,
      sampleSchemaPath,
    };
  },
});
</script>
