<template>
  <div class="mx-auto w-full max-w-sm">
    <BytebaseLogo class="mx-auto mb-8" />

    <NCard v-if="state.loading">
      <div class="flex justify-center py-8">
        <BBSpin />
      </div>
    </NCard>

    <NCard v-else-if="state.error">
      <div class="text-center py-4">
        <p class="text-red-600 mb-4">{{ state.error }}</p>
        <NButton @click="goBack">
          {{ $t("common.go-back") }}
        </NButton>
      </div>
    </NCard>

    <NCard v-else>
      <div class="space-y-6">
        <div class="text-center">
          <h1 class="text-xl font-semibold text-gray-900 mb-2">
            {{ $t("oauth2.consent.title") }}
          </h1>
          <p class="text-gray-600">
            {{
              $t("oauth2.consent.description", {
                clientName: state.clientName,
              })
            }}
          </p>
        </div>

        <div class="bg-gray-50 rounded-lg p-4">
          <p class="text-sm text-gray-500 mb-2">
            {{ $t("oauth2.consent.permissions") }}
          </p>
          <ul class="text-sm text-gray-700 space-y-1">
            <li class="flex items-center gap-2">
              <CheckIcon class="w-4 h-4 text-green-500" />
              {{ $t("oauth2.consent.permission-access") }}
            </li>
          </ul>
        </div>

        <form method="POST" :action="authorizeUrl">
          <input type="hidden" name="client_id" :value="clientId" />
          <input type="hidden" name="redirect_uri" :value="redirectUri" />
          <input type="hidden" name="state" :value="oauthState" />
          <input type="hidden" name="code_challenge" :value="codeChallenge" />
          <input
            type="hidden"
            name="code_challenge_method"
            :value="codeChallengeMethod"
          />

          <div class="flex gap-x-2">
            <NButton
              class="flex-1"
              size="large"
              :disabled="state.submitting"
              @click="deny"
            >
              {{ $t("common.deny") }}
            </NButton>
            <NButton
              type="primary"
              class="flex-1"
              size="large"
              :loading="state.submitting"
              attr-type="submit"
              name="action"
              value="allow"
            >
              {{ $t("common.allow") }}
            </NButton>
          </div>
        </form>
      </div>
    </NCard>
  </div>
</template>

<script lang="ts" setup>
import { CheckIcon } from "lucide-vue-next";
import { NButton, NCard } from "naive-ui";
import { onMounted, reactive } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import { useAuthStore } from "@/store";

interface LocalState {
  loading: boolean;
  submitting: boolean;
  error: string;
  clientName: string;
}

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const state = reactive<LocalState>({
  loading: true,
  submitting: false,
  error: "",
  clientName: "",
});

// Extract query parameters
const clientId = (route.query.client_id as string) || "";
const redirectUri = (route.query.redirect_uri as string) || "";
const oauthState = (route.query.state as string) || "";
const codeChallenge = (route.query.code_challenge as string) || "";
const codeChallengeMethod = (route.query.code_challenge_method as string) || "";

const authorizeUrl = "/api/oauth2/authorize";

onMounted(async () => {
  // Check if user is logged in
  if (!authStore.isLoggedIn) {
    // Redirect to login with return URL
    const returnUrl = route.fullPath;
    router.replace({
      name: AUTH_SIGNIN_MODULE,
      query: { redirect: returnUrl },
    });
    return;
  }

  // Validate required parameters
  if (!clientId || !redirectUri || !codeChallenge || !codeChallengeMethod) {
    state.error = "Missing required OAuth2 parameters";
    state.loading = false;
    return;
  }

  // Fetch client info
  try {
    const response = await fetch(
      `/api/oauth2/clients/${encodeURIComponent(clientId)}`
    );
    if (!response.ok) {
      const data = await response.json();
      state.error = data.error_description || "Client not found";
      state.loading = false;
      return;
    }
    const data = await response.json();
    state.clientName = data.client_name || clientId;
  } catch {
    state.error = "Failed to load client information";
  }

  state.loading = false;
});

const deny = () => {
  state.submitting = true;
  // Submit form with deny action
  const form = document.createElement("form");
  form.method = "POST";
  form.action = authorizeUrl;

  const fields = [
    { name: "client_id", value: clientId },
    { name: "redirect_uri", value: redirectUri },
    { name: "state", value: oauthState },
    { name: "code_challenge", value: codeChallenge },
    { name: "code_challenge_method", value: codeChallengeMethod },
    { name: "action", value: "deny" },
  ];

  for (const field of fields) {
    const input = document.createElement("input");
    input.type = "hidden";
    input.name = field.name;
    input.value = field.value;
    form.appendChild(input);
  }

  document.body.appendChild(form);
  form.submit();
};

const goBack = () => {
  router.back();
};
</script>
