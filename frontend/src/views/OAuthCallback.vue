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
import { parse } from "qs";
import { onMounted, reactive } from "vue";
import { useRoute } from "vue-router";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useAuthStore } from "@/store";
import { LoginRequestSchema } from "@/types/proto-es/v1/auth_service_pb";
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
  if (eventName.startsWith("bb.oauth.signin")) {
    if (oAuthState.popup) {
      window.opener.dispatchEvent(
        new CustomEvent(eventName, {
          detail: state.payload,
        })
      );
      window.close();
    } else {
      await authStore.login(
        create(LoginRequestSchema, {
          idpName: eventName.split(".").pop()!,
          idpContext: {
            context: {
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
