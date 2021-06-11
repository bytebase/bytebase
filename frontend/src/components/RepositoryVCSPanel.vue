<template>
  <div v-for="(vcs, index) in vcsList" :key="index">
    <button
      type="button"
      class="btn-normal items-center space-x-2"
      @click.prevent="selectVCS(vcs)"
    >
      <template v-if="vcs.type.startsWith('GITLAB')">
        <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
      </template>
      <span>{{ vcs.name }}</span>
    </button>
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
  OAuthWindowEvent,
  OAuthWindowEventPayload,
  openWindowForOAuth,
  ProjectRepoConfig,
  redirectURL,
  VCS,
} from "../types";

interface LocalState {
  selectedVCS?: VCS;
}

export default {
  name: "RepositoryVCSPanel",
  emits: ["next"],
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepoConfig>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({});

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
          applicationId: state.selectedVCS!.applicationId,
          secret: state.selectedVCS!.secret,
          redirectURL: redirectURL(),
        };
        store
          .dispatch("gitlab/exchangeToken", {
            oAuthConfig,
            code: payload.code,
          })
          .then((token: string) => {
            props.config.vcs = state.selectedVCS!;
            props.config.accessToken = token;
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

    const selectVCS = (vcs: VCS) => {
      state.selectedVCS = vcs;
      const newWindow = openWindowForOAuth(
        `${vcs.instanceURL}/oauth/authorize`,
        vcs.applicationId
      );
      if (newWindow) {
        window.addEventListener(OAuthWindowEvent, eventListener, false);
      }
    };

    return {
      state,
      vcsList,
      selectVCS,
    };
  },
};
</script>
