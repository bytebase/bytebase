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
        name="branch"
        type="text"
        class="textfield mt-2 w-full"
        placeholder="e.g. master"
        :disabled="!allowEdit"
        v-model="repositoryConfig.branchFilter"
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
        name="basedirectory"
        type="text"
        class="textfield mt-2 w-full"
        :disabled="!allowEdit"
        v-model="repositoryConfig.baseDirectory"
      />
    </div>
    <div>
      <div class="textlabel">
        File path template <span class="text-red-600">*</span>
        <a
          href="https://docs.bytebase.com/use-bytebase/vcs-integration/organize-repository-files"
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
        name="filepathtemplate"
        type="text"
        class="textfield mt-2 w-full"
        :disabled="!allowEdit"
        v-model="repositoryConfig.filePathTemplate"
      />
      <div class="mt-2 textinfolabel">
        <span class="text-red-600">*</span> Required placeholders:
        {{ REQUIRED_PLACEHOLDER }}. Optional placeholders:
        {{ OPTIONAL_PLACEHOLDER }}
      </div>
      <div class="mt-2 textinfolabel">Example:</div>
      <div class="mt-2 textinfolabel">
        File path for normal migration type:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "migrate"
          )
        }}
      </div>
      <div class="mt-2 textinfolabel">
        File path for baseline migration type:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "baseline"
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

const REQUIRED_PLACEHOLDER = "{{VERSION}}, {{DB_NAME}}, {{TYPE}}";
const OPTIONAL_PLACEHOLDER = "{{ENV_NAME}}, {{DESCRIPTION}}";

interface LocalState {}

export default {
  name: "RepositoryForm",
  emits: ["change-repository"],
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
  components: {},
  setup(props, { emit }) {
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

      let result: string = `${baseDirectory}/${filePathTemplate}`;
      for (const item of placeholderList) {
        const re = new RegExp(item.placeholder, "g");
        result = result.replace(re, item.sampleText);
      }
      return result;
    };

    return {
      REQUIRED_PLACEHOLDER,
      OPTIONAL_PLACEHOLDER,
      state,
      sampleFilePath,
    };
  },
};
</script>
