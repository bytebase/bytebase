<template>
  <BBModal
    :title="$t('identity-provider.test-modal.title')"
    @close="emit('cancel')"
  >
    <div class="w-96 h-auto">
      <div class="w-full flex flex-col justify-start items-start">
        <p>{{ $t("identity-provider.test-modal.start-test") }}</p>
        <button class="btn-normal my-2" @click="handleGetCodeButtonClick">
          {{
            $t("auth.sign-in.sign-in-with-idp", { idp: identityProvider.title })
          }}
        </button>
      </div>
      <div
        v-if="state.errorMessage"
        class="w-full flex flex-col justify-start items-start"
      >
        <hr class="w-full my-4" />
        <p>{{ $t("common.error-message") }}</p>
        <pre
          class="my-2 w-full flex flex-row justify-start items-start bg-gray-50 rounded p-4"
        >
          <code class="w-full whitespace-pre-wrap break-all text-sm">{{ state.errorMessage }}</code>
        </pre>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { ClientError } from "nice-grpc-common";
import { computed, onMounted, onUnmounted, PropType, reactive } from "vue";
import { OAuthWindowEventPayload } from "@/types";
import { IdentityProvider } from "@/types/proto/v1/idp_service";
import { openWindowForSSO } from "@/utils";
import { identityProviderClient } from "@/grpcweb";
import { useNotificationStore } from "@/store";

interface LocalState {
  errorMessage?: string;
}

const props = defineProps({
  identityProvider: {
    required: true,
    type: Object as PropType<IdentityProvider>,
  },
});
const emit = defineEmits(["cancel"]);

const notificationStore = useNotificationStore();
const state = reactive<LocalState>({});

const identityProvider = computed(() => {
  return props.identityProvider;
});

onMounted(() => {
  window.addEventListener(
    `bb.oauth.signin.${identityProvider.value.name}`,
    loginWithIdentityProviderEventListener,
    false
  );
});

onUnmounted(() => {
  window.removeEventListener(
    `bb.oauth.signin.${identityProvider.value.name}`,
    loginWithIdentityProviderEventListener,
    false
  );
});

const loginWithIdentityProviderEventListener = async (event: Event) => {
  const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
  if (payload.error) {
    return;
  }
  const code = payload.code;

  try {
    await identityProviderClient().testIdentityProvider({
      identityProvider: identityProvider.value,
      oauth2Context: {
        code: code,
      },
    });
  } catch (error) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: (error as ClientError).details,
    });
    return;
  }

  notificationStore.pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Test connection succeed",
  });
};

const handleGetCodeButtonClick = () => {
  openWindowForSSO(identityProvider.value);
};
</script>
