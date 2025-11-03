<template>
  <div class="flex items-center justify-center min-h-screen">
    <div class="text-center">
      <BBSpin class="w-12 h-12 mx-auto" />
      <h2 class="mt-4 text-xl font-semibold text-main">{{ message }}</h2>
      <p v-if="errorMessage" class="mt-2 text-red-600">{{ errorMessage }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useIdentityProviderStore } from "@/store";
import { openWindowForSSO } from "@/utils/sso";

const route = useRoute();
const router = useRouter();
const idpStore = useIdentityProviderStore();

const message = ref("Initializing secure login...");
const errorMessage = ref("");

onMounted(async () => {
  try {
    const idpSlug = route.query.idp;
    const relayState = route.query.relay_state;

    // Validate types - query params can be string, string[], or undefined
    const idpSlugStr = typeof idpSlug === "string" ? idpSlug : undefined;
    const relayStateStr =
      typeof relayState === "string" ? relayState : undefined;

    // Security: Validate relay_state
    if (relayStateStr && !isValidRelayState(relayStateStr)) {
      throw new Error("Invalid relay state parameter");
    }

    // Get IdP configuration
    const idps = await idpStore.fetchIdentityProviderList();
    const idp = idpSlugStr
      ? await idpStore.getOrFetchIdentityProviderByName(`idps/${idpSlugStr}`)
      : idps[0];

    if (!idp) {
      throw new Error("Identity provider not found");
    }

    message.value = `Logging in with ${idp.title}...`;

    // Trigger OAuth redirect (same as SP-initiated flow)
    // Pass relay_state if provided (for deep linking), otherwise undefined
    // The auth store will handle undefined by using default redirect logic
    await openWindowForSSO(idp, false, relayStateStr);
  } catch (error) {
    console.error("IdP-initiated SSO error:", error);
    errorMessage.value =
      error instanceof Error ? error.message : "Failed to initiate SSO";
    message.value = "Authentication failed";

    setTimeout(() => {
      router.push({
        name: AUTH_SIGNIN_MODULE,
        query: { error: "sso_init_failed" },
      });
    }, 3000);
  }
});

function isValidRelayState(state: string): boolean {
  // Only allow relative URLs starting with /
  // Reject protocol-relative URLs (//) to prevent open redirect attacks
  return state.startsWith("/") && !state.startsWith("//");
}
</script>
