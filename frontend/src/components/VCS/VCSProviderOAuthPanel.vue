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
          <a
            :href="createOAuthApplicationUrl"
            target="_blank"
            class="normal-link"
            >{{
              $t("gitops.setting.add-git-provider.oauth-info.direct-link")
            }}</a
          >
        </li>
        <li>
          3.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.create-oauth-app")
          }}
          <div class="m-4 flex justify-center">
            <dl
              class="divide-y divide-block-border border border-block-border shadow rounded-lg"
            >
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Name
                </dt>
                <dd class="text-sm text-main">Bytebase</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Redirect URI
                </dt>
                <dd class="text-sm text-main items-center flex">
                  {{ redirectUrl() }}
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyRedirectURI"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Trusted
                </dt>
                <dd class="text-sm text-main">Yes</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Confidential
                </dt>
                <dd class="text-sm text-main">Yes</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Scopes
                </dt>
                <dd class="text-sm text-main">api</dd>
              </div>
            </dl>
          </div>
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
          <a
            :href="createOAuthApplicationUrl"
            target="_blank"
            class="normal-link"
            >{{
              $t("gitops.setting.add-git-provider.oauth-info.direct-link")
            }}</a
          >
        </li>
        <li>
          3.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.create-oauth-app")
          }}
          <div class="m-4 flex justify-center">
            <dl
              class="divide-y divide-block-border border border-block-border shadow rounded-lg"
            >
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Name
                </dt>
                <dd class="text-sm text-main">Bytebase</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Redirect URI
                </dt>
                <dd class="text-sm text-main items-center flex">
                  {{ redirectUrl() }}
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyRedirectURI"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Confidential
                </dt>
                <dd class="text-sm text-main">Yes</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Scopes
                </dt>
                <dd class="text-sm text-main">api</dd>
              </div>
            </dl>
          </div>
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
          <div class="m-4 flex justify-center">
            <dl
              class="divide-y divide-block-border border border-block-border shadow rounded-lg"
            >
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Application name
                </dt>
                <dd class="text-sm text-main">Bytebase</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Homepage URL
                </dt>
                <dd class="text-sm text-main items-center flex">
                  https://bytebase.com
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyHomepageURL"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Authorization callback URL
                </dt>
                <dd class="text-sm text-main items-center flex">
                  {{ redirectUrl() }}
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyRedirectURI"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
            </dl>
          </div>
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
          <div class="m-4 flex justify-center">
            <dl
              class="divide-y divide-block-border border border-block-border shadow rounded-lg"
            >
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Application name
                </dt>
                <dd class="text-sm text-main">Bytebase</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Homepage URL
                </dt>
                <dd class="text-sm text-main items-center flex">
                  https://bytebase.com
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyHomepageURL"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Authorization callback URL
                </dt>
                <dd class="text-sm text-main items-center flex">
                  {{ redirectUrl() }}
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyRedirectURI"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
            </dl>
          </div>
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
          <div class="m-4 flex justify-center">
            <dl
              class="divide-y divide-block-border border border-block-border shadow rounded-lg"
            >
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Name
                </dt>
                <dd class="text-sm text-main">Bytebase</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Callback URL
                </dt>
                <dd class="text-sm text-main items-center flex">
                  {{ redirectUrl() }}
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyRedirectURI"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Permissions
                </dt>
                <dd class="text-sm text-main">
                  Account > Read, Webhooks > Read and write, Repositories >
                  Write and Admin, Pipelines > Edit variables, Pull requests >
                  Write
                </dd>
              </div>
            </dl>
          </div>
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
          <a
            :href="createOAuthApplicationUrl"
            target="_blank"
            class="normal-link"
          >
            {{ $t("gitops.setting.add-git-provider.oauth-info.direct-link") }}
          </a>
        </li>
        <li>
          3.
          {{
            $t("gitops.setting.add-git-provider.oauth-info.create-oauth-app")
          }}
          <div class="m-4 flex justify-center">
            <dl
              class="divide-y divide-block-border border border-block-border shadow rounded-lg"
            >
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Name
                </dt>
                <dd class="text-sm text-main">Bytebase</dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Homepage URL
                </dt>
                <dd class="text-sm text-main items-center flex">
                  https://bytebase.com
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyHomepageURL"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2 items-center">
                <dt class="text-sm font-medium text-control-light text-right">
                  Redirect URL
                </dt>
                <dd class="text-sm text-main items-center flex">
                  {{ redirectUrl() }}
                  <button
                    tabindex="-1"
                    class="ml-1 text-sm font-medium text-control-light hover:bg-gray-100"
                    @click.prevent="copyRedirectURI"
                  >
                    <heroicons-outline:clipboard class="w-6 h-6" />
                  </button>
                </dd>
              </div>
              <div class="grid grid-cols-2 gap-4 px-4 py-2">
                <dt class="text-sm font-medium text-control-light text-right">
                  Scopes
                </dt>
                <!-- TODO: decide necessary scopes -->
                <dd class="text-sm text-main">
                  Code (full), Identity (read), Project and team (read), Build
                  (read and execute)
                </dd>
              </div>
            </dl>
          </div>
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
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { VCSConfig, redirectUrl } from "@/types";
import { toClipboard } from "@/utils";

const props = defineProps<{
  config: VCSConfig;
}>();

const { t } = useI18n();

const createOAuthApplicationUrl = computed((): string => {
  if (props.config.uiType == "GITLAB_SELF_HOST") {
    return `${props.config.instanceUrl}/admin/applications/new`;
  } else if (props.config.uiType == "GITLAB_COM") {
    return `https://gitlab.com/-/profile/applications`;
  } else if (props.config.uiType == "GITHUB_COM") {
    return `https://github.com/settings/applications/new`;
  } else if (props.config.uiType == "AZURE_DEVOPS") {
    return `https://app.vsaex.visualstudio.com/app/register`;
  }
  return "";
});

const copyHomepageURL = () => {
  toClipboard("https://bytebase.com").then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("gitops.setting.add-git-provider.oauth-info.copy-homepage-url"),
    });
  });
};

const copyRedirectURI = () => {
  toClipboard(redirectUrl()).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("gitops.setting.add-git-provider.oauth-info.copy-redirect-uri"),
    });
  });
};

const changeAccessToken = (value: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.config.accessToken = value;
};
</script>
