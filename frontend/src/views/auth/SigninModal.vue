<template>
  <BBModal
    :show="shouldShow"
    :trap-focus="true"
    :show-close="false"
    :mask-closable="false"
    :header-class="'!hidden'"
  >
    <div class="flex items-center w-auto md:min-w-96 max-w-full h-auto md:py-4">
      <div class="flex flex-col justify-center items-center flex-1 space-y-2">
        <Signin :allow-signup="false">
          <template #footer>
            <NButton quaternary size="small" @click="logout">
              {{ $t("common.logout") }}
            </NButton>
          </template>
        </Signin>
      </div>
    </div>
  </BBModal>
</template>

<script lang="tsx" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useRoute } from "vue-router";
import { BBModal } from "@/bbkit";
import {
  AUTH_MFA_MODULE,
  AUTH_OAUTH_CALLBACK_MODULE,
  AUTH_OIDC_CALLBACK_MODULE,
  AUTH_PASSWORD_FORGOT_MODULE,
  AUTH_PASSWORD_RESET_MODULE,
  AUTH_SIGNIN_ADMIN_MODULE,
  AUTH_SIGNIN_MODULE,
  AUTH_SIGNUP_MODULE,
} from "@/router/auth";
import { useAuthStore } from "@/store";
import Signin from "@/views/auth/Signin.vue";

const route = useRoute();
const authStore = useAuthStore();

const logout = () => {
  authStore.unauthenticatedOccurred = false;
  authStore.logout();
};

const shouldShow = computed(() => {
  // Do not show the modal when the user is in auth related pages.
  if (
    route.name &&
    [
      AUTH_SIGNIN_MODULE,
      AUTH_SIGNIN_ADMIN_MODULE,
      AUTH_SIGNUP_MODULE,
      AUTH_MFA_MODULE,
      AUTH_PASSWORD_RESET_MODULE,
      AUTH_PASSWORD_FORGOT_MODULE,
      AUTH_OAUTH_CALLBACK_MODULE,
      AUTH_OIDC_CALLBACK_MODULE,
    ].includes(route.name.toString())
  ) {
    return false;
  }

  return Boolean(authStore.unauthenticatedOccurred);
});
</script>
