<template>
  <div class="p-4">
    <div>{{ state.message }}</div>
    <button
      v-if="state.hasError"
      type="button"
      class="btn-normal mt-2"
      @click.prevent="window.close()"
    >
      Close
    </button>
  </div>
</template>

<script lang="ts">
import { reactive } from "vue";
import { useRouter } from "vue-router";
import {
  getOAuthEventName,
  OAuthStateSessionKey,
  OAuthWindowEventPayload,
  OAuthType,
} from "../types";

interface LocalState {
  message: string;
  hasError: boolean;
}

export default {
  name: "OAuthCallback",
  setup() {
    const router = useRouter();

    const state = reactive<LocalState>({
      message: "",
      hasError: false,
    });

    const payload: OAuthWindowEventPayload = {
      error: "",
      code: "",
    };

    const expectedState = sessionStorage.getItem(OAuthStateSessionKey);
    let eventType = "unknown";
    if (
      !expectedState ||
      expectedState != router.currentRoute.value.query.state
    ) {
      state.hasError = true;
      state.message =
        "Failed to authorize. Invalid state passed to the oauth callback.";
      payload.error = state.message;
    } else {
      state.message =
        "Successfully authorized. Redirecting back to the application...";
      payload.code = router.currentRoute.value.query.code as string;
      eventType = expectedState.split("-")[0];
    }

    window.opener.dispatchEvent(
      new CustomEvent(getOAuthEventName(eventType as OAuthType), {
        detail: payload,
      })
    );
    window.close();

    return {
      state,
    };
  },
};
</script>
