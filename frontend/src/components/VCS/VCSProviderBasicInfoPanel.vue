<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div class="space-y-6">
    <div v-if="create">
      <div class="textlabel">
        {{ $t("gitops.setting.add-git-provider.choose") }}
        <span class="text-red-600">*</span>
      </div>
      <div class="flex flex-wrap pt-4 radio-set-row gap-4">
        <NRadioGroup
          v-model:value="config.uiType"
          class="!flex flex-row justify-start items-center flex-wrap gap-x-2 gap-y-4"
        >
          <NRadio
            v-for="vcsWithUIType in vcsListByUIType"
            :key="vcsWithUIType.uiType"
            :value="vcsWithUIType.uiType"
            @change="changeUIType()"
          >
            <div class="flex space-x-1">
              <VCSIcon custom-class="h-5" :type="vcsWithUIType.type" />
              <span class="whitespace-nowrap">
                {{ vcsWithUIType.title }}
              </span>
            </div>
          </NRadio>
        </NRadioGroup>
      </div>
    </div>
    <div>
      <div class="textlabel">
        {{ $t("gitops.setting.add-git-provider.basic-info.display-name") }}
        <span class="text-red-600">*</span>
      </div>
      <p class="mt-1 textinfolabel">
        {{
          $t("gitops.setting.add-git-provider.basic-info.display-name-label")
        }}
      </p>
      <BBTextField
        v-model:value="config.name"
        class="mt-2 w-full"
        :placeholder="namePlaceholder"
        :required="true"
      />
      <ResourceIdField
        v-model:value="config.resourceId"
        class="max-w-full flex-nowrap mt-1.5"
        editing-class="mt-4"
        resource-type="vcs-provider"
        :suffix="true"
        :resource-title="config.name"
        :validate="validateResourceId"
        :readonly="!create"
      />
    </div>
    <div>
      <div class="textlabel flex items-center">
        <VCSIcon custom-class="h-4 mr-1" :type="config.type" />
        {{ instanceUrlLabel }} <span class="text-red-600">*</span>
      </div>
      <p class="mt-1 textinfolabel">
        {{
          $t(
            "gitops.setting.add-git-provider.basic-info.gitlab-instance-url-label"
          )
        }}
      </p>
      <BBTextField
        class="mt-2 w-full"
        :required="true"
        :value="config.instanceUrl"
        :placeholder="instanceUrlPlaceholder"
        :disabled="instanceUrlDisabled"
        @update:value="changeUrl($event)"
      />
      <p v-if="state.showUrlError" class="mt-2 text-sm text-error">
        {{
          $t("gitops.setting.add-git-provider.basic-info.instance-url-error")
        }}
      </p>
    </div>
    <div>
      <div class="mt-4 textlabel">
        Access Token <span class="text-red-600">*</span>
      </div>
      <ul class="textinfolabel space-y-2 mt-2">
        <template v-if="config.type == VCSType.GITLAB">
          <li>
            {{
              $t(
                "gitops.setting.add-git-provider.access-token.gitlab-personal-access-token"
              )
            }}
          </li>
          <li>• api</li>
          <li>• read repository</li>
        </template>
        <template v-if="config.type === VCSType.GITHUB">
          <li>
            <i18n-t
              keypath="gitops.setting.add-git-provider.access-token.github-personal-access-token"
            >
              <template #token>
                <a
                  href="https://github.com/settings/tokens?type=beta"
                  target="_blank"
                  class="normal-link"
                >
                  {{
                    $t(
                      "gitops.setting.add-git-provider.access-token.personal-access-token"
                    )
                  }}
                </a>
              </template>
            </i18n-t>
          </li>
          <li>• Metadata (Read-only)</li>
          <li>• Contents (Read-only)</li>
          <li>• Pull requests (Read and write)</li>
          <li>• Webhooks (Read and write)</li>
        </template>
        <template v-if="config.type == VCSType.BITBUCKET">
          <li>
            <i18n-t
              keypath="gitops.setting.add-git-provider.access-token.bitbucket-app-access-token"
            >
              <template #app_password>
                <a
                  href="https://bitbucket.org/account/settings/app-passwords"
                  target="_blank"
                  class="normal-link"
                >
                  {{
                    $t(
                      "gitops.setting.add-git-provider.access-token.bitbucket-app-password"
                    )
                  }}
                </a>
              </template>
            </i18n-t>
          </li>
          <li>• Account (Read)</li>
          <li>• Workspace membership Profile (Read)</li>
          <li>• Projects (Read)</li>
          <li>• Webhooks (Read & Write)</li>
          <li>• Repositories (Read & Write)</li>
          <li>• Pull requests (Read & Write)</li>
        </template>
        <template v-if="config.type == VCSType.AZURE_DEVOPS">
          <li>
            {{
              $t(
                "gitops.setting.add-git-provider.access-token.azure-devops-personal-access-token"
              )
            }}
          </li>
          <li>• Code (Read)</li>
          <li>• User Profile (Read)</li>
          <li>• Project and Team (Read)</li>
          <li>• Pull Request Threads (Read & Write)</li>
        </template>
      </ul>
      <BBTextField
        class="mt-2 w-full"
        :required="create"
        :placeholder="accessTokenPlaceholder"
        :value="config.accessToken"
        @update:value="changeAccessToken($event)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import isEmpty from "lodash-es/isEmpty";
import { NRadio, NRadioGroup } from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, onUnmounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useVCSProviderStore } from "@/store";
import { vcsProviderPrefix } from "@/store/modules/v1/common";
import type { VCSConfig } from "@/types";
import { TEXT_VALIDATION_DELAY } from "@/types";
import type { ResourceId, ValidatedMessage } from "@/types";
import { VCSType } from "@/types/proto/v1/common";
import { isUrl } from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";
import { vcsListByUIType } from "./utils";

interface LocalState {
  urlValidationTimer?: ReturnType<typeof setTimeout>;
  showUrlError: boolean;
}

const props = defineProps<{
  config: VCSConfig;
  create: boolean;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  showUrlError:
    !isEmpty(props.config.instanceUrl) && !isUrl(props.config.instanceUrl),
});
const vcsV1Store = useVCSProviderStore();

onUnmounted(() => {
  if (state.urlValidationTimer) {
    clearInterval(state.urlValidationTimer);
  }
});

const namePlaceholder = computed((): string => {
  if (props.config.type === VCSType.GITLAB) {
    if (props.config.uiType == "GITLAB_SELF_HOST") {
      return t("gitops.setting.add-git-provider.gitlab-self-host");
    } else if (props.config.uiType == "GITLAB_COM") {
      return "GitLab.com";
    }
  } else if (props.config.type === VCSType.GITHUB) {
    if (props.config.uiType == "GITHUB_COM") {
      return "GitHub.com";
    } else if (props.config.uiType === "GITHUB_ENTERPRISE") {
      return "Self Host GitHub Enterprise";
    }
  } else if (props.config.type === VCSType.BITBUCKET) {
    return "Bitbucket.org";
  } else if (props.config.type === VCSType.AZURE_DEVOPS) {
    return "Azure DevOps";
  }
  return "";
});

const instanceUrlLabel = computed((): string => {
  switch (props.config.type) {
    case VCSType.GITLAB:
      return t(
        "gitops.setting.add-git-provider.basic-info.gitlab-instance-url"
      );
    case VCSType.GITHUB:
      return t(
        "gitops.setting.add-git-provider.basic-info.github-instance-url"
      );
    case VCSType.BITBUCKET:
      return t(
        "gitops.setting.add-git-provider.basic-info.bitbucket-instance-url"
      );
    case VCSType.AZURE_DEVOPS:
      return t("gitops.setting.add-git-provider.basic-info.azure-instance-url");
    default:
      return "";
  }
});

const instanceUrlPlaceholder = computed((): string => {
  if (props.config.type === VCSType.GITLAB) {
    if (props.config.uiType == "GITLAB_SELF_HOST") {
      return "https://gitlab.example.com";
    } else if (props.config.uiType == "GITLAB_COM") {
      return "https://gitlab.com";
    }
  } else if (props.config.type === VCSType.GITHUB) {
    if (props.config.uiType == "GITHUB_COM") {
      return "https://github.com";
    } else if (props.config.uiType == "GITHUB_ENTERPRISE") {
      return "https://github.companyname.com";
    }
  } else if (props.config.type === VCSType.BITBUCKET) {
    return "https://bitbucket.org";
  }
  return "";
});

// github.com instance url is always https://github.com
const instanceUrlDisabled = computed((): boolean => {
  return (
    (props.config.type === VCSType.GITHUB &&
      props.config.uiType == "GITHUB_COM") ||
    props.config.type === VCSType.BITBUCKET ||
    props.config.type === VCSType.AZURE_DEVOPS ||
    (props.config.type === VCSType.GITLAB &&
      props.config.uiType == "GITLAB_COM") ||
    !props.create
  );
});

const changeUrl = (value: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.config.instanceUrl = value;

  if (state.urlValidationTimer) {
    clearInterval(state.urlValidationTimer);
  }
  // If text becomes valid, we immediately clear the error.
  // otherwise, we delay TEXT_VALIDATION_DELAY to do the validation in case there is continous keystroke.
  if (isUrl(props.config.instanceUrl)) {
    state.showUrlError = false;
  } else {
    state.urlValidationTimer = setTimeout(() => {
      // If error is already displayed, we hide the error only if there is valid input.
      // Otherwise, we hide the error if input is either empty or valid.
      if (state.showUrlError) {
        state.showUrlError = !isUrl(props.config.instanceUrl);
      } else {
        state.showUrlError =
          !isEmpty(props.config.instanceUrl) &&
          !isUrl(props.config.instanceUrl);
      }
    }, TEXT_VALIDATION_DELAY);
  }
};

// FIXME: Unexpected mutation of "config" prop. Do we care?
/* eslint-disable vue/no-mutating-props */
const changeUIType = () => {
  switch (props.config.uiType) {
    case "GITLAB_SELF_HOST":
      props.config.type = VCSType.GITLAB;
      props.config.instanceUrl = "";
      props.config.name = t("gitops.setting.add-git-provider.gitlab-self-host");
      break;
    case "GITLAB_COM":
      props.config.type = VCSType.GITLAB;
      props.config.instanceUrl = "https://gitlab.com";
      props.config.name = "GitLab.com";
      break;
    case "GITHUB_COM":
      props.config.type = VCSType.GITHUB;
      props.config.instanceUrl = "https://github.com";
      props.config.name = "GitHub.com";
      break;
    case "GITHUB_ENTERPRISE":
      props.config.type = VCSType.GITHUB;
      props.config.instanceUrl = "";
      props.config.name = "Self Host GitHub Enterprise";
      break;
    case "BITBUCKET_ORG":
      props.config.type = VCSType.BITBUCKET;
      props.config.instanceUrl = "https://bitbucket.org";
      props.config.name = "Bitbucket.org";
      break;
    case "AZURE_DEVOPS":
      props.config.type = VCSType.AZURE_DEVOPS;
      props.config.instanceUrl = "https://dev.azure.com";
      props.config.name = "Azure DevOps";
      break;
    default:
      break;
  }
};

const accessTokenPlaceholder = computed(() => {
  switch (props.config.type) {
    case VCSType.BITBUCKET:
      return "<bitbucket username>:<generated app password>";
    case VCSType.GITHUB:
      return "github_11AYD374I0mlFAa4HbedewxR_sdFZEbbismN5rNQtqKoiPckxHryntBmQLJCJBEYfsCTA5j0";
    case VCSType.GITLAB:
      return "glpat-dFZEbbismN5rNQtqKoiPckxHryntBmQLJCJBEYfs";
  }
  return "b9e0efc7a233403799b42620c60ff98c146895a27b6219912a215f4e2251cc3a";
});

const changeAccessToken = (value: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.config.accessToken = value;
};

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  try {
    const vcs = await vcsV1Store.getOrFetchVCSByName(
      `${vcsProviderPrefix}${resourceId}`
    );
    if (vcs) {
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
