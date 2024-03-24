<template>
  <div class="space-y-4">
    <div class="textlabel">
      <template v-if="config.uiType == 'GITLAB_SELF_HOST'">
        {{
          $t(
            "gitops.setting.add-git-provider.oauth-info.gitlab-self-host-register-oauth-application"
          )
        }}
      </template>
      <template v-else-if="config.uiType == 'GITLAB_COM'">
        {{
          $t(
            "gitops.setting.add-git-provider.oauth-info.gitlab-com-register-oauth-application"
          )
        }}
      </template>
      <template v-else-if="config.uiType == 'GITHUB_COM'">
        {{
          $t(
            "gitops.setting.add-git-provider.oauth-info.github-register-oauth-application"
          )
        }}
      </template>
      <template v-else-if="config.uiType == 'GITHUB_ENTERPRISE'">
        {{
          $t(
            "gitops.setting.add-git-provider.oauth-info.github-register-oauth-application"
          )
        }}
      </template>
      <template v-else-if="config.uiType == 'BITBUCKET_ORG'">
        {{
          $t(
            "gitops.setting.add-git-provider.oauth-info.bitbucket-register-oauth-application"
          )
        }}
      </template>
      <template v-else-if="config.uiType == 'AZURE_DEVOPS'">
        {{
          $t(
            "gitops.setting.add-git-provider.oauth-info.azure-register-oauth-application"
          )
        }}
      </template>
    </div>
    <ol class="textinfolabel space-y-2">
      <template v-if="config.uiType == 'GITLAB_SELF_HOST'">
        <li>
          1.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.gitlab-self-host-login-as-admin"
            )
          }}
          <img class="w-auto" src="../../assets/gitlab_admin_area.png" />
        </li>
        <li>
          2.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.gitlab-self-host-visit-admin-page"
            )
          }}
        </li>
        <li>
          3.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.create-oauth-app")
          }}
        </li>
        <li>
          4.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.gitlab-paste-oauth-info"
            )
          }}
        </li>
      </template>
      <template v-else-if="config.uiType == 'GITLAB_COM'">
        <li>
          1.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.gitlab-com-login")
          }}
        </li>
        <li>
          2.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.gitlab-com-visit-application-page"
            )
          }}
        </li>
        <li>
          3.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.create-oauth-app")
          }}
        </li>
        <li>
          4.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.gitlab-paste-oauth-info"
            )
          }}
        </li>
      </template>
      <template v-else-if="config.uiType == 'GITHUB_COM'">
        <li>
          1.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.github-login-as-admin"
            )
          }}
          <img class="w-auto" src="../../assets/github_admin_settings.png" />
        </li>
        <li>
          2.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.github-visit-admin-page"
            )
          }}
        </li>
        <li>
          3.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.create-oauth-app")
          }}
        </li>
        <li>
          4.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.github-paste-oauth-info"
            )
          }}
        </li>
      </template>
      <template v-else-if="config.uiType == 'GITHUB_ENTERPRISE'">
        <li>
          1.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.github-enterprise-login-as-admin"
            )
          }}
          <img class="w-auto" src="../../assets/github_admin_settings.png" />
        </li>
        <li>
          2.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.github-visit-admin-page"
            )
          }}
        </li>
        <li>
          3.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.create-oauth-app")
          }}
        </li>
        <li>
          4.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.github-paste-oauth-info"
            )
          }}
        </li>
      </template>
      <template v-else-if="config.uiType == 'BITBUCKET_ORG'">
        <li>
          1.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.bitbucket-login-as-admin"
            )
          }}
        </li>
        <li>
          2.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.bitbucket-visit-admin-page"
            )
          }}
        </li>
        <li>
          3.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.create-oauth-app")
          }}
        </li>
        <li>
          4.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.bitbucket-paste-oauth-info"
            )
          }}
        </li>
      </template>
      <template v-else-if="config.uiType == 'AZURE_DEVOPS'">
        <li>
          1.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.azure-devops-login")
          }}
        </li>
        <li>
          2.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.azure-devops-visit-application-page"
            )
          }}
        </li>
        <li>
          3.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.create-oauth-app")
          }}
        </li>
        <li>
          4.
          {{
            $t(
              "gitops.setting.add-git-provider.oauth-info.azure-paste-oauth-info"
            )
          }}
        </li>
      </template>
    </ol>
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
