<template>
  <div class="w-full space-y-4">
    <div class="textinfolabel">
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
      <div class="w-full flex flex-row gap-4">
        <div class="min-w-max w-80">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.smtp-server-host") }}
            <span class="text-red-600">*</span>
          </div>
          <NInput
            class="text-main w-full h-max mt-2"
            :placeholder="'smtp.gmail.com'"
            v-model:value="state.mailDeliverySetting.server"
          />
        </div>
        <div class="min-w-max w-48">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.smtp-server-port") }}
            <span class="text-red-600">*</span>
          </div>
          <NInput
            id="port"
            type="number"
            name="port"
            class="text-main w-full h-max mt-2 rounded-md border-control-border focus:ring-control focus:border-control disabled:bg-gray-50"
            :placeholder="'587'"
            :required="true"
            v-model:value="state.mailDeliverySetting.port"
            @wheel="(event: MouseEvent) => {(event.target as HTMLInputElement).blur()}"
          />
        </div>
      </div>
      <div class="w-full flex flex-row gap-4 mt-8">
        <div class="min-w-max w-80">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.from") }}
            <span class="text-red-600">*</span>
          </div>
          <NInput
            class="text-main w-full h-max mt-2"
            :placeholder="'from@gmail.com'"
            v-model:value="state.mailDeliverySetting.from"
          />
        </div>
      </div>
      <!-- Authentication Related -->
      <div class="w-full gap-4 mt-8">
        <div class="min-w-max w-80">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.authentication-method") }}
          </div>
          <NSelect
            class="mt-2"
            :value="getSelectedAuthenticationTypeItem"
            :options="authenticationTypeOptions"
            :virtual-scroll="true"
            :fallback-option="false"
            @update:value="handleSelectAuthenticationType"
          />
        </div>
      </div>
      <!-- Not NONE Authentication-->
      <template
        v-if="
          state.mailDeliverySetting.authentication !==
          SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_NONE
        "
      >
        <div class="w-full flex flex-row gap-4 mt-4">
          <div class="min-w-max w-80">
            <div class="flex flex-row">
              <label class="textlabel pl-1">
                {{ $t("settings.mail-delivery.field.smtp-username") }}
                <span class="text-red-600">*</span>
              </label>
            </div>
            <NInput
              class="text-main w-full h-max mt-2"
              :placeholder="'support@bytebase.com'"
              v-model:value="state.mailDeliverySetting.username"
            />
          </div>
          <div class="min-w-max w-80">
            <div class="flex flex-row space-x-2">
              <label class="textlabel pl-1">
                {{ $t("settings.mail-delivery.field.smtp-password") }}
                <span class="text-red-600">*</span>
              </label>
              <BBCheckbox
                :title="$t('common.empty')"
                :value="state.useEmptyPassword"
                @toggle="handleToggleUseEmptyPassword"
              />
            </div>
            <NInput
              class="text-main w-full h-max mt-2"
              :disabled="state.useEmptyPassword"
              :placeholder="'PASSWORD - INPUT_ONLY'"
              v-model:value="state.mailDeliverySetting.password"
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
          <NSelect
            class="mt-2"
            :value="getSelectedEncryptionTypeItem"
            :options="encryptionTypeOptions"
            :virtual-scroll="true"
            :fallback-option="false"
            @update:value="handleSelectEncryptionType"
          />
        </div>
      </div>
      <div class="flex flex-row w-full">
        <div class="w-auto gap-4 mt-8 flex flex-row">
          <NButton
            type="primary"
            :disabled="
              !allowMailDeliveryActionButton ||
              state.isSendLoading ||
              state.isCreateOrUpdateLoading
            "
            @click.prevent="updateMailDeliverySetting"
          >
            {{ mailDeliverySettingButtonText }}
          </NButton>
          <BBSpin v-if="state.isCreateOrUpdateLoading" class="ml-1" />
        </div>
      </div>
      <div class="border-b mt-4"></div>
      <!-- Test Send Email To Someone -->
      <div class="w-full gap-4 mt-4 flex flex-row">
        <div class="min-w-max w-160">
          <div class="textlabel pl-1">
            {{ $t("settings.mail-delivery.field.send-test-email-to") }}
          </div>
          <div class="flex flex-row justify-start items-center mt-2 space-x-4">
            <NInput
              class="text-main h-max w-80"
              :placeholder="'someone@gmail.com'"
              v-model:value="state.testMailTo"
            />
            <NButton
              type="primary"
              :disabled="
                state.testMailTo === '' ||
                state.isSendLoading ||
                state.isCreateOrUpdateLoading
              "
              @click.prevent="testMailDeliverySetting"
            >
              {{ $t("settings.mail-delivery.field.send") }}
            </NButton>
            <BBSpin v-if="state.isSendLoading" class="ml-2" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { SelectOption } from "naive-ui";
import { ClientError } from "nice-grpc-web";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBCheckbox } from "@/bbkit";
import { pushNotification } from "@/store";
import { useWorkspaceMailDeliverySettingStore } from "@/store/modules/workspaceMailDeliverySetting";
import {
  SMTPMailDeliverySettingValue,
  SMTPMailDeliverySettingValue_Authentication,
  SMTPMailDeliverySettingValue_Encryption,
} from "@/types/proto/v1/setting_service";

interface LocalState {
  originMailDeliverySetting?: SMTPMailDeliverySettingValue;
  mailDeliverySetting: SMTPMailDeliverySettingValue;
  testMailTo: string;
  isSendLoading: boolean;
  isCreateOrUpdateLoading: boolean;
  useEmptyPassword: boolean;
}
const { t } = useI18n();

const defaultMailDeliverySetting = function (): SMTPMailDeliverySettingValue {
  return {
    server: "",
    port: 587,
    username: "",
    password: undefined,
    from: "",
    authentication:
      SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_PLAIN,
    encryption: SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_STARTTLS,
    to: "",
  };
};

const state = reactive<LocalState>({
  mailDeliverySetting: defaultMailDeliverySetting(),
  testMailTo: "",
  isSendLoading: false,
  isCreateOrUpdateLoading: false,
  useEmptyPassword: false,
});

const store = useWorkspaceMailDeliverySettingStore();
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

onMounted(async () => {
  await store.fetchMailDeliverySetting();
  const setting = store.mailDeliverySetting;
  state.originMailDeliverySetting = cloneDeep(setting);
  if (state.originMailDeliverySetting) {
    state.mailDeliverySetting = cloneDeep(state.originMailDeliverySetting!);
  }
});

const updateMailDeliverySetting = async () => {
  state.isCreateOrUpdateLoading = true;
  const mailDelivery = cloneDeep(state.mailDeliverySetting);
  try {
    const value = cloneDeep(mailDelivery);
    await store.updateMailDeliverySetting(value);
  } catch (error) {
    state.isCreateOrUpdateLoading = false;
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: (error as ClientError).details,
    });
    return;
  }

  const currentValue = cloneDeep(store.mailDeliverySetting);
  state.originMailDeliverySetting = cloneDeep(currentValue);
  state.useEmptyPassword = false;
  if (state.originMailDeliverySetting) {
    state.mailDeliverySetting = cloneDeep(state.originMailDeliverySetting!);
  }
  state.isCreateOrUpdateLoading = false;
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.mail-delivery.updated-tip"),
  });
};

const testMailDeliverySetting = async () => {
  state.isSendLoading = true;
  const mailDelivery = cloneDeep(state.mailDeliverySetting);
  mailDelivery.to = state.testMailTo;
  try {
    await store.validateMailDeliverySetting(mailDelivery);
  } catch (error) {
    state.isSendLoading = false;
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: (error as ClientError).details,
    });
    return;
  }
  state.isSendLoading = false;
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.mail-delivery.tested-tip", { address: mailDelivery.to }),
  });
};

const handleSelectEncryptionType = (method: string) => {
  switch (method) {
    case "NONE":
      state.mailDeliverySetting.encryption =
        SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_NONE;
      break;
    case "SSL/TLS":
      state.mailDeliverySetting.encryption =
        SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_SSL_TLS;
      break;
    case "STARTTLS":
      state.mailDeliverySetting.encryption =
        SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_STARTTLS;
      break;
    default:
      state.mailDeliverySetting.encryption =
        SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_NONE;
      break;
  }
};

const encryptionTypeOptions = computed((): SelectOption[] => {
  return ["NONE", "SSL/TLS", "STARTTLS"].map((item) => ({
    value: item,
    label: item,
  }));
});

const authenticationTypeOptions = computed((): SelectOption[] => {
  return ["NONE", "PLAIN", "LOGIN", "CRAM-MD5"].map((item) => ({
    value: item,
    label: item,
  }));
});

const getSelectedEncryptionTypeItem = computed(() => {
  switch (state.mailDeliverySetting.encryption) {
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_NONE:
      return "NONE";
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_SSL_TLS:
      return "SSL/TLS";
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_STARTTLS:
      return "STARTTLS";
    default:
      return "NONE";
  }
});

const handleSelectAuthenticationType = (method: string) => {
  switch (method) {
    case "NONE":
      state.mailDeliverySetting.authentication =
        SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_NONE;
      break;
    case "PLAIN":
      state.mailDeliverySetting.authentication =
        SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_PLAIN;
      break;
    case "LOGIN":
      state.mailDeliverySetting.authentication =
        SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_LOGIN;
      break;
    case "CRAM-MD5":
      state.mailDeliverySetting.authentication =
        SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_CRAM_MD5;
      break;
    default:
      state.mailDeliverySetting.authentication =
        SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_PLAIN;
      break;
  }
};
const getSelectedAuthenticationTypeItem = computed(() => {
  switch (state.mailDeliverySetting.authentication) {
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_NONE:
      return "NONE";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_PLAIN:
      return "PLAIN";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_LOGIN:
      return "LOGIN";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_CRAM_MD5:
      return "CRAM-MD5";
    default:
      return "PLAIN";
  }
});

const handleToggleUseEmptyPassword = (on: boolean) => {
  state.useEmptyPassword = on;
  if (on) {
    state.mailDeliverySetting.password = "";
  } else {
    state.mailDeliverySetting.password = undefined;
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
