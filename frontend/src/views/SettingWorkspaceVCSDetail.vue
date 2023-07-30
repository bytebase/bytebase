<template>
  <div class="mt-4 space-y-4">
    <div class="flex justify-end">
      <div
        v-if="vcsUIType == 'GITLAB_SELF_HOST'"
        class="flex flex-row items-center space-x-2"
      >
        <div class="textlabel whitespace-nowrap">
          {{ $t("gitops.setting.add-git-provider.gitlab-self-host-ce-ee") }}
        </div>
        <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
      </div>
      <div
        v-else-if="vcsUIType == 'GITLAB_COM'"
        class="flex flex-row items-center space-x-2"
      >
        <div class="textlabel whitespace-nowrap">GitLab.com</div>
        <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
      </div>
      <div
        v-else-if="vcsUIType == 'GITHUB_COM'"
        class="flex flex-row items-center space-x-2"
      >
        <div class="textlabel whitespace-nowrap">GitHub.com</div>
        <img class="h-6 w-auto" src="../assets/github-logo.svg" />
      </div>
      <div
        v-else-if="vcsUIType == 'BITBUCKET_ORG'"
        class="flex flex-row items-center space-x-2"
      >
        <div class="textlabel whitespace-nowrap">Bitbucket.org</div>
        <img class="h-6 w-auto" src="../assets/bitbucket-logo.svg" />
      </div>
    </div>

    <div>
      <label for="instanceurl" class="textlabel">
        {{ $t("common.instance") }} URL
      </label>
      <input
        id="instanceurl"
        name="instanceurl"
        type="text"
        class="textfield mt-1 w-full"
        disabled="true"
        :value="vcs?.url"
      />
    </div>

    <div>
      <label for="name" class="textlabel">
        {{ $t("gitops.setting.add-git-provider.basic-info.display-name") }}
      </label>
      <p class="mt-1 textinfolabel">
        {{
          $t("gitops.setting.add-git-provider.basic-info.display-name-label")
        }}
      </p>
      <input
        id="name"
        v-model="state.title"
        name="name"
        type="text"
        class="textfield mt-1 w-full"
      />
    </div>

    <div>
      <label for="applicationid" class="textlabel">
        {{ $t("common.application") }} ID
      </label>
      <p class="mt-1 textinfolabel">
        <template v-if="vcsUIType == 'GITLAB_SELF_HOST'">
          {{
            $t(
              "gitops.setting.git-provider.gitlab-self-host-application-id-label"
            )
          }}
          <a :href="adminApplicationUrl" target="_blank" class="normal-link">{{
            $t("gitops.setting.git-provider.view-in-gitlab")
          }}</a>
        </template>
        <template v-else-if="vcsUIType == 'GITLAB_COM'">
          {{
            $t("gitops.setting.git-provider.gitlab-com-application-id-label")
          }}
          <a :href="adminApplicationUrl" target="_blank" class="normal-link">{{
            $t("gitops.setting.git-provider.view-in-gitlab")
          }}</a>
        </template>
        <template v-else-if="vcsUIType == 'GITHUB_COM'">
          {{ $t("gitops.setting.git-provider.github-application-id-label") }}
        </template>
      </p>
      <input
        id="applicationid"
        v-model="state.applicationId"
        name="applicationid"
        type="text"
        class="textfield mt-1 w-full"
      />
    </div>

    <div>
      <label for="secret" class="textlabel"> Secret </label>
      <p class="mt-1 textinfolabel">
        <template v-if="vcsUIType == 'GITLAB_SELF_HOST'">
          {{ $t("gitops.setting.git-provider.gitlab-self-host-secret-label") }}
        </template>
        <template v-else-if="vcsUIType == 'GITLAB_COM'">
          {{ $t("gitops.setting.git-provider.gitlab-com-secret-label") }}
        </template>
        <template v-else-if="vcsUIType == 'GITHUB_COM'">
          {{ $t("gitops.setting.git-provider.secret-label-github") }}
        </template>
      </p>
      <input
        id="secret"
        v-model="state.secret"
        name="secret"
        type="text"
        class="textfield mt-1 w-full"
        :placeholder="$t('common.sensitive-placeholder')"
      />
    </div>

    <div class="pt-4 flex border-t justify-between">
      <template v-if="repositoryList.length == 0">
        <BBButtonConfirm
          :style="'DELETE'"
          :button-text="$t('gitops.setting.git-provider.delete')"
          :ok-text="$t('common.delete')"
          :confirm-title="
            $t('gitops.setting.git-provider.delete-confirm', {
              name: vcs?.title,
            })
          "
          :require-confirm="true"
          @confirm="deleteVCS"
        />
      </template>
      <template v-else>
        <div class="mt-1 textinfolabel">
          {{ $t("gitops.setting.git-provider.delete-forbidden") }}
        </div>
      </template>
      <div>
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="cancel"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          :disabled="!allowUpdate"
          @click.prevent="doUpdate"
        >
          {{ $t("common.update") }}
        </button>
      </div>
    </div>
  </div>

  <div class="py-6">
    <div class="text-lg leading-6 font-medium text-main">
      {{ $t("repository.linked") + ` (${repositoryList.length})` }}
    </div>
    <div class="mt-4">
      <RepositoryTable :repository-list="repositoryList" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, watchEffect, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import RepositoryTable from "../components/RepositoryTable.vue";
import isEmpty from "lodash-es/isEmpty";
import { idFromSlug, getVCSUIType } from "../utils";
import {
  openWindowForOAuth,
  OAuthWindowEventPayload,
  VCSUIType,
} from "../types";
import { pushNotification, useRepositoryV1Store, useVCSV1Store } from "@/store";
import {
  OAuthToken,
  ExternalVersionControl,
  ExternalVersionControl_Type,
} from "@/types/proto/v1/externalvs_service";

interface LocalState {
  title: string;
  applicationId: string;
  secret: string;
  oAuthResultCallback?: (token: OAuthToken | undefined) => void;
}

const props = defineProps({
  vcsSlug: {
    required: true,
    type: String,
  },
});

const vcsV1Store = useVCSV1Store();
const repositoryV1Store = useRepositoryV1Store();
const router = useRouter();

const vcs = computed((): ExternalVersionControl | undefined => {
  return vcsV1Store.getVCSByUid(idFromSlug(props.vcsSlug));
});

const vcsUIType = computed((): VCSUIType => {
  if (vcs.value) {
    return getVCSUIType(vcs.value);
  }
  return "GITLAB_SELF_HOST";
});

const state = reactive<LocalState>({
  title: vcs.value?.title ?? "",
  applicationId: vcs.value?.applicationId ?? "",
  secret: "",
});

onMounted(() => {
  window.addEventListener("bb.oauth.register-vcs", eventListener, false);
});

onUnmounted(() => {
  window.removeEventListener("bb.oauth.register-vcs", eventListener);
});

const eventListener = (event: Event) => {
  const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
  if (isEmpty(payload.error)) {
    if (
      vcs.value?.type === ExternalVersionControl_Type.GITLAB ||
      vcs.value?.type === ExternalVersionControl_Type.GITHUB ||
      vcs.value?.type === ExternalVersionControl_Type.BITBUCKET
    ) {
      vcsV1Store
        .exchangeToken({
          code: payload.code,
          vcsName: vcs.value.name,
          clientId: state.applicationId,
          clientSecret: state.secret,
        })
        .then((token: OAuthToken) => {
          state.oAuthResultCallback!(token);
        })
        .catch(() => {
          state.oAuthResultCallback!(undefined);
        });
    }
  } else {
    state.oAuthResultCallback!(undefined);
  }
};

watchEffect(async () => {
  if (vcs.value) {
    await repositoryV1Store.fetchRepositoryListByVCS(vcs.value.name);
  }
});

const adminApplicationUrl = computed(() => {
  if (vcsUIType.value == "GITLAB_SELF_HOST") {
    return `${vcs.value?.url}/admin/applications`;
  } else if (vcsUIType.value == "GITLAB_COM") {
    return "https://gitlab.com/-/profile/applications";
  }
  return "";
});

const repositoryList = computed(() =>
  repositoryV1Store.getRepositoryListByVCS(vcs.value?.name ?? "")
);

const allowUpdate = computed(() => {
  return (
    state.title != vcs.value?.title ||
    state.applicationId != vcs.value?.applicationId ||
    !isEmpty(state.secret)
  );
});

const doUpdate = () => {
  if (!vcs.value) {
    return;
  }

  if (
    state.applicationId != vcs.value.applicationId ||
    !isEmpty(state.secret)
  ) {
    let authorizeUrl = `${vcs.value.url}/oauth/authorize`;
    if (vcs.value.type === ExternalVersionControl_Type.GITHUB) {
      authorizeUrl = `https://github.com/login/oauth/authorize`;
    } else if (vcs.value.type === ExternalVersionControl_Type.BITBUCKET) {
      authorizeUrl = `https://bitbucket.org/site/oauth2/authorize`;
    } else if (vcs.value.type === ExternalVersionControl_Type.AZURE_DEVOPS) {
      authorizeUrl = "https://app.vssps.visualstudio.com/oauth2/authorize";
    }
    const newWindow = openWindowForOAuth(
      authorizeUrl,
      state.applicationId,
      "bb.oauth.register-vcs",
      vcs.value.type
    );
    if (newWindow) {
      state.oAuthResultCallback = (token: OAuthToken | undefined) => {
        if (!vcs.value) {
          return;
        }
        if (token) {
          const vcsPatch: Partial<ExternalVersionControl> = {
            name: vcs.value.name,
          };
          if (state.title != vcs.value.title) {
            vcsPatch.title = state.title;
          }
          if (state.applicationId != vcs.value.applicationId) {
            vcsPatch.applicationId = state.applicationId;
          }
          if (!isEmpty(state.secret)) {
            vcsPatch.secret = state.secret;
          }
          vcsV1Store
            .updateVCS(vcsPatch)
            .then((vcs: ExternalVersionControl | undefined) => {
              if (!vcs) {
                return;
              }
              pushNotification({
                module: "bytebase",
                style: "SUCCESS",
                title: `Successfully updated '${vcs.title}'`,
              });
            });
        } else {
          // If the application ID mismatches, the OAuth workflow will stop early.
          // So the only possibility to reach here is we have a matching application ID, while
          // we failed to exchange a token, and it's likely we are requesting with a wrong secret.
          let description = "";
          if (vcs.value.type == ExternalVersionControl_Type.GITLAB) {
            description =
              "Please make sure Secret matches the one from your GitLab instance Application.";
          } else if (vcs.value.type == ExternalVersionControl_Type.GITHUB) {
            description =
              "Please make sure Client secret matches the one from your GitHub.com Application.";
          } else if (vcs.value.type == ExternalVersionControl_Type.BITBUCKET) {
            description =
              "Please make sure Secret matches the one from your Bitbucket.org consumer.";
          }
          pushNotification({
            module: "bytebase",
            style: "CRITICAL",
            title: `Failed to update '${vcs.value.title}'`,
            description: description,
          });
        }
      };
    }
  } else if (state.title != vcs.value.title) {
    const vcsPatch: Partial<ExternalVersionControl> = {
      name: vcs.value.name,
      title: state.title,
    };
    vcsV1Store
      .updateVCS(vcsPatch)
      .then((updatedVCS: ExternalVersionControl | undefined) => {
        if (!updatedVCS) {
          return;
        }
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: `Successfully updated '${updatedVCS.title}'`,
        });
      });
  }
};

const cancel = () => {
  router.push({
    name: "setting.workspace.gitops",
  });
};

const deleteVCS = () => {
  if (!vcs.value) {
    return;
  }
  const title = vcs.value.title;
  vcsV1Store.deleteVCS(vcs.value.name).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: `Successfully deleted '${title}'`,
    });
    router.push({
      name: "setting.workspace.gitops",
    });
  });
};
</script>
