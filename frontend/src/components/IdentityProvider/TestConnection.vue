<template>
  <NButton
    @click="testConnection"
    :loading="isTestingInProgress"
    :disabled="isTestingInProgress"
    v-bind="$attrs"
  >
    {{ $t("identity-provider.test-connection") }}
  </NButton>

  <!-- Claims Dialog -->
  <NModal
    v-model:show="showClaimsDialog"
    preset="dialog"
    class="!w-128"
    :show-icon="false"
  >
    <template #header>
      <div class="flex items-center gap-x-2">
        <heroicons:check-circle class="w-6 h-6 text-success" />
        <span>{{ $t("identity-provider.test-connection-success") }}</span>
      </div>
    </template>
    <div class="space-y-4">
      <p class="text-sm text-control-light">
        {{ $t("identity-provider.claims-description") }}
      </p>
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
        <div class="space-y-2">
          <div
            v-if="Object.keys(claims).length === 0"
            class="text-sm text-control-light italic"
          >
            {{ $t("identity-provider.no-claims") }}
          </div>
          <div v-else>
            <div
              v-for="[key, value] in Object.entries(claims)"
              :key="key"
              class="grid grid-cols-3 gap-2 py-1 border-b border-gray-200 dark:border-gray-600 last:border-b-0"
            >
              <div
                class="text-sm font-medium text-control truncate"
                :title="key"
              >
                {{ key }}
              </div>
              <div
                class="col-span-2 text-sm text-main break-all"
                :title="value"
              >
                {{ value }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </NModal>
</template>

<script lang="ts" setup>
import { NButton, NModal } from "naive-ui";
import { ref, onUnmounted, watch } from "vue";
import { identityProviderClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import type { OAuthWindowEventPayload } from "@/types";
import {
  IdentityProviderType,
  type IdentityProvider,
} from "@/types/proto/v1/idp_service";
import { openWindowForSSO } from "@/utils";

const props = defineProps<{
  idp: IdentityProvider;
  isCreating?: boolean;
}>();

// Reactive state for the claims dialog
const showClaimsDialog = ref(false);
const claims = ref<Record<string, string>>({});

// Track current event listener to prevent duplicates
const currentEventName = ref<string>("");

// Track if test is in progress to prevent multiple calls
const isTestingInProgress = ref(false);

const loginWithIdentityProviderEventListener = async (event: Event) => {
  // Prevent multiple concurrent API calls
  if (isTestingInProgress.value) {
    return;
  }

  const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
  if (payload.error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: payload.error,
    });
    return;
  }

  const code = payload.code;
  try {
    isTestingInProgress.value = true;
    const response = await identityProviderClient.testIdentityProvider({
      identityProvider: props.idp,
      oauth2Context: {
        code: code,
      },
    });

    // Show claims in dialog
    claims.value = response.claims || {};
    showClaimsDialog.value = true;
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: (error as any).details,
    });
  } finally {
    isTestingInProgress.value = false;
  }
};

const testConnection = async () => {
  // Prevent multiple test connections from running simultaneously
  if (isTestingInProgress.value) {
    return;
  }

  const { idp, isCreating } = props;
  if (
    idp.type === IdentityProviderType.OAUTH2 ||
    idp.type === IdentityProviderType.OIDC
  ) {
    let idpForTesting = idp;
    // For OIDC, we need to obtain the auth endpoint from the issuer in backend.
    if (isCreating && idp.type === IdentityProviderType.OIDC) {
      idpForTesting = await identityProviderClient.createIdentityProvider({
        identityProviderId: idp.name,
        identityProvider: idp,
        validateOnly: true,
      });
    }

    // Ensure event listener is set up for the correct IDP name
    const eventName = `bb.oauth.signin.${idpForTesting.name}`;

    // Remove any existing listener first.
    if (currentEventName.value) {
      window.removeEventListener(
        currentEventName.value,
        loginWithIdentityProviderEventListener,
        false
      );
    }
    // Add a new event listener.
    window.addEventListener(
      eventName,
      loginWithIdentityProviderEventListener,
      false
    );
    currentEventName.value = eventName;

    try {
      await openWindowForSSO(idpForTesting);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `Request error occurred`,
        description: (error as any).message,
      });
    }
  } else if (idp.type === IdentityProviderType.LDAP) {
    try {
      isTestingInProgress.value = true;
      const response = await identityProviderClient.testIdentityProvider({
        identityProvider: idp,
      });

      // Show claims in dialog (LDAP will have empty claims)
      claims.value = response.claims || {};
      showClaimsDialog.value = true;
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `Request error occurred`,
        description: (error as any).details,
      });
    } finally {
      isTestingInProgress.value = false;
    }
  }
};

// Reset testing state when component unmounts
onUnmounted(() => {
  if (currentEventName.value) {
    window.removeEventListener(
      currentEventName.value,
      loginWithIdentityProviderEventListener,
      false
    );
    currentEventName.value = "";
  }
  isTestingInProgress.value = false;
});

// Watch for dialog close to reset testing state if needed
watch(showClaimsDialog, (newValue) => {
  if (!newValue) {
    // Reset testing state when dialog is closed
    isTestingInProgress.value = false;
  }
});
</script>
