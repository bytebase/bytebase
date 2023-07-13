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
        <template v-if="vcs.type === ExternalVersionControl_Type.GITLAB">
          <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
        </template>
        <template v-if="vcs.type === ExternalVersionControl_Type.GITHUB">
          <img class="h-6 w-auto" src="../assets/github-logo.svg" />
        </template>
        <template v-if="vcs.type === ExternalVersionControl_Type.BITBUCKET">
          <img class="h-6 w-auto" src="../assets/bitbucket-logo.svg" />
        </template>
        <span>{{ vcs.title }}</span>
      </button>
    </template>
  </div>
  <div class="mt-2 textinfolabel">
    <template v-if="canManageVCSProvider">
      <i18n-t keypath="repository.choose-git-provider-visit-workspace">
        <template #workspace>
          <router-link class="normal-link" to="/setting/gitops"
            >{{ $t("common.workspace") }} -
            {{ $t("common.gitops") }}</router-link
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
import { OAuthWindowEventPayload, openWindowForOAuth } from "../types";
import { hasWorkspacePermissionV1 } from "../utils";
import { pushNotification, useCurrentUserV1, useVCSV1Store } from "@/store";
import {
  ExternalVersionControl,
  ExternalVersionControl_Type,
} from "@/types/proto/v1/externalvs_service";

interface LocalState {
  selectedVCS?: ExternalVersionControl;
}

const emit = defineEmits<{
  (event: "next"): void;
  (event: "set-vcs", payload: ExternalVersionControl): void;
  (event: "set-code", payload: string): void;
}>();

const vcsV1Store = useVCSV1Store();
const state = reactive<LocalState>({});

const currentUserV1 = useCurrentUserV1();

const prepareVCSList = () => {
  vcsV1Store.fetchVCSList();
};

watchEffect(prepareVCSList);

onMounted(() => {
  window.addEventListener("bb.oauth.link-vcs-repository", eventListener, false);
});
onUnmounted(() => {
  window.removeEventListener("bb.oauth.link-vcs-repository", eventListener);
});

const vcsList = computed(() => {
  return vcsV1Store.getVCSList();
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

const canManageVCSProvider = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-vcs-provider",
    currentUserV1.value.userRole
  );
});

const selectVCS = (vcs: ExternalVersionControl) => {
  state.selectedVCS = vcs;
  emit("set-vcs", vcs);

  let authorizeUrl = `${vcs.url}/oauth/authorize`;
  if (vcs.type === ExternalVersionControl_Type.GITHUB) {
    authorizeUrl = `${vcs.url}/login/oauth/authorize`;
  } else if (vcs.type === ExternalVersionControl_Type.BITBUCKET) {
    authorizeUrl = `https://bitbucket.org/site/oauth2/authorize`;
  }
  openWindowForOAuth(
    authorizeUrl,
    vcs.applicationId,
    "bb.oauth.link-vcs-repository",
    vcs.type
  );
};
</script>
