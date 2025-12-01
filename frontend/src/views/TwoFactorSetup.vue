<template>
  <p class="text-sm text-gray-500 mb-4">
    {{ $t("two-factor.description") }}
    <LearnMoreLink
      url="https://docs.bytebase.com/administration/2fa?source=console"
    />
  </p>
  <StepTab
    class="mb-8"
    :step-list="stepTabList"
    :allow-next="allowNext"
    :finish-title="$t('two-factor.setup-steps.recovery-codes-saved')"
    :current-index="state.currentStep"
    @update:current-index="tryChangeStep"
    @finish="tryFinishSetup"
    @cancel="cancelSetup"
  >
    <template #0>
      <div
        class="w-full max-w-2xl mx-auto flex flex-col justify-start items-start gap-y-4 my-8"
      >
        <p>
          {{ $t("two-factor.setup-steps.setup-auth-app.description") }}
        </p>
        <div
          :class="[
            'w-full border rounded-md p-3',
            state.isExpired || state.isExpiringSoon
              ? 'bg-red-50 border-red-200'
              : 'bg-yellow-50 border-yellow-200',
          ]"
        >
          <div class="flex items-center justify-between">
            <p
              :class="[
                'text-sm',
                state.isExpired || state.isExpiringSoon
                  ? 'text-red-800'
                  : 'text-yellow-800',
              ]"
            >
              ⏱️
              {{
                state.isExpired
                  ? $t("two-factor.setup-steps.setup-auth-app.expired-notice")
                  : $t("two-factor.setup-steps.setup-auth-app.time-remaining", {
                      time: state.timeRemaining,
                    })
              }}
            </p>
            <button
              v-if="state.isExpired"
              type="button"
              class="ml-3 px-3 py-1 text-sm font-medium text-white bg-blue-600 rounded-sm hover:bg-blue-700"
              @click="handleRegenerateSecret"
            >
              {{ $t("two-factor.setup-steps.setup-auth-app.regenerate") }}
            </button>
          </div>
        </div>
        <p class="text-2xl">
          {{ $t("two-factor.setup-steps.setup-auth-app.scan-qr-code.self") }}
        </p>
        <p>
          <i18n-t
            keypath="two-factor.setup-steps.setup-auth-app.scan-qr-code.description"
          >
            <template #action>
              <span
                :class="
                  !state.showSecretModal && 'cursor-pointer text-blue-600'
                "
                @click="state.showSecretModal = true"
                >{{
                  $t(
                    "two-factor.setup-steps.setup-auth-app.scan-qr-code.enter-the-text"
                  )
                }}</span
              >
            </template>
          </i18n-t>
        </p>
        <div class="w-full flex flex-col justify-center items-center pb-8">
          <NQrCode :value="otpauthUrl" :size="150" :padding="0" />
          <span class="mt-4 mb-2 text-sm font-medium">{{
            $t("two-factor.setup-steps.setup-auth-app.verify-code")
          }}</span>
          <NInputOtp
            v-model:value="state.otpCodes"
            @finish="onOtpCodeFinish"
          />
        </div>
      </div>
    </template>
    <template #1>
      <div class="w-full max-w-2xl mx-auto">
        <RecoveryCodesView
          :recovery-codes="currentUser.tempRecoveryCodes"
          @download="state.recoveryCodesDownloaded = true"
        />
      </div>
    </template>
  </StepTab>

  <TwoFactorSecretModal
    v-if="state.showSecretModal"
    :secret="currentUser.tempOtpSecret"
    @close="state.showSecretModal = false"
  />
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import type { ConnectError } from "@connectrpc/connect";
import { NInputOtp, NQrCode } from "naive-ui";
import { computed, onMounted, onUnmounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import RecoveryCodesView from "@/components/RecoveryCodesView.vue";
import TwoFactorSecretModal from "@/components/TwoFactorSecretModal.vue";
import { StepTab } from "@/components/v2";
import { AUTH_2FA_SETUP_MODULE } from "@/router/auth";
import { SETTING_ROUTE_PROFILE } from "@/router/dashboard/workspaceSetting";
import { pushNotification, useCurrentUserV1, useUserStore } from "@/store";
import { UpdateUserRequestSchema } from "@/types/proto-es/v1/user_service_pb";

const issuerName = "Bytebase";
const digits = 6;

const SETUP_AUTH_APP_STEP = 0;
const DOWNLOAD_RECOVERY_CODES_STEP = 1;

type Step = typeof SETUP_AUTH_APP_STEP | typeof DOWNLOAD_RECOVERY_CODES_STEP;

interface LocalState {
  currentStep: Step;
  showSecretModal: boolean;
  otpCodes: string[];
  recoveryCodesDownloaded: boolean;
  timeRemaining: string;
  isExpired: boolean;
  isExpiringSoon: boolean;
}

const props = defineProps<{
  cancelAction?: () => void;
}>();

const { t } = useI18n();
const router = useRouter();
const userStore = useUserStore();
const state = reactive<LocalState>({
  currentStep: SETUP_AUTH_APP_STEP,
  showSecretModal: false,
  otpCodes: [],
  recoveryCodesDownloaded: false,
  timeRemaining: "5:00",
  isExpired: false,
  isExpiringSoon: false,
});
const currentUser = useCurrentUserV1();
const MFA_TEMP_SECRET_EXPIRATION = 5 * 60 * 1000; // 5 minutes in milliseconds
let countdownInterval: ReturnType<typeof setInterval> | null = null;

const stepTabList = computed(() => {
  return [
    { title: t("two-factor.setup-steps.setup-auth-app.self") },
    { title: t("two-factor.setup-steps.download-recovery-codes.self") },
  ];
});

const allowNext = computed(() => {
  if (state.currentStep === SETUP_AUTH_APP_STEP) {
    return (
      state.otpCodes.filter((v) => v).length === digits && !state.isExpired
    );
  } else {
    return state.recoveryCodesDownloaded;
  }
});

const updateCountdown = () => {
  if (!currentUser.value.tempOtpSecretCreatedTime) {
    state.isExpired = true;
    state.timeRemaining = "0:00";
    return;
  }

  const createdAt =
    Number(currentUser.value.tempOtpSecretCreatedTime.seconds) * 1000;
  const now = Date.now();
  const elapsed = now - createdAt;
  const remaining = MFA_TEMP_SECRET_EXPIRATION - elapsed;

  if (remaining <= 0) {
    state.isExpired = true;
    state.timeRemaining = "0:00";
    state.isExpiringSoon = false;
    if (countdownInterval) {
      clearInterval(countdownInterval);
      countdownInterval = null;
    }
  } else {
    state.isExpired = false;
    const minutes = Math.floor(remaining / 60000);
    const seconds = Math.floor((remaining % 60000) / 1000);
    state.timeRemaining = `${minutes}:${seconds.toString().padStart(2, "0")}`;
    state.isExpiringSoon = remaining < 60000; // Less than 1 minute
  }
};

const startCountdown = () => {
  updateCountdown();
  if (countdownInterval) {
    clearInterval(countdownInterval);
  }
  countdownInterval = setInterval(updateCountdown, 1000);
};

const handleRegenerateSecret = async () => {
  state.otpCodes = [];
  await regenerateTempMfaSecret();
  startCountdown();
};

onMounted(async () => {
  await regenerateTempMfaSecret();
  startCountdown();
});

onUnmounted(() => {
  if (countdownInterval) {
    clearInterval(countdownInterval);
    countdownInterval = null;
  }
});

const regenerateTempMfaSecret = async () => {
  await userStore.updateUser(
    create(UpdateUserRequestSchema, {
      user: {
        name: currentUser.value.name,
      },
      updateMask: create(FieldMaskSchema, {
        paths: [],
      }),
      regenerateTempMfaSecret: true,
    })
  );
};

const verifyOTPCode = async () => {
  try {
    await userStore.updateUser(
      create(UpdateUserRequestSchema, {
        user: {
          name: currentUser.value.name,
        },
        updateMask: create(FieldMaskSchema, {
          paths: [],
        }),
        otpCode: state.otpCodes.join(""),
      })
    );
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: (error as ConnectError).message,
    });
    return false;
  }
  return true;
};

const cancelSetup = () => {
  if (props.cancelAction) {
    props.cancelAction();
  } else {
    router.replace({
      name: SETTING_ROUTE_PROFILE,
    });
  }
};

const onOtpCodeFinish = async (value: string[]) => {
  state.otpCodes = value;
  const result = await verifyOTPCode();
  if (result && state.currentStep === SETUP_AUTH_APP_STEP) {
    state.currentStep++;
  }
};

const tryChangeStep = async (nextStepIndex: number) => {
  switch (nextStepIndex) {
    case DOWNLOAD_RECOVERY_CODES_STEP: {
      const result = await verifyOTPCode();
      if (!result) {
        return;
      }
      break;
    }
    case SETUP_AUTH_APP_STEP:
      state.otpCodes = [];
      break;
    default:
      break;
  }
  state.currentStep = nextStepIndex as Step;
};

const tryFinishSetup = async () => {
  await userStore.updateUser(
    create(UpdateUserRequestSchema, {
      user: {
        name: currentUser.value.name,
        mfaEnabled: true,
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["mfa_enabled"],
      }),
    })
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("two-factor.messages.2fa-enabled"),
  });

  // When user is on the 2fa.setup page, we'd like to redirect to the home page.
  // Only happens when the workspace level 2fa-setup is enabled.
  if (router.currentRoute.value.name === AUTH_2FA_SETUP_MODULE) {
    router.replace({
      path: "/",
    });
  } else {
    router.replace({
      name: SETTING_ROUTE_PROFILE,
    });
  }
};

const otpauthUrl = computed(
  () =>
    `otpauth://totp/${issuerName}:${currentUser.value.email}?algorithm=SHA1&digits=${digits}&issuer=${issuerName}&period=30&secret=${currentUser.value.tempOtpSecret}`
);
</script>
