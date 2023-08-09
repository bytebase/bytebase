<template>
  <div class="p-4">
    <div>{{ state.message }}</div>
    <div v-if="state.hasError" class="mt-2">
      <button
        v-if="state.oAuthState?.popup"
        type="button"
        class="btn-normal"
        @click="window.close()"
      >
        Close
      </button>
      <router-link v-else to="/auth" class="btn-normal"
        >Back to Sign in</router-link
      >
    </div>
  </div>
</template>

<script lang="ts" setup>
import { parse } from "qs";
import { onMounted, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useAuthStore } from "@/store";
import { OAuthState, OAuthWindowEventPayload } from "../types";

interface LocalState {
  message: string;
  hasError: boolean;
  oAuthState: OAuthState | undefined;
  payload: OAuthWindowEventPayload;
}

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();

const state = reactive<LocalState>({
  message: "",
  hasError: false,
  oAuthState: undefined,
  payload: {
    error: "",
    code: "",
  },
});

onMounted(() => {
  const queryState = parse((route.query.state as string) || "");

  if (queryState.event) {
    state.oAuthState = {
      event: queryState.event as string,
      popup: queryState.popup === "true" ? true : false,
      redirect: (queryState.redirect as string) || "",
    };
    state.hasError = false;
    state.message = "Successfully authorized. Redirecting back to Bytebase...";
    state.payload.code = (route.query.code as string) || "";
  } else {
    state.hasError = true;
    state.message =
      "Failed to authorize. Invalid state passed to the oauth callback.";
    state.oAuthState = undefined;
  }

  triggerAuthCallback();
});

const triggerAuthCallback = async () => {
  const { oAuthState, hasError } = state;
  if (hasError || !oAuthState) {
    window.opener.dispatchEvent(
      new CustomEvent("bb.oauth.unknown", {
        detail: state.payload,
      })
    );
    return;
  }

  const eventName = oAuthState.event;
  const eventType = eventName.slice(0, eventName.lastIndexOf("."));
  if (eventName.startsWith("bb.oauth.signin")) {
    if (oAuthState.popup) {
      window.opener.dispatchEvent(
        new CustomEvent(eventName, {
          detail: state.payload,
        })
      );
      window.close();
    } else {
      const mfaTempToken = await authStore.login({
        idpName: eventName.split(".").pop()!,
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
            redirect: oAuthState.redirect || "",
          },
        });
      } else {
        router.push(oAuthState.redirect || "/");
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
};
</script>
