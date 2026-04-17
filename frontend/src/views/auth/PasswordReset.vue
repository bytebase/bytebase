<template>
  <div class="mx-auto w-full max-w-sm">
    <img class="h-12 w-auto" src="@/assets/logo-full.svg" alt="Bytebase" />
    <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
      {{ $t("auth.password-reset.title") }}
    </h2>
    <p class="textinfo mt-2">
      {{ $t("auth.password-reset.content") }}
    </p>

    <div class="mt-8 flex flex-col gap-y-6">
      <!-- Code-based mode: email + code fields -->
      <template v-if="codeMode">
        <div>
          <label class="block text-sm font-medium leading-5 text-control">
            {{ $t("common.email") }}
            <RequiredStar />
          </label>
          <BBTextField
            v-model:value="state.email"
            required
            :input-props="{
              autocomplete: 'email',
              type: 'email',
            }"
            class="mt-1"
            disabled
          />
        </div>
        <div>
          <label class="block text-sm font-medium leading-5 text-control">
            {{ $t("auth.password-reset.code-label") }}
            <RequiredStar />
          </label>
          <NInputOtp
            v-model:value="state.codeParts"
            :length="6"
            class="mt-1 w-full email-code-otp"
          />
          <div class="mt-2 flex items-center justify-end">
            <NButton
              text
              :disabled="resendCountdown > 0"
              @click="resendCode"
            >
              {{
                resendCountdown > 0
                  ? $t("auth.sign-in.resend-in", { seconds: resendCountdown })
                  : $t("auth.sign-in.resend-code")
              }}
            </NButton>
          </div>
        </div>
      </template>

      <UserPassword
        ref="userPasswordRef"
        v-model:password="state.password"
        v-model:password-confirm="state.passwordConfirm"
        :show-learn-more="false"
        :password-restriction="passwordRestrictionSetting"
      />

      <NButton
        type="primary"
        size="large"
        style="width: 100%"
        :disabled="!allowConfirm"
        @click="onConfirm"
      >
        {{ $t("common.confirm") }}
      </NButton>
    </div>

    <div v-if="codeMode" class="mt-6 relative">
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
import { create, create as createProto } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { NButton, NInputOtp } from "naive-ui";
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBTextField } from "@/bbkit";
import RequiredStar from "@/components/RequiredStar.vue";
import UserPassword from "@/components/User/Settings/UserPassword.vue";
import { authServiceClientConnect } from "@/connect";
import { AUTH_SIGNIN_MODULE } from "@/router/auth";
import {
  pushNotification,
  useActuatorV1Store,
  useAuthStore,
  useCurrentUserV1,
  useUserStore,
} from "@/store";
import {
  LoginRequestSchema,
  ResetPasswordRequestSchema,
} from "@/types/proto-es/v1/auth_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";
import { resolveWorkspaceName } from "@/utils";

interface LocalState {
  email: string;
  codeParts: string[];
  password: string;
  passwordConfirm: string;
}

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const me = useCurrentUserV1();
const userStore = useUserStore();
const authStore = useAuthStore();
const userPasswordRef = ref<InstanceType<typeof UserPassword>>();

const state = reactive<LocalState>({
  email: "",
  codeParts: [],
  password: "",
  passwordConfirm: "",
});

const codeMode = computed(() => !!route.query.email);

// Resend cooldown — starts at 60s (code was just sent from PasswordForgot).
const resendCountdown = ref(60);
let countdownTimer: ReturnType<typeof setInterval> | null = null;

const startCountdown = () => {
  resendCountdown.value = 60;
  if (countdownTimer) clearInterval(countdownTimer);
  countdownTimer = setInterval(() => {
    resendCountdown.value -= 1;
    if (resendCountdown.value <= 0 && countdownTimer) {
      clearInterval(countdownTimer);
      countdownTimer = null;
    }
  }, 1000);
};

const resendCode = async () => {
  if (resendCountdown.value > 0 || !state.email) return;
  try {
    await authServiceClientConnect.requestPasswordReset({
      email: state.email,
      workspace: resolveWorkspaceName(),
    });
    startCountdown();
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("auth.password-forget.failed-to-send-code"),
    });
  }
};

onUnmounted(() => {
  if (countdownTimer) clearInterval(countdownTimer);
});

const passwordRestrictionSetting = computed(
  () => useActuatorV1Store().serverInfo?.restriction?.passwordRestriction
);

const redirectQuery = computed(() => {
  const query = new URLSearchParams(window.location.search);
  return query.get("redirect") || "/";
});

onMounted(() => {
  if (codeMode.value) {
    // If password signin is disabled, resetting a password is pointless.
    if (useActuatorV1Store().serverInfo?.restriction?.disallowPasswordSignin) {
      router.replace({ name: AUTH_SIGNIN_MODULE, query: route.query });
      return;
    }
    state.email = route.query.email as string;
    startCountdown();
    return;
  }
  // Forced-reset mode: if user doesn't need to reset, redirect away.
  if (!authStore.requireResetPassword) {
    router.replace(redirectQuery.value);
  }
});

const allowConfirm = computed(() => {
  if (!state.password) return false;
  if (codeMode.value && (!state.email || state.codeParts.join("").length !== 6))
    return false;
  return (
    !userPasswordRef.value?.passwordHint &&
    !userPasswordRef.value?.passwordMismatch
  );
});

const onConfirm = async () => {
  if (codeMode.value) {
    try {
      await authServiceClientConnect.resetPassword(
        create(ResetPasswordRequestSchema, {
          email: state.email,
          code: state.codeParts.join(""),
          newPassword: state.password,
        })
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      // Auto-login with the new password so the user doesn't have to re-enter it.
      await authStore.login({
        request: create(LoginRequestSchema, {
          email: state.email,
          password: state.password,
          workspace: resolveWorkspaceName(),
        }),
      });
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("auth.password-reset.invalid-or-expired-code"),
      });
    }
    return;
  }

  // Legacy forced-reset mode.
  const patch: User = { ...me.value, password: state.password };
  await userStore.updateUser(
    createProto(UpdateUserRequestSchema, {
      user: patch,
      updateMask: createProto(FieldMaskSchema, { paths: ["password"] }),
    })
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  authStore.setRequireResetPassword(false);
  router.replace(redirectQuery.value);
};
</script>

<style scoped>
.email-code-otp :deep(.n-input) {
  flex: 1;
}
</style>
