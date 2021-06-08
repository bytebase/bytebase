<template>
  <h1>OAuth callback</h1>
</template>

<script lang="ts">
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import {
  OAuthState,
  oauthStateKey,
  redirectURL,
  VCSTokenCreate,
} from "../types";
export default {
  name: "OAuthCallback",
  components: {},
  props: {},
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const item = router.currentRoute.value.query.state
      ? sessionStorage.getItem(
          oauthStateKey(router.currentRoute.value.query.state as string)
        )
      : undefined;
    if (!item) {
      store.dispatch("notification/pushNotification", {
        module: "bytebase",
        style: "CRITICAL",
        title: `Invalid state passed to the oauth callback`,
      });
      router.replace({
        name: "setting.workspace.version-control",
      });
    } else {
      const oauthState = JSON.parse(item) as OAuthState;
      if (
        !oauthState ||
        oauthState.resourceType != "VCS" ||
        !oauthState.resourceId
      ) {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "CRITICAL",
          title: `Invalid oauth session state`,
        });
        router.replace({
          name: "setting.workspace.version-control",
        });
      } else {
        const tokenCreate: VCSTokenCreate = {
          code: router.currentRoute.value.query.code as string,
          redirectURL: redirectURL(),
        };
        store.dispatch("vcs/createVCSToken", {
          vcsId: oauthState.resourceId,
          tokenCreate,
        });
      }
    }
  },
};
</script>
