<template>
  <form class="flex flex-col gap-y-6 px-1" @submit.prevent="handleSubmit">
    <div>
      <label
        for="email-code-email"
        class="block text-sm font-medium leading-5 text-control"
      >
        {{ $t("common.email") }}
        <RequiredStar />
      </label>
      <div class="mt-1 rounded-md shadow-xs">
        <BBTextField
          v-model:value="state.email"
          required
          :input-props="{
            id: 'email-code-email',
            autocomplete: 'email',
            type: 'email',
          }"
          placeholder="jim@example.com"
          :disabled="step === 'code' || emailFromQuery"
        />
      </div>
    </div>

    <div v-if="step === 'code'" class="flex flex-col gap-y-2">
      <label class="block text-sm font-medium leading-5 text-control">
        {{ $t("auth.sign-in.verification-code") }}
        <RequiredStar />
      </label>
      <div class="text-sm text-control-light">
        {{ $t("auth.sign-in.code-sent-hint", { email: state.email }) }}
      </div>
      <NInputOtp
        v-model:value="state.codeParts"
        :length="6"
        class="email-code-otp w-full"
        @finish="handleSubmit"
      />
      <div class="flex items-center justify-end">
        <NButton text :disabled="resendCountdown > 0" @click="sendCode">
          {{
            resendCountdown > 0
              ? $t("auth.sign-in.resend-in", { seconds: resendCountdown })
              : $t("auth.sign-in.resend-code")
          }}
        </NButton>
      </div>
    </div>

    <div class="w-full">
      <NButton
        v-if="step === 'email'"
        attr-type="button"
        type="primary"
        :disabled="!isValidEmail(state.email) || state.sending"
        :loading="state.sending"
        size="large"
        style="width: 100%"
        @click="sendCode"
      >
        {{ $t("auth.sign-in.send-code") }}
      </NButton>
      <NButton
        v-else
        attr-type="submit"
        type="primary"
        :disabled="!allowSignin"
        :loading="props.loading"
        size="large"
        style="width: 100%"
      >
        {{ $t("common.sign-in") }}
      </NButton>
    </div>
  </form>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NButton, NInputOtp } from "naive-ui";
import { computed, onMounted, onUnmounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBTextField } from "@/bbkit";
import RequiredStar from "@/components/RequiredStar.vue";
import { pushNotification, useAuthStore } from "@/store";
import {
  type LoginRequest,
  LoginRequestSchema,
} from "@/types/proto-es/v1/auth_service_pb";
import { isValidEmail, resolveWorkspaceName } from "@/utils";

interface LocalState {
  email: string;
  codeParts: string[];
  sending: boolean;
}

const props = defineProps<{
  loading: boolean;
}>();

const emit = defineEmits<{
  (event: "signin", payload: LoginRequest): void;
}>();

const { t } = useI18n();
const authStore = useAuthStore();

const step = ref<"email" | "code">("email");
const resendCountdown = ref(0);
let countdownTimer: ReturnType<typeof setInterval> | null = null;

const state = reactive<LocalState>({
  email: "",
  codeParts: [],
  sending: false,
});

const emailFromQuery = ref(false);

onMounted(() => {
  const url = new URL(window.location.href);
  const params = new URLSearchParams(url.search);
  const queryEmail = params.get("email") ?? "";
  if (queryEmail) {
    state.email = queryEmail;
    emailFromQuery.value = true;
  }
});

onUnmounted(() => {
  if (countdownTimer) clearInterval(countdownTimer);
});

const allowSignin = computed(() => {
  return state.email && state.codeParts.join("").length === 6;
});

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

const sendCode = async () => {
  if (!isValidEmail(state.email) || state.sending || resendCountdown.value > 0)
    return;
  state.sending = true;
  try {
    await authStore.sendEmailLoginCode(state.email, resolveWorkspaceName());
    step.value = "code";
    startCountdown();
  } catch (e) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("auth.sign-in.failed-to-send-code", {
        error: e,
      }),
    });
  } finally {
    state.sending = false;
  }
};

const handleSubmit = async () => {
  if (step.value === "email") {
    await sendCode();
    return;
  }
  const code = state.codeParts.join("");
  if (!state.email || code.length !== 6) return;
  emit(
    "signin",
    create(LoginRequestSchema, {
      email: state.email,
      emailCode: code,
      workspace: resolveWorkspaceName(),
    })
  );
};
</script>

<style scoped>
.email-code-otp :deep(.n-input) {
  flex: 1;
}
</style>
