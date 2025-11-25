<template>
  <div class="p-4">
    <div v-if="state.hasError" class="mt-2">
      <div>{{ state.message }}</div>
      <NButton v-if="state.oAuthState?.popup" @click="window.close()">
        {{ $t("common.close") }}
      </NButton>
      <router-link v-else :to="{ name: AUTH_SIGNIN_MODULE }" class="btn-normal">
        {{ $t("auth.back-to-signin") }}
      </router-link>
    </div>
    <div v-else class="mt-2">
      <div class="flex items-center gap-x-2">
        <NSpin size="small" />
        <span>{{ state.message || "Processing authentication..." }}</span>
      </div>
      <NButton
        v-if="state.oAuthState?.popup && state.showCloseButton"
        class="mt-4"
        @click="window.close()"
      >
        {{ $t("auth.close-window") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton, NSpin } from "naive-ui";
import { onMounted, reactive } from "vue";
import { useRoute } from "vue-router";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useAuthStore } from "@/store";
import { LoginRequestSchema } from "@/types/proto-es/v1/auth_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";
import { clearOAuthState, retrieveOAuthState } from "@/utils/sso";
import type { OAuthState, OAuthWindowEventPayload } from "../types";

interface LocalState {
  message: string;
  hasError: boolean;
  oAuthState: OAuthState | undefined;
  payload: OAuthWindowEventPayload;
  showCloseButton: boolean;
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
  showCloseButton: false,
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

  // Handle popup flow with robust error handling
  if (oAuthState?.popup) {
    try {
      // Check if opener window is available and not closed
      if (!window.opener || window.opener.closed) {
        state.hasError = true;
        state.message =
          "Authentication completed, but the parent window is no longer available. Please close this window and try again.";
        state.showCloseButton = true;
        return;
      }

      const eventName = hasError ? "bb.oauth.unknown" : oAuthState.event;

      // Dispatch event to parent window
      window.opener.dispatchEvent(
        new CustomEvent(eventName, {
          detail: state.payload,
        })
      );

      // Try to close the popup window
      try {
        window.close();

        // If window.close() doesn't work immediately (some browsers delay it),
        // show a close button after a short delay
        setTimeout(() => {
          if (!window.closed) {
            state.showCloseButton = true;
            state.message = hasError
              ? state.message
              : "Authentication completed. You can close this window.";
          }
        }, 500);
      } catch {
        // Browser blocked window.close()
        state.showCloseButton = true;
        state.message = hasError
          ? state.message
          : "Authentication completed. Please close this window.";
      }
    } catch (error) {
      console.error("Failed to communicate with opener window:", error);
      state.hasError = true;
      state.message =
        "Authentication completed, but failed to communicate with the parent window. Please close this window and try again.";
      state.showCloseButton = true;
    }
    return;
  }

  // Handle redirect flow (non-popup)
  if (hasError || !oAuthState) {
    // For redirect flow, errors are already displayed in the template
    return;
  }

  const eventName = oAuthState.event;
  if (eventName.startsWith("bb.oauth.signin")) {
    // Determine context type based on the stored IdP type
    // OIDC providers use oidcContext, OAuth2 providers use oauth2Context
    const isOidc = oAuthState.idpType === IdentityProviderType.OIDC;
    const idpName = eventName.split(".").pop();
    if (!idpName) {
      return;
    }

    await authStore.login({
      request: create(LoginRequestSchema, {
        idpName,
        idpContext: {
          context: {
            case: isOidc ? "oidcContext" : "oauth2Context",
            value: {
              code: state.payload.code,
            },
          },
        },
      }),
      redirect: true,
      redirectUrl: oAuthState.redirect,
    });
  }
};
</script>
