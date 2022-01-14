<template>
  <div class="textlabel">
    {{ $t("repository.choose-git-provider-description") }}
  </div>
  <div class="mt-4 flex flex-wrap">
    <template v-for="(vcs, index) in vcsList" :key="index">
      <button
        type="button"
        class="btn-normal items-center space-x-2 mx-2 my-2"
        @click.prevent="selectVCS(vcs)"
      >
        <template v-if="vcs.type.startsWith('GITLAB')">
          <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
        </template>
        <span>{{ vcs.name }}</span>
      </button>
    </template>
  </div>
  <div class="mt-2 textinfolabel">
    <template v-if="isCurrentUserOwner">
      <i18n-t keypath="repository.choose-git-provider-visit-workspace">
        <template #workspace>
          <router-link class="normal-link" to="/setting/version-control"
            >{{ $t("common.workspace") }} -
            {{ $t("common.version-control") }}</router-link
          >
        </template>
      </i18n-t>
    </template>
    <template v-else>
      {{ $t("repository.choose-git-provider-contact-workspace-owner") }}
    </template>
  </div>
</template>

<script lang="ts">
import { useStore } from "vuex";
import { reactive, computed, PropType, watchEffect } from "vue";
import isEmpty from "lodash-es/isEmpty";
import {
  OAuthConfig,
  OAuthToken,
  getOAuthEventName,
  OAuthWindowEventPayload,
  openWindowForOAuth,
  ProjectRepositoryConfig,
  redirectUrl,
  VCS,
} from "../types";
import { isOwner } from "../utils";

interface LocalState {
  selectedVCS?: VCS;
}

export default {
  name: "RepositoryVCSProviderPanel",
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepositoryConfig>,
    },
  },
  emits: ["next"],
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({});

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareVCSList = () => {
      store.dispatch("vcs/fetchVCSList");
    };

    watchEffect(prepareVCSList);

    const vcsList = computed(() => {
      return store.getters["vcs/vcsList"]();
    });

    const eventListener = (event: Event) => {
      const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
      if (isEmpty(payload.error)) {
        props.config.code = payload.code;
        const oAuthConfig: OAuthConfig = {
          endpoint: `${state.selectedVCS!.instanceUrl}/oauth/token`,
          applicationId: state.selectedVCS!.applicationId,
          secret: state.selectedVCS!.secret,
          redirectUrl: redirectUrl(),
        };
        store
          .dispatch("gitlab/exchangeToken", {
            oAuthConfig,
            code: payload.code,
          })
          .then((token: OAuthToken) => {
            props.config.vcs = state.selectedVCS!;
            props.config.token = token;
            emit("next");
          });
      } else {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "CRITICAL",
          title: payload.error,
        });
      }
      window.removeEventListener(
        getOAuthEventName("register_vcs"),
        eventListener
      );
    };

    const isCurrentUserOwner = computed(() => {
      return isOwner(currentUser.value.role);
    });

    const selectVCS = (vcs: VCS) => {
      state.selectedVCS = vcs;
      const newWindow = openWindowForOAuth(
        `${vcs.instanceUrl}/oauth/authorize`,
        vcs.applicationId,
        "register_vcs"
      );
      if (newWindow) {
        window.addEventListener(
          getOAuthEventName("register_vcs"),
          eventListener,
          false
        );
      }
    };

    return {
      state,
      currentUser,
      vcsList,
      isCurrentUserOwner,
      selectVCS,
    };
  },
};
</script>
