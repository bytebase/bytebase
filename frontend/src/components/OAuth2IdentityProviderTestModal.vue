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
      <template v-if="state.code">
        <hr class="my-2 mb-4" />
        <div class="w-full flex flex-col justify-start items-start">
          <p>✅ authorize code</p>
          <pre
            class="my-2 w-full flex flex-row justify-start items-start bg-gray-50 rounded p-4"
          >
            <code class="w-full whitespace-pre-wrap break-all text-sm">{{ state.code }}</code>
          </pre>
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <p v-if="state.accessToken || !isTesting">
            {{ state.accessToken ? "✅" : !isTesting ? "❌" : "" }} access token
          </p>
          <pre
            v-if="state.accessToken"
            class="my-2 w-full flex flex-row justify-start items-start bg-gray-50 rounded p-4"
          >
            <code class="w-full whitespace-pre-wrap break-all text-sm">{{ state.accessToken }}</code>
          </pre>
        </div>
        <div class="w-full flex flex-col justify-start items-start">
          <p v-if="state.userInfo || !isTesting">
            {{ state.userInfo ? "✅" : !isTesting ? "❌" : "" }} converted user
            information
          </p>
          <pre
            v-if="state.userInfo"
            class="my-2 w-full flex flex-row justify-start items-start bg-gray-50 rounded p-4"
          >
            <code class="w-full whitespace-pre-wrap break-all text-sm">{{ state.userInfo }}</code>
          </pre>
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
        <p v-if="isTesting">{{ $t("common.loading") }}...</p>
      </template>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, PropType, reactive } from "vue";
import { OAuthWindowEventPayload } from "@/types";
import { IdentityProvider } from "@/types/proto/v1/idp_service";
import { openWindowForSSO } from "@/utils";
import { IdentityProviderUserInfo } from "@/types/proto/store/idp";
import { identityProviderClient } from "@/grpcweb";
import { isUndefined } from "lodash-es";

interface LocalState {
  code?: string;
  accessToken?: string;
  userInfo?: IdentityProviderUserInfo;
  errorMessage?: string;
}

const props = defineProps({
  identityProvider: {
    required: true,
    type: Object as PropType<IdentityProvider>,
  },
});

const emit = defineEmits(["cancel"]);
const state = reactive<LocalState>({});

const identityProvider = computed(() => {
  return props.identityProvider;
});

const isTesting = computed(() => {
  if (isUndefined(state.code)) {
    return false;
  }
  if (
    !isUndefined(state.code) &&
    !isUndefined(state.accessToken) &&
    !isUndefined(state.userInfo)
  ) {
    return false;
  }
  if (!isUndefined(state.errorMessage)) {
    return false;
  }
  return true;
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
  state.code = code;

  const { message, oauth2Result } =
    await identityProviderClient().testIdentityProvider({
      identityProvider: identityProvider.value,
      oauth2Context: {
        code: code,
      },
    });
  state.errorMessage = message;
  state.accessToken = oauth2Result?.accessToken;
  state.userInfo = oauth2Result?.userInfo;
};

const handleGetCodeButtonClick = () => {
  openWindowForSSO(identityProvider.value);
};
</script>
