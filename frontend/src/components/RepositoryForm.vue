<template>
  <div class="space-y-4">
    <div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="gitprovider" class="textlabel"> Git provider </label>
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
        <label for="repository" class="textlabel"> Repository </label>
        <div
          v-if="!create && allowEdit"
          class="ml-1 normal-link text-sm"
          @click.prevent="$emit('change-repository')"
        >
          Change
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
      <div class="textlabel">Branch <span class="text-red-600">*</span></div>
      <div class="mt-1 textinfolabel">
        The branch where Bytebase observes the file change.
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
        Tip: You can also use wildcard like 'feature/*'
      </div>
    </div>
    <div>
      <div class="textlabel">Base directory</div>
      <div class="mt-1 textinfolabel">
        The root directory where Bytebase observes the file change. If empty,
        then it observes the entire repository.
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
        File path template <span class="text-red-600">*</span>
        <a
          href="https://docs.bytebase.com/use-bytebase/vcs-integration/organize-repository-files#file-path-template"
          target="__blank"
          class="font-normal normal-link"
        >
          config guide</a
        >
      </div>
      <div class="mt-1 textinfolabel">
        Bytebase only observes the file path name matching the template pattern
        relative to the base directory.
      </div>
      <input
        id="filepathtemplate"
        v-model="repositoryConfig.filePathTemplate"
        name="filepathtemplate"
        type="text"
        class="textfield mt-2 w-full"
        :disabled="!allowEdit"
      />
      <div class="mt-2 textinfolabel">
        <span class="text-red-600">*</span> Required placeholders:
        {{ FILE_REQUIRED_PLACEHOLDER }}; optional placeholders:
        {{ FILE_OPTIONAL_PLACEHOLDER }}
      </div>
      <div class="mt-2 textinfolabel">
        • File path example for normal migration type:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "migrate"
          )
        }}
      </div>
      <div class="mt-2 textinfolabel">
        • File path example for baseline migration type:
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
        Schema path template
        <a
          href="https://docs.bytebase.com/use-bytebase/vcs-integration/organize-repository-files#schema-path-template"
          target="__blank"
          class="font-normal normal-link"
        >
          config guide</a
        >
      </div>
      <div class="mt-1 textinfolabel">
        When specified, after each migration, Bytebase will write the latest
        schema to the schema path relative to the base directory in the same
        branch as the original commit triggering the migration. Leave empty if
        you don't want Bytebase to do this.
        <span class="font-medium text-main"
          >Make sure the changed branch is not protected or allow repository
          maintainer to push to that protected branch</span
        >.
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
        <span class="text-red-600">*</span> If specified, required placeholder:
        {{ SCHEMA_REQUIRED_PLACEHOLDER }}; optional placeholder:
        {{ SCHEMA_OPTIONAL_PLACEHOLDER }}
      </div>
      <div
        v-if="repositoryConfig.schemaPathTemplate"
        class="mt-2 textinfolabel"
      >
        • Schema path example:
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
import { reactive } from "@vue/reactivity";
import { PropType } from "@vue/runtime-core";
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
