<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img class="h-12 w-auto" src="@/assets/logo-full.svg" alt="Bytebase" />
      <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
        {{ $t("auth.password-forget.title") }}
      </h2>
    </div>

    <div class="mt-8">
      <div class="mt-6 flex flex-col gap-y-4">
        <BBAttention
          v-if="!passwordResetEnabled"
          type="warning"
          :title="$t('auth.password-forget.selfhost')"
        />
        <template v-else>
          <div>
            <label
              for="forgot-email"
              class="block text-sm font-medium leading-5 text-control"
            >
              {{ $t("common.email") }}
            </label>
            <div class="mt-1">
              <BBTextField
                v-model:value="email"
                required
                :input-props="{
                  id: 'forgot-email',
                  autocomplete: 'email',
                  type: 'email',
                }"
                placeholder="jim@example.com"
                @keyup.enter="onSubmit"
              />
            </div>
          </div>
          <NButton
            type="primary"
            size="large"
            style="width: 100%"
            :disabled="!isValidEmail(email) || isLoading"
            :loading="isLoading"
            @click="onSubmit"
          >
            {{ $t("auth.password-forget.send-reset-code") }}
          </NButton>
        </template>
      </div>
    </div>

    <div class="mt-6 relative">
      <div class="absolute inset-0 flex items-center" aria-hidden="true">
        <div class="w-full border-t border-control-border"></div>
      </div>
      <div class="relative flex justify-center text-sm">
        <router-link
          :to="{ name: AUTH_SIGNIN_MODULE, query: $route.query }"
          class="accent-link bg-white px-2"
        >
          {{ $t("auth.password-forget.return-to-sign-in") }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBAttention, BBTextField } from "@/bbkit";
import { authServiceClientConnect } from "@/connect";
import { AUTH_PASSWORD_RESET_MODULE, AUTH_SIGNIN_MODULE } from "@/router/auth";
import { pushNotification, useActuatorV1Store } from "@/store";
import { isValidEmail, resolveWorkspaceName } from "@/utils";

const actuatorStore = useActuatorV1Store();
const router = useRouter();
const route = useRoute();
const { t } = useI18n();

const passwordResetEnabled = computed(
  () => actuatorStore.serverInfo?.restriction?.passwordResetEnabled ?? false
);

onMounted(() => {
  if (actuatorStore.serverInfo?.restriction?.disallowPasswordSignin) {
    router.replace({ name: AUTH_SIGNIN_MODULE, query: route.query });
  }
});

const email = ref("");
const isLoading = ref(false);

const onSubmit = async () => {
  if (!isValidEmail(email.value) || isLoading.value) return;
  isLoading.value = true;
  try {
    await authServiceClientConnect.requestPasswordReset({
      email: email.value,
      workspace: resolveWorkspaceName(),
    });
    router.push({
      name: AUTH_PASSWORD_RESET_MODULE,
      query: { ...route.query, email: email.value },
    });
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("auth.password-forget.failed-to-send-code"),
    });
  } finally {
    isLoading.value = false;
  }
};
</script>
