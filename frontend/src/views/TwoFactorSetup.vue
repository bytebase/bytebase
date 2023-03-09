<template>
  <p class="text-sm text-gray-500 mb-4">
    {{ $t("two-factor.description") }}
    <LearnMoreLink url="" />
  </p>
  <BBStepTab
    class="mb-8"
    :step-item-list="stepTabList"
    :allow-next="allowNext"
    :finish-title="$t('two-factor.setup-steps.recovery-codes-saved')"
    :current-step="state.currentStep"
    @cancel="cancelSetup"
    @try-change-step="tryChangeStep"
    @try-finish="tryFinishSetup"
  >
    <template #0>
      <div
        class="w-full max-w-2xl mx-auto flex flex-col justify-start items-start space-y-4 my-8"
      >
        <p>
          {{ $t("two-factor.setup-steps.setup-auth-app.description") }}
        </p>
        <p class="text-2xl">
          {{ $t("two-factor.setup-steps.setup-auth-app.scan-qr-code.self") }}
        </p>
        <p>
          <i18n-t
            keypath="two-factor.setup-steps.setup-auth-app.scan-qr-code.description"
          >
            <template #action>
              <span
                :class="!state.showSecretText && 'cursor-pointer text-blue-600'"
                @click="state.showSecretText = true"
                >{{
                  $t(
                    "two-factor.setup-steps.setup-auth-app.scan-qr-code.enter-the-text"
                  )
                }}</span
              >
              <code
                v-if="state.showSecretText"
                class="ml-1 bg-gray-100 px-1 rounded font-mono"
                >{{ currentUser.mfaSecret }}</code
              >
            </template>
          </i18n-t>
        </p>
        <div class="w-full flex flex-col justify-center items-center pb-8">
          <img
            :src="state.qrcodeDataUrl"
            class="border w-64 mt-4 rounded-lg"
            alt=""
          />
          <span class="mt-4 mb-2 text-sm font-medium">{{
            $t("two-factor.setup-steps.setup-auth-app.verify-code")
          }}</span>
          <input
            v-model="state.otpCode"
            class="textfield w-64"
            placeholder="XXXXXX"
            type="text"
          />
        </div>
      </div>
    </template>
    <template #1>
      <RecoveryCodesView
        :recovery-codes="currentUser.recoveryCodes"
        @download="state.recoveryCodesDownloaded = true"
      />
    </template>
  </BBStepTab>
</template>

<script lang="ts" setup>
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { computed, onMounted, reactive, watch } from "vue";
import { pushNotification, useAuthStore, useUserStore } from "@/store";
import * as QRCode from "qrcode";
import { UpdateUserRequest } from "@/types/proto/v1/auth_service";
import RecoveryCodesView from "@/components/RecoveryCodesView.vue";
import { useRouter } from "vue-router";

const issuerName = "Bytebase";

const SETUP_AUTH_APP_STEP = 0;
const DOWNLOAD_RECOVERY_CODES_STEP = 1;

type Step = typeof SETUP_AUTH_APP_STEP | typeof DOWNLOAD_RECOVERY_CODES_STEP;

interface LocalState {
  currentStep: Step;
  showSecretText: boolean;
  qrcodeDataUrl: string;
  otpCode: string;
  recoveryCodesDownloaded: boolean;
}

const { t } = useI18n();
const router = useRouter();
const authStore = useAuthStore();
const userStore = useUserStore();
const state = reactive<LocalState>({
  currentStep: SETUP_AUTH_APP_STEP,
  showSecretText: false,
  qrcodeDataUrl: "",
  otpCode: "",
  recoveryCodesDownloaded: false,
});

const stepTabList = computed(() => {
  return [
    { title: t("two-factor.setup-steps.setup-auth-app.self") },
    { title: t("two-factor.setup-steps.download-recovery-codes.self") },
  ];
});
const currentUser = computed(() => authStore.currentUser);
const allowNext = computed(() => {
  if (state.currentStep === SETUP_AUTH_APP_STEP) {
    return state.otpCode.length >= 6;
  } else {
    return state.recoveryCodesDownloaded;
  }
});

onMounted(async () => {
  await regenerateTempMfaSecret();
});

const regenerateTempMfaSecret = async () => {
  await userStore.updateUser(
    UpdateUserRequest.fromPartial({
      user: {
        name: currentUser.value.name,
      },
      updateMask: [],
      regenerateTempMfaSecret: true,
    })
  );
  await authStore.refreshUserIfNeeded(currentUser.value.name);
};

const verifyTOPCode = async () => {
  try {
    await userStore.updateUser(
      UpdateUserRequest.fromPartial({
        user: {
          name: currentUser.value.name,
        },
        updateMask: [],
        otpCode: state.otpCode,
      })
    );
  } catch (error: any) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: error.details,
    });
    return false;
  }
  return true;
};

const cancelSetup = () => {
  router.replace({
    name: "setting.profile",
  });
};

const tryChangeStep = async (
  oldStep: number,
  newStep: number,
  allowChangeCallback: () => void
) => {
  if (newStep === DOWNLOAD_RECOVERY_CODES_STEP) {
    const result = await verifyTOPCode();
    if (!result) {
      return;
    }
  }
  state.currentStep = newStep as Step;
  allowChangeCallback();
};

const tryFinishSetup = async () => {
  await userStore.updateUser(
    UpdateUserRequest.fromPartial({
      user: {
        name: currentUser.value.name,
        mfaEnabled: true,
      },
      updateMask: ["user.mfa_enabled"],
    })
  );
  await authStore.refreshUserIfNeeded(currentUser.value.name);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("two-factor.messages.2fa-enabled"),
  });
  router.replace({
    name: "setting.profile",
  });
};

watch(
  [currentUser],
  async () => {
    const otpauthUrl = `otpauth://totp/${issuerName}:${currentUser.value.email}?algorithm=SHA1&digits=6&issuer=${issuerName}&period=30&secret=${currentUser.value.mfaSecret}`;
    state.qrcodeDataUrl = await QRCode.toDataURL(otpauthUrl);
  },
  { deep: true, immediate: true }
);
</script>
