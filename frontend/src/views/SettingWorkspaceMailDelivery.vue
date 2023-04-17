<template>
  <div class="w-full mt-4 space-y-4">
    <div class="pt-4 border-b textinfolabel p-2">
      {{ $t("settings.mail-delivery.description") }}
      <a
        class="normal-link inline-flex items-center"
        href="https://www.bytebase.com/docs/administration/mail-delivery?source=console"
        target="__BLANK"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4 ml-1" />
      </a>
    </div>
    <div class="w-full flex flex-col">
      <!-- Host and Port -->
      <div class="w-full flex flex-row gap-4 mt-8">
        <div class="min-w-max w-80">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.smtp-server-host") }}
            <span class="text-red-600">*</span>
          </div>
          <BBTextField
            class="text-main w-full h-max mt-2"
            :placeholder="'smtp.gmail.com'"
            :value="state.mailDeliverySetting?.smtpServerHost"
            @input="(e: any) =>
            state.mailDeliverySetting!.smtpServerHost = e.target.value"
          />
        </div>
        <div class="min-w-max w-48">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.smtp-server-port") }}
            <span class="text-red-600">*</span>
          </div>
          <input
            id="port"
            type="number"
            name="port"
            class="text-main w-full h-max mt-2 rounded-md border-control-border focus:ring-control focus:border-control disabled:bg-gray-50"
            :placeholder="'587'"
            :required="true"
            :value="state.mailDeliverySetting?.smtpServerPort"
            @wheel="(event: MouseEvent) => {(event.target as HTMLInputElement).blur()}"
            @input="(event) => {state.mailDeliverySetting!.smtpServerPort = (event.target as HTMLInputElement).valueAsNumber}"
          />
        </div>
      </div>
      <div class="w-full flex flex-row gap-4 mt-8">
        <div class="min-w-max w-80">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.from") }}
            <span class="text-red-600">*</span>
          </div>
          <BBTextField
            class="text-main w-full h-max mt-2"
            :placeholder="'from@gmail.com'"
            :value="state.mailDeliverySetting?.smtpFrom"
            @input="(e: any) => state.mailDeliverySetting!.smtpFrom = e.target.value"
          />
        </div>
      </div>

      <!-- Authentication Related -->
      <div class="w-full gap-4 mt-8">
        <div class="min-w-max w-80">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.authentication-method") }}
          </div>
          <BBSelect
            class="mt-2"
            :selected-item="getSelectedAuthenticationTypeItem"
            :item-list="['NONE', 'PLAIN', 'LOGIN', 'CRAM-MD5']"
            :disabled="false"
            :show-prefix-item="true"
            @select-item="handleSelectAuthenticationType"
          >
            <template #menuItem="{ item }">
              <div class="text-main flex items-center gap-x-2">
                {{ item }}
              </div>
            </template></BBSelect
          >
        </div>
      </div>
      <!-- Not NONE Authentication-->
      <template
        v-if="
          state.mailDeliverySetting.smtpAuthenticationType !==
          SMTPMailDeliverySetting_Authentication.AUTHENTICATION_NONE
        "
      >
        <div class="w-full flex flex-row gap-4 mt-4">
          <div class="min-w-max w-80">
            <div class="flex flex-row">
              <label class="textlabel pl-1">
                {{ $t("settings.mail-delivery.field.smtp-username") }}
              </label>
            </div>
            <BBTextField
              class="text-main w-full h-max mt-2"
              :placeholder="'support@bytebase.com'"
              :value="state.mailDeliverySetting?.smtpUsername"
              @input="(e: any) => state.mailDeliverySetting!.smtpUsername = e.target.value"
            />
          </div>
          <div class="min-w-max w-80">
            <div class="flex flex-row space-x-2">
              <label class="textlabel pl-1">
                {{ $t("settings.mail-delivery.field.smtp-password") }}
              </label>
              <BBCheckbox
                :title="$t('common.empty')"
                :value="state.useEmptyPassword"
                @toggle="handleToggleUseEmptyPassword"
              />
            </div>
            <BBTextField
              class="text-main w-full h-max mt-2"
              :disabled="state.useEmptyPassword"
              :placeholder="'PASSWORD - INPUT_ONLY'"
              :value="state.mailDeliverySetting?.smtpPassword"
              @input="(e: any) => state.mailDeliverySetting!.smtpPassword = e.target.value"
            />
          </div>
        </div>
      </template>

      <!-- Encryption Related -->
      <div class="w-full gap-4 mt-8">
        <div class="min-w-max w-80">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.encryption") }}
          </div>
          <BBSelect
            class="mt-2"
            :selected-item="getSelectedEncryptionTypeItem"
            :item-list="['NONE', 'SSL/TLS', 'STARTTLS']"
            :disabled="false"
            :show-prefix-item="true"
            @select-item="handleSelectEncryptionType"
          >
            <template #menuItem="{ item }">
              <div class="text-main flex items-center gap-x-2">
                {{ item }}
              </div>
            </template></BBSelect
          >
        </div>
      </div>

      <!-- Test Send Email To Someone -->
      <div class="w-full gap-4 mt-8">
        <div class="min-w-max w-80">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.send-test-email-to") }}
          </div>
          <BBTextField
            class="text-main w-full h-max mt-2"
            :placeholder="'someone@gmail.com'"
            :value="state.testMailTo"
            @input="(e: any) => state.testMailTo = e.target.value"
          />
        </div>
      </div>

      <div class="flex flex-row w-full">
        <div class="w-auto gap-4 mt-8">
          <button
            type="button"
            class="btn-primary inline-flex justify-center py-2 px-4"
            :disabled="state.testMailTo === '' || state.isLoading"
            @click.prevent="testMailDeliverySetting"
          >
            {{ $t("settings.mail-delivery.field.send") }}
          </button>
          <BBSpin v-if="state.isLoading" class="ml-1" />
        </div>

        <div class="w-auto gap-4 mt-8 ml-10">
          <button
            type="button"
            class="btn-primary inline-flex justify-center py-2 px-4"
            :disabled="!allowMailDeliveryActionButton || state.isLoading"
            @click.prevent="updateMailDeliverySetting"
          >
            {{ mailDeliverySettingButtonText }}
          </button>
          <BBSpin v-if="state.isLoading" class="ml-1" />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { BBCheckbox, BBSelect, BBTextField } from "@/bbkit";
import { useSettingStore, pushNotification } from "@/store";
import {
  SettingWorkspaceMailDeliveryValue,
  TestWorkspaceDeliveryValue,
} from "@/types/setting";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { cloneDeep, isEqual } from "lodash-es";
import {
  SMTPMailDeliverySetting_Encryption,
  SMTPMailDeliverySetting_Authentication,
} from "@/types/proto/store/setting";

interface LocalState {
  originMailDeliverySetting?: SettingWorkspaceMailDeliveryValue;
  mailDeliverySetting: SettingWorkspaceMailDeliveryValue;
  testMailTo: string;
  isLoading: boolean;
  useEmptyPassword: boolean;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  mailDeliverySetting: {
    smtpServerHost: "",
    smtpServerPort: NaN,
    smtpUsername: "",
    smtpPassword: "",
    smtpFrom: "",
    smtpAuthenticationType:
      SMTPMailDeliverySetting_Authentication.AUTHENTICATION_LOGIN,
    smtpEncryptionType: SMTPMailDeliverySetting_Encryption.ENCRYPTION_STARTTLS,
  },
  testMailTo: "",
  isLoading: false,
  useEmptyPassword: false,
});
const settingStore = useSettingStore();

const mailDeliverySettingButtonText = computed(() => {
  return state.originMailDeliverySetting === undefined
    ? t("common.create")
    : t("common.update");
});

const allowMailDeliveryActionButton = computed(() => {
  return (
    state.useEmptyPassword ||
    !isEqual(state.originMailDeliverySetting, state.mailDeliverySetting)
  );
});

onMounted(() => {
  const setting = settingStore.getSettingByName("bb.workspace.mail-delivery");
  if (setting) {
    const mailDelivery = JSON.parse(
      setting.value || "{}"
    ) as SettingWorkspaceMailDeliveryValue;
    state.originMailDeliverySetting = cloneDeep(mailDelivery);
    state.mailDeliverySetting = mailDelivery;
  }
});

const updateMailDeliverySetting = async () => {
  state.isLoading = true;
  const mailDelivery = cloneDeep(state.mailDeliverySetting);
  try {
    await settingStore.updateSettingByName({
      name: "bb.workspace.mail-delivery",
      value: JSON.stringify(mailDelivery),
    });
  } catch (error) {
    state.isLoading = false;
    return;
  }

  // Remove the sensitive information from the state.
  mailDelivery.smtpPassword = "";
  state.isLoading = false;
  state.originMailDeliverySetting = cloneDeep(mailDelivery);
  state.useEmptyPassword = false;
  state.mailDeliverySetting = mailDelivery;

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.mail-delivery.updated-tip"),
  });
};

const testMailDeliverySetting = async () => {
  state.isLoading = true;
  const mailDelivery = cloneDeep(state.mailDeliverySetting);
  const testMailTo = state.testMailTo;
  const testValue = {
    ...mailDelivery,
    sendTo: testMailTo,
  } as TestWorkspaceDeliveryValue;

  try {
    await settingStore.validateSettingByName({
      name: "bb.workspace.mail-delivery",
      value: JSON.stringify(testValue),
    });
  } catch (error) {
    state.isLoading = false;
    return;
  }

  state.isLoading = false;
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.mail-delivery.tested-tip", { address: testMailTo }),
  });
};

const handleSelectEncryptionType = (method: string) => {
  switch (method) {
    case "NONE":
      state.mailDeliverySetting.smtpEncryptionType =
        SMTPMailDeliverySetting_Encryption.ENCRYPTION_NONE;
      break;
    case "SSL/TLS":
      state.mailDeliverySetting.smtpEncryptionType =
        SMTPMailDeliverySetting_Encryption.ENCRYPTION_SSL_TLS;
      break;
    case "STARTTLS":
      state.mailDeliverySetting.smtpEncryptionType =
        SMTPMailDeliverySetting_Encryption.ENCRYPTION_STARTTLS;
      break;
    default:
      state.mailDeliverySetting.smtpEncryptionType =
        SMTPMailDeliverySetting_Encryption.ENCRYPTION_NONE;
      break;
  }
};

const getSelectedEncryptionTypeItem = computed(() => {
  switch (state.mailDeliverySetting.smtpEncryptionType) {
    case SMTPMailDeliverySetting_Encryption.ENCRYPTION_NONE:
      return "NONE";
    case SMTPMailDeliverySetting_Encryption.ENCRYPTION_SSL_TLS:
      return "SSL/TLS";
    case SMTPMailDeliverySetting_Encryption.ENCRYPTION_STARTTLS:
      return "STARTTLS";
    default:
      return "NONE";
  }
});

const handleSelectAuthenticationType = (method: string) => {
  switch (method) {
    case "NONE":
      state.mailDeliverySetting.smtpAuthenticationType =
        SMTPMailDeliverySetting_Authentication.AUTHENTICATION_NONE;
      break;
    case "PLAIN":
      state.mailDeliverySetting.smtpAuthenticationType =
        SMTPMailDeliverySetting_Authentication.AUTHENTICATION_PLAIN;
      break;
    case "LOGIN":
      state.mailDeliverySetting.smtpAuthenticationType =
        SMTPMailDeliverySetting_Authentication.AUTHENTICATION_LOGIN;
      break;
    case "CRAM-MD5":
      state.mailDeliverySetting.smtpAuthenticationType =
        SMTPMailDeliverySetting_Authentication.AUTHENTICATION_CRAM_MD5;
      break;
    default:
      state.mailDeliverySetting.smtpAuthenticationType =
        SMTPMailDeliverySetting_Authentication.AUTHENTICATION_PLAIN;
      break;
  }
};

const getSelectedAuthenticationTypeItem = computed(() => {
  switch (state.mailDeliverySetting.smtpAuthenticationType) {
    case SMTPMailDeliverySetting_Authentication.AUTHENTICATION_NONE:
      return "NONE";
    case SMTPMailDeliverySetting_Authentication.AUTHENTICATION_PLAIN:
      return "PLAIN";
    case SMTPMailDeliverySetting_Authentication.AUTHENTICATION_LOGIN:
      return "LOGIN";
    case SMTPMailDeliverySetting_Authentication.AUTHENTICATION_CRAM_MD5:
      return "CRAM-MD5";
    default:
      return "PLAIN";
  }
});

const handleToggleUseEmptyPassword = (on: boolean) => {
  state.useEmptyPassword = on;
  if (on) {
    state.mailDeliverySetting.smtpPassword = "";
  }
};
</script>

<style scoped>
/*  Removed the ticker in the number field  */
input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* Firefox */
input[type="number"] {
  -moz-appearance: textfield;
}
</style>
