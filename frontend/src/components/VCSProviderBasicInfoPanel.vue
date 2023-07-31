<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div class="textlabel">
    {{ $t("gitops.setting.add-git-provider.choose") }}
    <span class="text-red-600">*</span>
  </div>
  <div class="flex flex-wrap pt-4 radio-set-row gap-4">
    <label class="radio space-x-2">
      <input
        v-model="config.uiType"
        name="Self-host GitLab"
        tabindex="-1"
        type="radio"
        class="btn"
        value="GITLAB_SELF_HOST"
        @change="changeUIType()"
      />
      <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
      <span class="whitespace-nowrap">
        {{ $t("gitops.setting.add-git-provider.gitlab-self-host-ce-ee") }}
      </span>
    </label>
    <label class="radio space-x-2">
      <input
        v-model="config.uiType"
        name="GitLab.com"
        tabindex="-1"
        type="radio"
        class="btn"
        value="GITLAB_COM"
        @change="changeUIType()"
      />
      <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
      <span class="whitespace-nowrap">GitLab.com</span>
    </label>
    <label class="radio space-x-2">
      <input
        v-model="config.uiType"
        name="GitHub.com"
        tabindex="-1"
        type="radio"
        class="btn"
        value="GITHUB_COM"
        @change="changeUIType()"
      />
      <img class="h-6 w-auto" src="../assets/github-logo.svg" />
      <span class="whitespace-nowrap">GitHub.com</span>
    </label>
    <label class="radio space-x-2">
      <input
        v-model="config.uiType"
        name="GitHub Enterprise"
        tabindex="-1"
        type="radio"
        class="btn"
        value="GITHUB_ENTERPRISE"
        @change="changeUIType()"
      />
      <img class="h-6 w-auto" src="../assets/github-logo.svg" />
      <span class="whitespace-nowrap">GitHub Enterprise</span>
    </label>
    <label class="radio space-x-2">
      <input
        v-model="config.uiType"
        name="Bitbucket.org"
        tabindex="-1"
        type="radio"
        class="btn"
        value="BITBUCKET_ORG"
        @change="changeUIType()"
      />
      <img class="h-6 w-auto" src="../assets/bitbucket-logo.svg" />
      <span class="whitespace-nowrap">Bitbucket.org</span>
    </label>
    <label class="radio space-x-2">
      <input
        v-model="config.uiType"
        name="Azure DevOps"
        tabindex="-1"
        type="radio"
        class="btn"
        value="AZURE_DEVOPS"
        @change="changeUIType()"
      />
      <img class="h-6 w-auto" src="../assets/azure-devops-logo.svg" />
      <span class="whitespace-nowrap">Azure DevOps</span>
    </label>
  </div>
  <div class="mt-6 pt-6 border-t border-block-border textlabel">
    {{ instanceUrlLabel }} <span class="text-red-600">*</span>
  </div>
  <p class="mt-1 textinfolabel">
    {{
      $t("gitops.setting.add-git-provider.basic-info.gitlab-instance-url-label")
    }}
  </p>
  <BBTextField
    class="mt-2 w-full"
    :value="config.instanceUrl"
    :placeholder="instanceUrlPlaceholder"
    :disabled="instanceUrlDisabled"
    @input="changeUrl(($event.target as HTMLInputElement).value)"
  />
  <p v-if="state.showUrlError" class="mt-2 text-sm text-error">
    {{ $t("gitops.setting.add-git-provider.basic-info.instance-url-error") }}
  </p>
  <div class="mt-4 textlabel">
    {{ $t("gitops.setting.add-git-provider.basic-info.display-name") }}
  </div>
  <p class="mt-1 textinfolabel">
    {{ $t("gitops.setting.add-git-provider.basic-info.display-name-label") }}
  </p>
  <BBTextField
    class="mt-2 w-full"
    :placeholder="namePlaceholder"
    :value="config.name"
    @input="config.name = ($event.target as HTMLInputElement).value"
  />
</template>

<script lang="ts">
import {
  computed,
  defineComponent,
  onUnmounted,
  PropType,
  reactive,
} from "vue";
import isEmpty from "lodash-es/isEmpty";
import { TEXT_VALIDATION_DELAY, VCSConfig } from "../types";
import { isUrl, isDev } from "../utils";
import { useI18n } from "vue-i18n";
import { ExternalVersionControl_Type } from "@/types/proto/v1/externalvs_service";

interface LocalState {
  urlValidationTimer?: ReturnType<typeof setTimeout>;
  showUrlError: boolean;
}

export default defineComponent({
  name: "VCSProviderBasicInfoPanel",
  props: {
    config: {
      required: true,
      type: Object as PropType<VCSConfig>,
    },
  },
  setup(props) {
    const { t } = useI18n();
    const state = reactive<LocalState>({
      showUrlError:
        !isEmpty(props.config.instanceUrl) && !isUrl(props.config.instanceUrl),
    });

    onUnmounted(() => {
      if (state.urlValidationTimer) {
        clearInterval(state.urlValidationTimer);
      }
    });

    const namePlaceholder = computed((): string => {
      if (props.config.type === ExternalVersionControl_Type.GITLAB) {
        if (props.config.uiType == "GITLAB_SELF_HOST") {
          return t("gitops.setting.add-git-provider.gitlab-self-host");
        } else if (props.config.uiType == "GITLAB_COM") {
          return "GitLab.com";
        }
      } else if (props.config.type === ExternalVersionControl_Type.GITHUB) {
        if (props.config.uiType == "GITHUB_COM") {
          return "GitHub.com";
        } else if (props.config.uiType === "GITHUB_ENTERPRISE") {
          return "Self Host GitHub Enterprise";
        }
      } else if (props.config.type === ExternalVersionControl_Type.BITBUCKET) {
        return "Bitbucket.org";
      }
      return "";
    });

    const instanceUrlLabel = computed((): string => {
      switch (props.config.type) {
        case ExternalVersionControl_Type.GITLAB:
          return t(
            "gitops.setting.add-git-provider.basic-info.gitlab-instance-url"
          );
        case ExternalVersionControl_Type.GITHUB:
          return t(
            "gitops.setting.add-git-provider.basic-info.github-instance-url"
          );
        case ExternalVersionControl_Type.BITBUCKET:
          return t(
            "gitops.setting.add-git-provider.basic-info.bitbucket-instance-url"
          );
        case ExternalVersionControl_Type.AZURE_DEVOPS:
          return t(
            "gitops.setting.add-git-provider.basic-info.azure-instance-url"
          );
        default:
          return "";
      }
    });

    const instanceUrlPlaceholder = computed((): string => {
      if (props.config.type === ExternalVersionControl_Type.GITLAB) {
        if (props.config.uiType == "GITLAB_SELF_HOST") {
          return "https://gitlab.example.com";
        } else if (props.config.uiType == "GITLAB_COM") {
          return "https://gitlab.com";
        }
      } else if (props.config.type === ExternalVersionControl_Type.GITHUB) {
        if (props.config.uiType == "GITHUB_COM") {
          return "https://github.com";
        } else if (props.config.uiType == "GITHUB_ENTERPRISE") {
          return "https://github.companyname.com";
        }
      } else if (props.config.type === ExternalVersionControl_Type.BITBUCKET) {
        return "https://bitbucket.org";
      }
      return "";
    });

    // github.com instance url is always https://github.com
    const instanceUrlDisabled = computed((): boolean => {
      return (
        (props.config.type === ExternalVersionControl_Type.GITHUB &&
          props.config.uiType == "GITHUB_COM") ||
        props.config.type === ExternalVersionControl_Type.BITBUCKET ||
        props.config.type === ExternalVersionControl_Type.AZURE_DEVOPS ||
        (props.config.type === ExternalVersionControl_Type.GITLAB &&
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
          props.config.type = ExternalVersionControl_Type.GITLAB;
          props.config.instanceUrl = "";
          props.config.name = t(
            "gitops.setting.add-git-provider.gitlab-self-host"
          );
          break;
        case "GITLAB_COM":
          props.config.type = ExternalVersionControl_Type.GITLAB;
          props.config.instanceUrl = "https://gitlab.com";
          props.config.name = "GitLab.com";
          break;
        case "GITHUB_COM":
          props.config.type = ExternalVersionControl_Type.GITHUB;
          props.config.instanceUrl = "https://github.com";
          props.config.name = "GitHub.com";
          break;
        case "GITHUB_ENTERPRISE":
          props.config.type = ExternalVersionControl_Type.GITHUB;
          props.config.instanceUrl = "";
          props.config.name = "Self Host GitHub Enterprise";
          break;
        case "BITBUCKET_ORG":
          props.config.type = ExternalVersionControl_Type.BITBUCKET;
          props.config.instanceUrl = "https://bitbucket.org";
          props.config.name = "Bitbucket.org";
          break;
        case "AZURE_DEVOPS":
          props.config.type = ExternalVersionControl_Type.AZURE_DEVOPS;
          props.config.instanceUrl = "https://dev.azure.com";
          props.config.name = "Azure DevOps";
          break;
        default:
          break;
      }
    };
    /* eslint-enable */

    return {
      state,
      namePlaceholder,
      instanceUrlLabel,
      instanceUrlPlaceholder,
      instanceUrlDisabled,
      changeUrl,
      changeUIType,
      isDev,
    };
  },
});
</script>
