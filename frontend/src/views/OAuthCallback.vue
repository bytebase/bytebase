<template>
  <div class="p-4">
    <div>{{ state.message }}</div>
    <button
      v-if="state.hasError"
      type="button"
      class="btn-normal mt-2"
      @click="window.close()"
    >
      Close
    </button>
  </div>
</template>

<script lang="ts" setup>
import { useAuthStore } from "@/store";
import { SSOConfigSessionKey } from "@/utils/sso";
import { onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { OAuthWindowEventPayload, OAuthStateSessionKey } from "../types";

interface LocalState {
  message: string;
  hasError: boolean;
  openAsPopup: boolean;
  payload: OAuthWindowEventPayload;
}

const router = useRouter();
const authStore = useAuthStore();

const state = reactive<LocalState>({
  message: "",
  hasError: false,
  openAsPopup: true,
  payload: {
    error: "",
    code: "",
  },
});

onMounted(() => {
  const sessionState = sessionStorage.getItem(OAuthStateSessionKey);
  if (!sessionState || sessionState !== router.currentRoute.value.query.state) {
    state.hasError = true;
    state.message =
      "Failed to authorize. Invalid state passed to the oauth callback.";
    state.payload.error = state.message;
  } else {
    state.message =
      "Successfully authorized. Redirecting back to the application...";
    state.payload.code = router.currentRoute.value.query.code as string;
  }
  triggerAuthCallback();
});

const triggerAuthCallback = async () => {
  if (state.hasError) {
    window.opener.dispatchEvent(
      new CustomEvent("bb.oauth.unknown", {
        detail: state.payload,
      })
    );
  } else {
    const eventName = sessionStorage.getItem(OAuthStateSessionKey) || "";
    const eventType = eventName.slice(0, eventName.lastIndexOf("."));
    if (eventName.startsWith("bb.oauth.signin")) {
      const ssoConfig = JSON.parse(
        sessionStorage.getItem(SSOConfigSessionKey) || "{}"
      );
      if (ssoConfig.openAsPopup) {
        window.opener.dispatchEvent(
          new CustomEvent(eventName, {
            detail: state.payload,
          })
        );
        window.close();
      } else {
        const mfaTempToken = await authStore.login({
          idpName: ssoConfig.identityProviderName,
          idpContext: {
            oauth2Context: {
              code: state.payload.code,
            },
          },
          web: true,
        });
        if (mfaTempToken) {
          router.push({
            name: "auth.mfa",
            query: {
              mfaTempToken,
              redirect: "",
            },
          });
        } else {
          router.push("/");
        }
      }
    } else if (
      eventName.startsWith("bb.oauth.register-vcs") ||
      eventName.startsWith("bb.oauth.link-vcs-repository")
    ) {
      window.opener.dispatchEvent(
        new CustomEvent(eventType, {
          detail: state.payload,
        })
      );
      window.close();
    } else {
      window.opener.dispatchEvent(
        new CustomEvent("bb.oauth.unknown", {
          detail: state.payload,
        })
      );
      window.close();
    }
  }
};
</script>
