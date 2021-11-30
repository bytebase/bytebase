<template>
  <div class="mt-4 space-y-4">
    <div class="flex justify-end">
      <div
        v-if="vcs.type == 'GITLAB_SELF_HOST'"
        class="flex flex-row items-center space-x-2"
      >
        <div class="textlabel whitespace-nowrap">
          Self-host GitLab Enterprise Edition (EE) or Community Edition (CE)
        </div>
        <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
      </div>
    </div>

    <div>
      <label for="instanceurl" class="textlabel"> Instance URL </label>
      <p class="mt-1 textinfolabel">
        <template v-if="vcs.type == 'GITLAB_SELF_HOST'">
          Your GitLab instance location.
        </template>
      </p>
      <input
        id="instanceurl"
        name="instanceurl"
        type="text"
        class="textfield mt-1 w-full"
        disabled="true"
        :value="vcs.instanceURL"
      />
    </div>

    <div>
      <label for="name" class="textlabel"> Display name </label>
      <p class="mt-1 textinfolabel">
        An optional display name to help identifying among different configs
        using the same Git provider.
      </p>
      <input
        id="name"
        v-model="state.name"
        name="name"
        type="text"
        class="textfield mt-1 w-full"
      />
    </div>

    <div>
      <label for="applicationid" class="textlabel"> Application ID </label>
      <p class="mt-1 textinfolabel">
        <template v-if="vcs.type == 'GITLAB_SELF_HOST'">
          Application ID for the registered GitLab instance-wide OAuth
          application.
          <a :href="adminApplicationURL" target="_blank" class="normal-link"
            >View in GitLab</a
          >
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
        <template v-if="vcs.type == 'GITLAB_SELF_HOST'">
          Secret for the registered GitLab instance-wide OAuth application.
        </template>
      </p>
      <input
        id="secret"
        v-model="state.secret"
        name="secret"
        type="text"
        class="textfield mt-1 w-full"
        placeholder="sensitive - write only"
      />
    </div>

    <div class="pt-4 flex border-t justify-between">
      <template v-if="repositoryList.length == 0">
        <BBButtonConfirm
          :style="'DELETE'"
          :button-text="'Delete this Git provider'"
          :ok-text="'Delete'"
          :confirm-title="`Delete Git provider '${vcs.name}'?`"
          :require-confirm="true"
          @confirm="deleteVCS"
        />
      </template>
      <template v-else>
        <div class="mt-1 textinfolabel">
          To delete this provider, unlink all repositories first.
        </div>
      </template>
      <div>
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="cancel"
        >
          Cancel
        </button>
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          :disabled="!allowUpdate"
          @click.prevent="doUpdate"
        >
          Update
        </button>
      </div>
    </div>
  </div>

  <div class="py-6">
    <div class="text-lg leading-6 font-medium text-main">
      {{ `Linked repositories (${repositoryList.length})` }}
    </div>
    <div class="mt-4">
      <RepositoryTable :repository-list="repositoryList" />
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, computed, watchEffect } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import RepositoryTable from "../components/RepositoryTable.vue";
import isEmpty from "lodash-es/isEmpty";
import { idFromSlug } from "../utils";
import {
  VCS,
  VCSPatch,
  openWindowForOAuth,
  OAuthWindowEvent,
  OAuthWindowEventPayload,
  OAuthConfig,
  redirectURL,
  OAuthToken,
} from "../types";

interface LocalState {
  name: string;
  applicationId: string;
  secret: string;
  oAuthResultCallback?: (token: OAuthToken | undefined) => void;
}

export default {
  name: "SettingWorkspaceVCSDetail",
  components: { RepositoryTable },
  props: {
    vcsSlug: {
      required: true,
      type: String,
    },
  },
  setup(props) {
    const store = useStore();
    const router = useRouter();

    const vcs = computed((): VCS => {
      return store.getters["vcs/vcsById"](idFromSlug(props.vcsSlug));
    });

    const state = reactive<LocalState>({
      name: vcs.value.name,
      applicationId: vcs.value.applicationId,
      secret: "",
    });

    const eventListener = (event: Event) => {
      const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
      if (isEmpty(payload.error)) {
        if (vcs.value.type == "GITLAB_SELF_HOST") {
          const oAuthConfig: OAuthConfig = {
            endpoint: `${vcs.value.instanceURL}/oauth/token`,
            applicationId: state.applicationId,
            secret: state.secret,
            redirectURL: redirectURL(),
          };
          store
            .dispatch("gitlab/exchangeToken", {
              oAuthConfig,
              code: payload.code,
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

      window.removeEventListener(OAuthWindowEvent, eventListener);
    };

    const prepareRepositoryList = () => {
      store.dispatch("repository/fetchRepositoryListByVCSId", vcs.value.id);
    };

    watchEffect(prepareRepositoryList);

    const adminApplicationURL = computed(() => {
      if (vcs.value.type == "GITLAB_SELF_HOST") {
        return `${vcs.value.instanceURL}/admin/applications`;
      }
      return "";
    });

    const repositoryList = computed(() =>
      store.getters["repository/repositoryListByVCSId"](vcs.value.id)
    );

    const allowUpdate = computed(() => {
      return (
        state.name != vcs.value.name ||
        state.applicationId != vcs.value.applicationId ||
        !isEmpty(state.secret)
      );
    });

    const doUpdate = () => {
      if (
        state.applicationId != vcs.value.applicationId ||
        !isEmpty(state.secret)
      ) {
        const newWindow = openWindowForOAuth(
          `${vcs.value.instanceURL}/oauth/authorize`,
          vcs.value.applicationId
        );
        if (newWindow) {
          state.oAuthResultCallback = (token: OAuthToken | undefined) => {
            if (token) {
              const vcsPatch: VCSPatch = {};
              if (state.name != vcs.value.name) {
                vcsPatch.name = state.name;
              }
              if (state.applicationId != vcs.value.applicationId) {
                vcsPatch.applicationId = state.applicationId;
              }
              if (!isEmpty(state.secret)) {
                vcsPatch.secret = state.secret;
              }
              store
                .dispatch("vcs/patchVCS", {
                  vcsId: vcs.value.id,
                  vcsPatch,
                })
                .then((vcs: VCS) => {
                  store.dispatch("notification/pushNotification", {
                    module: "bytebase",
                    style: "SUCCESS",
                    title: `Successfully updated '${vcs.name}'`,
                  });
                });
            } else {
              var description = "";
              if (vcs.value.type == "GITLAB_SELF_HOST") {
                // If application id mismatches, the OAuth workflow will stop early.
                // So the only possibility to reach here is we have a matching application id, while
                // we failed to exchange a token, and it's likely we are requesting with a wrong secret.
                description =
                  "Please make sure Secret matches the one from your GitLab instance Application.";
              }
              store.dispatch("notification/pushNotification", {
                module: "bytebase",
                style: "CRITICAL",
                title: `Failed to update '${vcs.value.name}'`,
                description: description,
              });
            }
          };
          window.addEventListener(OAuthWindowEvent, eventListener, false);
        }
      } else if (state.name != vcs.value.name) {
        const vcsPatch: VCSPatch = {
          name: state.name,
        };
        store
          .dispatch("vcs/patchVCS", {
            vcsId: vcs.value.id,
            vcsPatch,
          })
          .then((updatedVCS: VCS) => {
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "SUCCESS",
              title: `Successfully updated '${updatedVCS.name}'`,
            });
          });
      }
    };

    const cancel = () => {
      router.push({
        name: "setting.workspace.version-control",
      });
    };

    const deleteVCS = () => {
      const name = vcs.value.name;
      store.dispatch("vcs/deleteVCSById", vcs.value.id).then(() => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "SUCCESS",
          title: `Successfully deleted '${name}'`,
        });
        router.push({
          name: "setting.workspace.version-control",
        });
      });
    };

    return {
      state,
      vcs,
      repositoryList,
      adminApplicationURL,
      allowUpdate,
      doUpdate,
      cancel,
      deleteVCS,
    };
  },
};
</script>
