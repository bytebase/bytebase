<template>
  <div class="p-4">
    <div v-if="state.hasError" class="mt-2">
      <div>{{ state.message }}</div>
      <NButton v-if="state.oAuthState?.popup" @click="window.close()">
        Close
      </NButton>
      <router-link v-else :to="{ name: AUTH_SIGNIN_MODULE }" class="btn-normal">
        Back to Sign in
      </router-link>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton } from "naive-ui";
import { onMounted, reactive } from "vue";
import { useRoute } from "vue-router";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useAuthStore } from "@/store";
import { LoginRequestSchema } from "@/types/proto-es/v1/auth_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";
import { retrieveOAuthState, clearOAuthState } from "@/utils/sso";
import type { OAuthState, OAuthWindowEventPayload } from "../types";

interface LocalState {
  message: string;
  hasError: boolean;
  oAuthState: OAuthState | undefined;
  payload: OAuthWindowEventPayload;
}

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
  // State parameter is now an opaque token (RFC 6749 compliant)
  const stateToken = route.query.state as string;

  // Validate state token exists
  if (
    !stateToken ||
    typeof stateToken !== "string" ||
    stateToken.length === 0
  ) {
    state.hasError = true;
    state.message =
      "Authentication failed: Invalid callback state. Please try again.";
    state.oAuthState = undefined;
    triggerAuthCallback();
    return;
  }

  // Retrieve and validate stored state from localStorage using opaque token
  const storedState = retrieveOAuthState(stateToken);

  if (!storedState) {
    state.hasError = true;
    state.message =
      "Authentication failed: Session expired or invalid. Please try again.";
    state.oAuthState = undefined;
    triggerAuthCallback();
    return;
  }

  // Validate token matches (constant-time comparison via retrieval)
  if (storedState.token !== stateToken) {
    state.hasError = true;
    state.message =
      "Authentication failed: Security validation failed. Please try again.";
    state.oAuthState = undefined;
    clearOAuthState(stateToken);
    triggerAuthCallback();
    return;
  }

  // State validation successful
  state.oAuthState = storedState;
  state.hasError = false;
  state.message = "Successfully authorized. Redirecting back to Bytebase...";
  state.payload.code = (route.query.code as string) || "";

  // Clear state from localStorage (single-use token)
  clearOAuthState(storedState.token);

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
  if (eventName.startsWith("bb.oauth.signin")) {
    if (oAuthState.popup) {
      window.opener.dispatchEvent(
        new CustomEvent(eventName, {
          detail: state.payload,
        })
      );
      window.close();
    } else {
      // Determine context type based on the stored IdP type
      // OIDC providers use oidcContext, OAuth2 providers use oauth2Context
      const isOidc = oAuthState.idpType === IdentityProviderType.OIDC;

      await authStore.login(
        create(LoginRequestSchema, {
          idpName: eventName.split(".").pop()!,
          idpContext: {
            context: isOidc
              ? {
                  case: "oidcContext",
                  value: {
                    code: state.payload.code,
                  },
                }
              : {
                  case: "oauth2Context",
                  value: {
                    code: state.payload.code,
                  },
                },
          },
          web: true,
        }),
        oAuthState.redirect
      );
    }
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
