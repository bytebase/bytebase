<template>
  <div class="space-y-4">
    <div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="gitprovider" class="textlabel"> {{ $t('repository.git-provider') }} </label>
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
        <label for="repository" class="textlabel"> {{ $t('common.repository') }} </label>
        <div
          v-if="!create && allowEdit"
          class="ml-1 normal-link text-sm"
          @click.prevent="$emit('change-repository')"
        >
          {{ $t('common.change') }}
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
      <div class="textlabel">{{ $t('common.branch') }} <span class="text-red-600">*</span></div>
      <div class="mt-1 textinfolabel">
        {{ $t('repository.branch-observe-file-change') }}
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
      <div v-if="vcsType == 'GITLAB_SELF_HOST'" class="mt-2 textinfolabel">
        {{ $t('repository.branch-specify-tip') }}
      </div>
    </div>
    <div>
      <div class="textlabel">{{ $t('repository.base-directory') }}</div>
      <div class="mt-1 textinfolabel">
        {{ $t('repository.base-directory-description') }}
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
        {{ $t('repository.file-path-template') }} <span class="text-red-600">*</span>
        <a
          href="https://docs.bytebase.com/use-bytebase/vcs-integration/organize-repository-files#file-path-template"
          target="__blank"
          class="font-normal normal-link"
        >
          {{ $t('common.config-guide') }}</a
        >
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t('repository.file-path-template-description') }}
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
        <span class="text-red-600">*</span> {{ $t('common.required-placeholder') }}:
        {{ FILE_REQUIRED_PLACEHOLDER }}; {{ $t('common.optional-placeholder') }}:
        {{ FILE_OPTIONAL_PLACEHOLDER }}
      </div>
      <div class="mt-2 textinfolabel">
        • {{ $t('repository.file-path-example-normal-migration') }}:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "migrate"
          )
        }}
      </div>
      <div class="mt-2 textinfolabel">
        • {{ $t('repository.file-path-example-baseline-migration') }}:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "baseline"
          )
        }}
      </div>
    </div>
    <div>
      <div class="textlabel">
        {{ $t('repository.schema-path-template') }}
        <a
          href="https://docs.bytebase.com/use-bytebase/vcs-integration/organize-repository-files#schema-path-template"
          target="__blank"
          class="font-normal normal-link"
        >
          {{ $t('common.config-guide') }}</a
        >
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t('repository.schema-writeback-description') }}
        <span class="font-medium text-main"
          >{{ $t('repository.schema-writeback-protected-branch') }}</span
        >
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
        <span class="text-red-600">*</span> {{ $t('repository.if-specified') }}, {{ $t('common.required-placeholder') }}:
        {{ SCHEMA_REQUIRED_PLACEHOLDER }}; {{ $t('common.optional-placeholder') }}:
        {{ SCHEMA_OPTIONAL_PLACEHOLDER }}
      </div>
      <div
        v-if="repositoryConfig.schemaPathTemplate"
        class="mt-2 textinfolabel"
      >
        • {{ $t('repository.schema-path-example') }}:
        {{
          sampleSchemaPath(
            repositoryConfig.baseDirectory,
            repositoryConfig.schemaPathTemplate
          )
        }}
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { ExternalRepositoryInfo, RepositoryConfig, VCSType } from "../types";

const FILE_REQUIRED_PLACEHOLDER = "{{DB_NAME}}, {{VERSION}}, {{TYPE}}";
const FILE_OPTIONAL_PLACEHOLDER = "{{ENV_NAME}}, {{DESCRIPTION}}";
const SCHEMA_REQUIRED_PLACEHOLDER = "{{DB_NAME}}";
const SCHEMA_OPTIONAL_PLACEHOLDER = "{{ENV_NAME}}";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default {
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
  },
  emits: ["change-repository"],
  setup() {
    const state = reactive<LocalState>({});

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

    return {
      FILE_REQUIRED_PLACEHOLDER,
      FILE_OPTIONAL_PLACEHOLDER,
      SCHEMA_REQUIRED_PLACEHOLDER,
      SCHEMA_OPTIONAL_PLACEHOLDER,
      state,
      sampleFilePath,
      sampleSchemaPath,
    };
  },
};
</script>
