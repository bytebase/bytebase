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
import { computed, onUnmounted, watchEffect } from "@vue/runtime-core";
import isEmpty from "lodash-es/isEmpty";
import {
  OAuthWindowEvent,
  OAuthWindowEventPayload,
  openWindowForVCSOAuth,
  VCS,
} from "../types";

interface LocalState {
  selectedVCS?: VCS;
}

var newWindow;

export default {
  name: "RepositoryVCSPanel",
  emits: ["select-vcs"],
  props: {},
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
        emit("select-vcs", state.selectedVCS);
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
      const newWindow = openWindowForVCSOAuth(vcs);
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
