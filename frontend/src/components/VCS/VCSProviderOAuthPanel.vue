<template>
  <div class="space-y-4">
    <ul class="textinfolabel space-y-2">
      <template
        v-if="
          config.uiType == 'GITLAB_SELF_HOST' || config.uiType == 'GITLAB_COM'
        "
      >
        <li>
          •
          {{
            $t(
              "gitops.setting.add-git-provider.access-token.gitlab-project-access-token"
            )
          }}
        </li>
        <li>
          •
          {{
            $t(
              "gitops.setting.add-git-provider.access-token.gitlab-group-access-token"
            )
          }}
        </li>
        <li>
          •
          {{
            $t(
              "gitops.setting.add-git-provider.access-token.gitlab-personal-access-token"
            )
          }}
        </li>
      </template>
      <template
        v-if="
          config.uiType == 'GITHUB_COM' || config.uiType == 'GITHUB_ENTERPRISE'
        "
      >
        <li>
          •
          {{
            $t(
              "gitops.setting.add-git-provider.access-token.github-personal-access-token"
            )
          }}
        </li>
      </template>
      <template v-if="config.uiType == 'BITBUCKET_ORG'">
        <li>
          •
          {{
            $t(
              "gitops.setting.add-git-provider.access-token.bitbucket-resource-access-token"
            )
          }}
        </li>
        <li>
          •
          {{
            $t(
              "gitops.setting.add-git-provider.access-token.bitbucket-personal-access-token"
            )
          }}
        </li>
      </template>
      <template v-if="config.uiType == 'AZURE_DEVOPS'">
        <li>
          •
          {{
            $t(
              "gitops.setting.add-git-provider.access-token.azure-devops-personal-access-token"
            )
          }}
        </li>
      </template>
    </ul>
    <div>
      <div class="mt-4 textlabel">
        Access Token <span class="text-red-600">*</span>
      </div>
      <BBTextField
        class="mt-2 w-full"
        :placeholder="'ex. b9e0efc7a233403799b42620c60ff98c146895a27b6219912a215f4e2251cc3a'"
        :value="config.accessToken"
        @update:value="changeAccessToken($event)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { VCSConfig } from "@/types";

const props = defineProps<{
  config: VCSConfig;
}>();

const changeAccessToken = (value: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.config.accessToken = value;
};
</script>
