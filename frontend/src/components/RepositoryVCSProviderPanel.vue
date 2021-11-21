<template>
  <div class="textlabel">
    Choose the Git provider where your database schema scripts (.sql) are
    hosted. When you push the changed script to the Git repository, Bytebase
    will automatically applies the script change to the database.
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
      Visit
      <router-link class="normal-link" to="/setting/version-control"
        >Workspace Version Control</router-link
      >
      setting to add more Git providers.
    </template>
    <template v-else>
      Contact your Bytebase owner if you want other Git providers to appear
      here. Bytebase currently supports self-host GitLab EE/CE, and plan to add
      GitLab.com, GitHub Enterprise and GitHub.com later.
    </template>
  </div>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { useStore } from "vuex";
import {
  computed,
  onUnmounted,
  PropType,
  watchEffect,
} from "@vue/runtime-core";
import isEmpty from "lodash-es/isEmpty";
import {
  OAuthConfig,
  OAuthToken,
  OAuthWindowEvent,
  OAuthWindowEventPayload,
  openWindowForOAuth,
  ProjectRepositoryConfig,
  redirectURL,
  VCS,
} from "../types";
import { isOwner } from "../utils";

interface LocalState {
  selectedVCS?: VCS;
}

export default {
  name: "RepositoryVCSProviderPanel",
  emits: ["next"],
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepositoryConfig>,
    },
  },
  components: {},
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
          endpoint: `${state.selectedVCS!.instanceURL}/oauth/token`,
          applicationID: state.selectedVCS!.applicationID,
          secret: state.selectedVCS!.secret,
          redirectURL: redirectURL(),
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
    };

    onUnmounted(() => {
      window.removeEventListener(OAuthWindowEvent, eventListener);
    });

    const isCurrentUserOwner = computed(() => {
      return isOwner(currentUser.value.role);
    });

    const selectVCS = (vcs: VCS) => {
      state.selectedVCS = vcs;
      const newWindow = openWindowForOAuth(
        `${vcs.instanceURL}/oauth/authorize`,
        vcs.applicationID
      );
      if (newWindow) {
        window.addEventListener(OAuthWindowEvent, eventListener, false);
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
