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
import { nextTick, onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { OAuthWindowEventPayload, OAuthStateSessionKey } from "../types";

interface LocalState {
  message: string;
  hasError: boolean;
  payload: OAuthWindowEventPayload;
}

const router = useRouter();

const state = reactive<LocalState>({
  message: "",
  hasError: false,
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
  closeWindow();
});

const closeWindow = () => {
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
      window.opener.dispatchEvent(
        new CustomEvent(eventName, {
          detail: state.payload,
        })
      );
    } else if (
      eventName.startsWith("bb.oauth.register-vcs") ||
      eventName.startsWith("bb.oauth.link-vcs-repository")
    ) {
      window.opener.dispatchEvent(
        new CustomEvent(eventType, {
          detail: state.payload,
        })
      );
    } else {
      window.opener.dispatchEvent(
        new CustomEvent("bb.oauth.unknown", {
          detail: state.payload,
        })
      );
    }
  }

  nextTick(() => {
    window.close();
  });
};
</script>
