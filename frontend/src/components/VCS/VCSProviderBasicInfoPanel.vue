<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div class="space-y-6">
    <div>
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
      <div class="mt-4 textlabel">
        {{ $t("gitops.setting.add-git-provider.basic-info.display-name") }}
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
      />
      <div>
        <ResourceIdField
          v-model:value="config.resourceId"
          class="max-w-full flex-nowrap"
          resource-type="vcs-provider"
          :resource-title="config.name"
          :validate="validateResourceId"
        />
      </div>
    </div>
    <div>
      <div class="mt-6 pt-6 border-t border-block-border textlabel">
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
            config.uiType == 'GITHUB_COM' ||
            config.uiType == 'GITHUB_ENTERPRISE'
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
import isEmpty from "lodash-es/isEmpty";
import { NRadio, NRadioGroup } from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, onUnmounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useVCSV1Store } from "@/store";
import { vcsProviderPrefix } from "@/store/modules/v1/common";
import type { VCSConfig } from "@/types";
import { TEXT_VALIDATION_DELAY } from "@/types";
import type { ResourceId, ValidatedMessage } from "@/types";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";
import { isUrl } from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";
import { vcsListByUIType } from "./utils";

interface LocalState {
  urlValidationTimer?: ReturnType<typeof setTimeout>;
  showUrlError: boolean;
}

const props = defineProps<{
  config: VCSConfig;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  showUrlError:
    !isEmpty(props.config.instanceUrl) && !isUrl(props.config.instanceUrl),
});
const vcsV1Store = useVCSV1Store();

onUnmounted(() => {
  if (state.urlValidationTimer) {
    clearInterval(state.urlValidationTimer);
  }
});

const namePlaceholder = computed((): string => {
  if (props.config.type === VCSProvider_Type.GITLAB) {
    if (props.config.uiType == "GITLAB_SELF_HOST") {
      return t("gitops.setting.add-git-provider.gitlab-self-host");
    } else if (props.config.uiType == "GITLAB_COM") {
      return "GitLab.com";
    }
  } else if (props.config.type === VCSProvider_Type.GITHUB) {
    if (props.config.uiType == "GITHUB_COM") {
      return "GitHub.com";
    } else if (props.config.uiType === "GITHUB_ENTERPRISE") {
      return "Self Host GitHub Enterprise";
    }
  } else if (props.config.type === VCSProvider_Type.BITBUCKET) {
    return "Bitbucket.org";
  } else if (props.config.type === VCSProvider_Type.AZURE_DEVOPS) {
    return "Azure DevOps";
  }
  return "";
});

const instanceUrlLabel = computed((): string => {
  switch (props.config.type) {
    case VCSProvider_Type.GITLAB:
      return t(
        "gitops.setting.add-git-provider.basic-info.gitlab-instance-url"
      );
    case VCSProvider_Type.GITHUB:
      return t(
        "gitops.setting.add-git-provider.basic-info.github-instance-url"
      );
    case VCSProvider_Type.BITBUCKET:
      return t(
        "gitops.setting.add-git-provider.basic-info.bitbucket-instance-url"
      );
    case VCSProvider_Type.AZURE_DEVOPS:
      return t("gitops.setting.add-git-provider.basic-info.azure-instance-url");
    default:
      return "";
  }
});

const instanceUrlPlaceholder = computed((): string => {
  if (props.config.type === VCSProvider_Type.GITLAB) {
    if (props.config.uiType == "GITLAB_SELF_HOST") {
      return "https://gitlab.example.com";
    } else if (props.config.uiType == "GITLAB_COM") {
      return "https://gitlab.com";
    }
  } else if (props.config.type === VCSProvider_Type.GITHUB) {
    if (props.config.uiType == "GITHUB_COM") {
      return "https://github.com";
    } else if (props.config.uiType == "GITHUB_ENTERPRISE") {
      return "https://github.companyname.com";
    }
  } else if (props.config.type === VCSProvider_Type.BITBUCKET) {
    return "https://bitbucket.org";
  }
  return "";
});

// github.com instance url is always https://github.com
const instanceUrlDisabled = computed((): boolean => {
  return (
    (props.config.type === VCSProvider_Type.GITHUB &&
      props.config.uiType == "GITHUB_COM") ||
    props.config.type === VCSProvider_Type.BITBUCKET ||
    props.config.type === VCSProvider_Type.AZURE_DEVOPS ||
    (props.config.type === VCSProvider_Type.GITLAB &&
      props.config.uiType == "GITLAB_COM")
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
      props.config.type = VCSProvider_Type.GITLAB;
      props.config.instanceUrl = "";
      props.config.name = t("gitops.setting.add-git-provider.gitlab-self-host");
      break;
    case "GITLAB_COM":
      props.config.type = VCSProvider_Type.GITLAB;
      props.config.instanceUrl = "https://gitlab.com";
      props.config.name = "GitLab.com";
      break;
    case "GITHUB_COM":
      props.config.type = VCSProvider_Type.GITHUB;
      props.config.instanceUrl = "https://github.com";
      props.config.name = "GitHub.com";
      break;
    case "GITHUB_ENTERPRISE":
      props.config.type = VCSProvider_Type.GITHUB;
      props.config.instanceUrl = "";
      props.config.name = "Self Host GitHub Enterprise";
      break;
    case "BITBUCKET_ORG":
      props.config.type = VCSProvider_Type.BITBUCKET;
      props.config.instanceUrl = "https://bitbucket.org";
      props.config.name = "Bitbucket.org";
      break;
    case "AZURE_DEVOPS":
      props.config.type = VCSProvider_Type.AZURE_DEVOPS;
      props.config.instanceUrl = "https://dev.azure.com";
      props.config.name = "Azure DevOps";
      break;
    default:
      break;
  }
};

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
