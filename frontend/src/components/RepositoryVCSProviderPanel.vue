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
export default { name: "RepositoryVCSProviderPanel" };
</script>

<script setup lang="ts">
import { reactive, computed, watchEffect, onUnmounted, onMounted } from "vue";
import isEmpty from "lodash-es/isEmpty";
import { OAuthWindowEventPayload, openWindowForOAuth, VCS } from "../types";
import { isOwner } from "../utils";
import { pushNotification, useCurrentUser, useVcsStore } from "@/store";

interface LocalState {
  selectedVCS?: VCS;
}

const emit = defineEmits<{
  (event: "next"): void;
  (event: "set-vcs", payload: VCS): void;
  (event: "set-code", payload: string): void;
}>();

const vcsStore = useVcsStore();
const state = reactive<LocalState>({});

const currentUser = useCurrentUser();

const prepareVCSList = () => {
  vcsStore.fetchVCSList();
};

watchEffect(prepareVCSList);

onMounted(() => {
  window.addEventListener("bb.oauth.link-vcs-repository", eventListener, false);
});
onUnmounted(() => {
  window.removeEventListener("bb.oauth.link-vcs-repository", eventListener);
});

const vcsList = computed(() => {
  return vcsStore.getVCSList();
});

const eventListener = (event: Event) => {
  const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
  if (isEmpty(payload.error)) {
    emit("set-code", payload.code);
    emit("next");
  } else {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: payload.error,
    });
  }
};

const isCurrentUserOwner = computed(() => {
  return isOwner(currentUser.value.role);
});

const selectVCS = (vcs: VCS) => {
  state.selectedVCS = vcs;
  emit("set-vcs", vcs);
  openWindowForOAuth(
    `${vcs.instanceUrl}/oauth/authorize`,
    vcs.applicationId,
    "bb.oauth.link-vcs-repository"
  );
};
</script>
