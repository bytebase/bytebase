<template>
  <div
    class="w-full flex flex-col justify-start items-start overflow-x-hidden px-1"
  >
    <div
      v-if="isCreating"
      class="w-full flex flex-col justify-start items-start"
    >
      <div class="w-full flex flex-row justify-between items-center">
        <p class="textlabel my-2">{{ $t("settings.sso.form.type") }}</p>
        <a
          v-if="userDocLink"
          :href="userDocLink"
          class="normal-link text-sm inline-flex flex-row items-center"
          target="_blank"
        >
          {{ $t("settings.sso.form.learn-more-with-user-doc") }}
          <heroicons-outline:external-link class="w-4 h-4" />
        </a>
      </div>
      <div class="w-full flex flex-row justify-start items-start space-x-2">
        <label
          v-for="item in identityProviderTypeList"
          :key="item"
          class="flex flex-row justify-center items-center cursor-pointer mr-2"
          :for="`radio-${item}`"
        >
          <input
            :id="`radio-${item}`"
            v-model="state.type"
            type="radio"
            class="btn mr-2"
            :value="item"
            :checked="state.type === item"
          />
          <span>{{ identityProviderTypeToString(item) }}</span>
        </label>
      </div>
    </div>

    <!-- OAuth2 templates group -->
    <div
      v-if="isCreating && state.type === IdentityProviderType.OAUTH2"
      class="w-full flex flex-col justify-start items-start space-y-3"
    >
      <p class="textlabel mt-4">
        {{ $t("settings.sso.form.use-template") }}
      </p>
      <div class="w-full flex flex-row justify-start items-start space-x-2">
        <label
          v-for="template in templateList"
          :key="template.title"
          class="w-24 h-24 border rounded-md flex flex-col justify-center items-center cursor-pointer hover:bg-gray-100"
          :for="`radio-${template.title}`"
          @click="handleTemplateSelect(template)"
        >
          <span>{{ template.title }}</span>
          <input
            :id="`radio-${template.title}`"
            type="radio"
            class="btn mt-4"
            :checked="selectedTemplate?.title === template.title"
          />
        </label>
      </div>
    </div>

    <div class="w-full flex flex-col justify-start items-start space-y-3">
      <p class="text-lg font-medium !mt-4">
        {{ $t("settings.sso.form.basic-information") }}
      </p>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.name") }}
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="identityProvider.title"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          :placeholder="$t('settings.sso.form.name-description')"
        />
        <ResourceIdField
          ref="resourceIdField"
          resource-type="idp"
          :readonly="!isCreating"
          :value="resourceId"
          :resource-title="identityProvider.title"
          :validate="validateResourceId"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.domain") }}
        </p>
        <input
          v-model="identityProvider.domain"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          :placeholder="$t('settings.sso.form.domain-description')"
        />
      </div>
    </div>

    <!-- OAuth2 form group -->
    <div
      v-if="state.type === IdentityProviderType.OAUTH2"
      class="w-full flex flex-col justify-start items-start space-y-3"
    >
      <div class="w-full flex flex-col justify-start items-start">
        <p class="text-lg font-medium mt-2">
          {{ $t("settings.sso.form.identity-provider-information") }}
        </p>
        <p class="textinfolabel">
          {{
            $t("settings.sso.form.identity-provider-information-description")
          }}
        </p>
      </div>
      <div
        v-if="isCreating"
        class="w-auto max-w-full p-4 rounded flex flex-col justify-start items-start border"
      >
        <p class="textinfolabel flex flex-row justify-start items-center mb-2">
          {{ $t("settings.sso.form.identity-provider-needed-information") }}
          <ShowMoreIcon
            class="ml-1 mr-2"
            :content="$t('settings.sso.form.redirect-url-description')"
          />
        </p>
        <div class="w-128 flex flex-row justify-start items-center space-x-4">
          <p class="textlabel my-auto text-right whitespace-nowrap">
            {{ $t("settings.sso.form.redirect-url") }}
          </p>
          <div class="w-full relative break-all pr-8 text-sm">
            {{ redirectUrl }}
            <button
              tabindex="-1"
              class="absolute right-0 top-1/2 -translate-y-1/2 p-1 text-control-light rounded hover:bg-gray-100"
              @click.prevent="copyRedirectUrl"
            >
              <heroicons-outline:clipboard class="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Client ID
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="configForOAuth2.clientId"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. 6655asd77895265aa110ac0d3"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Client secret
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="configForOAuth2.clientSecret"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          :placeholder="
            isCreating
              ? 'e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8'
              : $t('common.sensitive-placeholder')
          "
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Auth URL
          <span class="text-red-600">*</span>
        </p>
        <p class="textinfolabel">
          {{ $t("settings.sso.form.auth-url-description") }}
        </p>
        <input
          v-model="configForOAuth2.authUrl"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. https://github.com/login/oauth/authorize"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Scopes
          <span class="text-red-600">*</span>
        </p>
        <p class="textinfolabel">
          {{ $t("settings.sso.form.scopes-description") }}
        </p>
        <input
          v-model="scopesStringOfConfig"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. user"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Token URL
          <span class="text-red-600">*</span>
        </p>
        <p class="textinfolabel">
          {{ $t("settings.sso.form.token-url-description") }}
        </p>
        <input
          v-model="configForOAuth2.tokenUrl"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. https://github.com/login/oauth/access_token"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          User information URL
          <span class="text-red-600">*</span>
        </p>
        <p class="textinfolabel">
          {{ $t("settings.sso.form.user-info-url-description") }}
        </p>
        <input
          v-model="configForOAuth2.userInfoUrl"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. https://api.github.com/user"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.auth-style.self") }}
          <span class="text-red-600">*</span>
        </p>
        <p class="textinfolabel mt-1">
          <NRadioGroup v-model:value="configForOAuth2.authStyle">
            <NRadio
              :value="OAuth2AuthStyle.IN_PARAMS"
              :label="$t('settings.sso.form.auth-style.in-params.self')"
            />
            <NRadio
              :value="OAuth2AuthStyle.IN_HEADER"
              :label="$t('settings.sso.form.auth-style.in-header.self')"
            />
          </NRadioGroup>
        </p>
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.connection-security") }}
        </p>
        <p class="textinfolabel mt-1">
          <BBCheckbox
            :title="$t('settings.sso.form.connection-security-skip-tls-verify')"
            :v-model="configForOAuth2.skipTlsVerify"
            :disabled="!allowEdit"
            @toggle="
              () =>
                (configForOAuth2.skipTlsVerify = !configForOAuth2.skipTlsVerify)
            "
          />
        </p>
      </div>

      <div class="w-full flex flex-col justify-start items-start">
        <p class="text-lg font-medium mt-2">
          {{ $t("settings.sso.form.user-information-mapping") }}
        </p>
        <p class="textinfolabel">
          {{ $t("settings.sso.form.user-information-mapping-description") }}
          <a
            href="https://www.bytebase.com/docs/administration/sso/oauth2#user-information-field-mapping?source=console"
            class="normal-link text-sm inline-flex flex-row items-center"
            target="_blank"
          >
            {{ $t("common.learn-more") }}
            <heroicons-outline:external-link class="w-4 h-4" />
          </a>
        </p>
      </div>
      <div class="w-full grid grid-cols-[256px_1fr]">
        <input
          v-model="configForOAuth2.fieldMapping!.identifier"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. login"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p class="flex flex-row justify-start items-center">
            {{ $t("settings.sso.form.identifier") }}
            <NTooltip>
              <template #trigger>
                <heroicons-outline:information-circle
                  class="ml-1 w-4 h-auto text-blue-500"
                />
              </template>
              {{ $t("settings.sso.form.identifier-tips") }}
            </NTooltip>
          </p>
        </div>
      </div>
      <div class="w-full grid grid-cols-[256px_1fr]">
        <input
          v-model="configForOAuth2.fieldMapping!.displayName"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. name"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p>
            {{ $t("settings.sso.form.display-name") }}
          </p>
        </div>
      </div>
      <div class="w-full grid grid-cols-[256px_1fr]">
        <input
          v-model="configForOAuth2.fieldMapping!.phone"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. phone"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p>
            {{ $t("settings.sso.form.phone") }}
          </p>
        </div>
      </div>
    </div>

    <!-- OIDC form group -->
    <div
      v-else-if="state.type === IdentityProviderType.OIDC"
      class="w-full flex flex-col justify-start items-start space-y-3"
    >
      <div class="w-full flex flex-col justify-start items-start">
        <p class="text-lg font-medium mt-2">
          {{ $t("settings.sso.form.identity-provider-information") }}
        </p>
        <p class="textinfolabel">
          {{
            $t("settings.sso.form.identity-provider-information-description")
          }}
        </p>
      </div>
      <div
        v-if="isCreating"
        class="w-auto max-w-full p-4 rounded flex flex-col justify-start items-start border"
      >
        <p class="textinfolabel flex flex-row justify-start items-center mb-2">
          {{ $t("settings.sso.form.identity-provider-needed-information") }}
          <ShowMoreIcon
            class="ml-1 mr-2"
            :content="$t('settings.sso.form.redirect-url-description')"
          />
        </p>
        <div class="w-128 flex flex-row justify-start items-center space-x-4">
          <p class="textlabel my-auto text-right whitespace-nowrap">
            {{ $t("settings.sso.form.redirect-url") }}
          </p>
          <div class="w-full relative break-all pr-8 text-sm">
            {{ redirectUrl }}
            <button
              tabindex="-1"
              class="absolute right-0 top-1/2 -translate-y-1/2 p-1 text-control-light rounded hover:bg-gray-100"
              @click.prevent="copyRedirectUrl"
            >
              <heroicons-outline:clipboard class="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Issuer
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="configForOIDC.issuer"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. https://acme.okta.com"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Client ID
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="configForOIDC.clientId"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. 6655asd77895265aa110ac0d3"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Client secret
          <span class="text-red-600">*</span>
        </p>
        <input
          v-model="configForOIDC.clientSecret"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          :placeholder="
            isCreating
              ? 'e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8'
              : $t('common.sensitive-placeholder')
          "
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.auth-style.self") }}
          <span class="text-red-600">*</span>
        </p>
        <p class="textinfolabel mt-1">
          <NRadioGroup v-model:value="configForOIDC.authStyle">
            <NRadio
              :value="OAuth2AuthStyle.IN_PARAMS"
              :label="$t('settings.sso.form.auth-style.in-params.self')"
            />
            <NRadio
              :value="OAuth2AuthStyle.IN_HEADER"
              :label="$t('settings.sso.form.auth-style.in-header.self')"
            />
          </NRadioGroup>
        </p>
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.connection-security") }}
        </p>
        <p class="textinfolabel mt-1">
          <BBCheckbox
            :title="$t('settings.sso.form.connection-security-skip-tls-verify')"
            :value="configForOIDC.skipTlsVerify"
            :disabled="!allowEdit"
            @toggle="
              () => (configForOIDC.skipTlsVerify = !configForOIDC.skipTlsVerify)
            "
          />
        </p>
      </div>

      <div class="w-full flex flex-col justify-start items-start">
        <p class="text-lg font-medium mt-2">
          {{ $t("settings.sso.form.user-information-mapping") }}
        </p>
        <p class="textinfolabel">
          {{ $t("settings.sso.form.user-information-mapping-description") }}
          <a
            href="https://www.bytebase.com/docs/administration/sso/oidc#configuration?source=console"
            class="normal-link text-sm inline-flex flex-row items-center"
            target="_blank"
          >
            {{ $t("common.learn-more") }}
            <heroicons-outline:external-link class="w-4 h-4" />
          </a>
        </p>
      </div>
      <p class="textinfolabel !mt-4">
        {{ $t("settings.sso.form.user-information-mapping") }}
      </p>
      <div class="w-full grid grid-cols-[256px_1fr]">
        <input
          v-model="configForOIDC.fieldMapping!.identifier"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. preferred_username"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p class="flex flex-row justify-start items-center">
            {{ $t("settings.sso.form.identifier") }}
            <NTooltip>
              <template #trigger>
                <heroicons-outline:information-circle
                  class="ml-1 w-4 h-auto text-blue-500"
                />
              </template>
              {{ $t("settings.sso.form.identifier-tips") }}
            </NTooltip>
          </p>
        </div>
      </div>
      <div class="w-full grid grid-cols-[256px_1fr]">
        <input
          v-model="configForOIDC.fieldMapping!.displayName"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. name"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p>
            {{ $t("settings.sso.form.display-name") }}
          </p>
        </div>
      </div>
      <div class="w-full grid grid-cols-[256px_1fr]">
        <input
          v-model="configForOIDC.fieldMapping!.phone"
          :disabled="!allowEdit"
          type="text"
          class="textfield mt-1 w-full"
          placeholder="e.g. phone"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p>
            {{ $t("settings.sso.form.phone") }}
          </p>
        </div>
      </div>
    </div>

    <!-- Button group -->
    <div
      class="mt-4 space-x-4 w-full flex flex-row justify-between items-center"
    >
      <div class="space-x-4 flex flex-row justify-start items-center">
        <button
          v-if="!isDeleted"
          :disabled="!allowTestConnection"
          class="btn-normal"
          @click="testConnection"
        >
          {{ $t("identity-provider.test-connection") }}
        </button>
        <template v-if="!isCreating">
          <BBButtonConfirm
            v-if="!isDeleted"
            :style="'ARCHIVE'"
            :button-text="$t('settings.sso.archive')"
            :ok-text="$t('common.archive')"
            :confirm-title="$t('settings.sso.archive')"
            :confirm-description="$t('settings.sso.archive-info')"
            :require-confirm="true"
            @confirm="handleDeleteButtonClick"
          />
          <BBButtonConfirm
            v-else
            :style="'RESTORE'"
            :button-text="$t('settings.sso.restore')"
            :ok-text="$t('common.restore')"
            :confirm-title="$t('settings.sso.restore')"
            :confirm-description="''"
            :require-confirm="true"
            @confirm="handleRestoreButtonClick"
          />
        </template>
      </div>
      <div
        v-if="!isDeleted"
        class="space-x-4 flex flex-row justify-end items-center"
      >
        <template v-if="isCreating">
          <button class="btn-normal" @click="handleCancelButtonClick">
            {{ $t("common.cancel") }}
          </button>
          <button
            class="btn-primary"
            :disabled="!allowCreate"
            @click="handleCreateButtonClick"
          >
            {{ $t("common.create") }}
          </button>
        </template>
        <template v-else>
          <button
            class="btn-normal"
            :disabled="!allowUpdate"
            @click="handleDiscardChangesButtonClick"
          >
            {{ $t("common.discard-changes") }}
          </button>
          <button
            class="btn-primary"
            :disabled="!allowUpdate"
            @click="handleUpdateButtonClick"
          >
            {{ $t("common.update") }}
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, head, isEqual } from "lodash-es";
import { ClientError, Status } from "nice-grpc-common";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { useI18n } from "vue-i18n";
import { computed, reactive, defineProps, ref, onMounted, watch } from "vue";
import {
  FieldMapping,
  IdentityProvider,
  IdentityProviderConfig,
  IdentityProviderType,
  OAuth2AuthStyle,
  OAuth2IdentityProviderConfig,
  OIDCIdentityProviderConfig,
} from "@/types/proto/v1/idp_service";
import { useIdentityProviderStore } from "@/store/modules/idp";
import { pushNotification, useActuatorV1Store } from "@/store";
import {
  IdentityProviderTemplate,
  identityProviderTemplateList,
  identityProviderTypeToString,
  openWindowForSSO,
} from "@/utils";
import { OAuthWindowEventPayload, ResourceId, ValidatedMessage } from "@/types";
import { identityProviderClient } from "@/grpcweb";
import { State } from "@/types/proto/v1/common";
import { useRouter } from "vue-router";
import {
  getIdentityProviderResourceId,
  idpNamePrefix,
} from "@/store/modules/v1/common";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { getErrorCode } from "@/utils/grpcweb";
import { NRadioGroup, NRadio, NTooltip } from "naive-ui";

interface LocalState {
  type: IdentityProviderType;
}

const props = defineProps<{
  identityProviderName?: string;
}>();

const { t } = useI18n();
const router = useRouter();
const identityProviderStore = useIdentityProviderStore();
const state = reactive<LocalState>({
  type: IdentityProviderType.OAUTH2,
});
const identityProvider = ref<IdentityProvider>(
  IdentityProvider.fromPartial({})
);
const configForOAuth2 = ref<OAuth2IdentityProviderConfig>(
  OAuth2IdentityProviderConfig.fromPartial({
    fieldMapping: FieldMapping.fromPartial({}),
  })
);
const scopesStringOfConfig = ref<string>("");
const configForOIDC = ref<OIDCIdentityProviderConfig>(
  OIDCIdentityProviderConfig.fromPartial({
    fieldMapping: FieldMapping.fromPartial({}),
  })
);
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();
const selectedTemplate = ref<IdentityProviderTemplate>();

const currentIdentityProvider = computed(() => {
  return identityProviderStore.getIdentityProviderByName(
    props.identityProviderName || ""
  );
});

const identityProviderTypeList = computed(() => {
  return [IdentityProviderType.OAUTH2, IdentityProviderType.OIDC];
});

const redirectUrl = computed(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    return `${
      useActuatorV1Store().serverInfo?.externalUrl || window.origin
    }/oauth/callback`;
  } else if (state.type === IdentityProviderType.OIDC) {
    return `${
      useActuatorV1Store().serverInfo?.externalUrl || window.origin
    }/oidc/callback`;
  } else {
    throw new Error(`identity provider type ${state.type} is invalid`);
  }
});

const isCreating = computed(() => {
  return currentIdentityProvider.value === undefined;
});

const isDeleted = computed(() => {
  return currentIdentityProvider.value?.state === State.DELETED;
});

const resourceId = computed(() => {
  return isCreating.value
    ? ""
    : getIdentityProviderResourceId(identityProvider.value.name);
});

const userDocLink = computed(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    return "https://www.bytebase.com/docs/administration/sso/oauth2?source=console";
  } else if (state.type === IdentityProviderType.OIDC) {
    return "https://www.bytebase.com/docs/administration/sso/oidc?source=console";
  }
  return "";
});

const templateList = computed(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    return identityProviderTemplateList.filter(
      (template) => template.type === IdentityProviderType.OAUTH2
    );
  }
  return [];
});

const originIdentityProvider = computed(() => {
  return identityProviderStore.getIdentityProviderByName(
    props.identityProviderName || ""
  );
});

const isFormCompleted = computed(() => {
  if (!identityProvider.value.title) {
    return false;
  }

  if (state.type === IdentityProviderType.OAUTH2) {
    if (
      !configForOAuth2.value.clientId ||
      !configForOAuth2.value.authUrl ||
      !configForOAuth2.value.tokenUrl ||
      !configForOAuth2.value.userInfoUrl ||
      !configForOAuth2.value.fieldMapping?.identifier ||
      // Only request client secret when creating.
      (isCreating.value && !configForOAuth2.value.clientSecret)
    ) {
      return false;
    }
  } else if (state.type === IdentityProviderType.OIDC) {
    if (
      !configForOIDC.value.clientId ||
      !configForOIDC.value.issuer ||
      !configForOIDC.value.fieldMapping?.identifier ||
      (isCreating.value && !configForOIDC.value.clientSecret)
    ) {
      return false;
    }
  } else {
    return false;
  }

  return true;
});

const allowEdit = computed(() => {
  return !isDeleted.value;
});

const allowCreate = computed(() => {
  if (
    !isFormCompleted.value ||
    !resourceIdField.value?.resourceId ||
    !resourceIdField.value?.isValidated
  ) {
    return false;
  }
  return true;
});

const allowTestConnection = computed(() => {
  if (
    state.type === IdentityProviderType.OAUTH2 ||
    state.type === IdentityProviderType.OIDC
  ) {
    if (isFormCompleted.value) {
      return true;
    }
  }
  return false;
});

const editedIdentityProvider = computed(() => {
  const tempIdentityProvider: IdentityProvider = {
    ...identityProvider.value,
    type: state.type,
    config: IdentityProviderConfig.fromPartial({}),
  };
  if (tempIdentityProvider.type === IdentityProviderType.OAUTH2) {
    tempIdentityProvider.config!.oauth2Config = {
      ...configForOAuth2.value,
      scopes: scopesStringOfConfig.value.split(" "),
    };
  } else if (tempIdentityProvider.type === IdentityProviderType.OIDC) {
    tempIdentityProvider.config!.oidcConfig = configForOIDC.value;
  } else {
    throw new Error(`identity provider type ${state.type} is invalid`);
  }
  return tempIdentityProvider;
});

const allowUpdate = computed(() => {
  if (!isFormCompleted.value) {
    return false;
  }
  if (isEqual(editedIdentityProvider.value, originIdentityProvider.value)) {
    return false;
  }
  return true;
});

onMounted(async () => {
  if (originIdentityProvider.value) {
    const tempIdentityProvider = cloneDeep(originIdentityProvider.value);
    identityProvider.value = tempIdentityProvider;
    state.type = tempIdentityProvider.type;
    if (tempIdentityProvider.type === IdentityProviderType.OAUTH2) {
      configForOAuth2.value =
        tempIdentityProvider.config?.oauth2Config ||
        OAuth2IdentityProviderConfig.fromPartial({
          fieldMapping: FieldMapping.fromPartial({}),
        });
      scopesStringOfConfig.value = configForOAuth2.value.scopes.join(" ");
    } else if (tempIdentityProvider.type === IdentityProviderType.OIDC) {
      configForOIDC.value =
        tempIdentityProvider.config?.oidcConfig ||
        OIDCIdentityProviderConfig.fromPartial({
          fieldMapping: FieldMapping.fromPartial({}),
        });
    }
  }
});

const loginWithIdentityProviderEventListener = async (event: Event) => {
  const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
  if (payload.error) {
    return;
  }

  const code = payload.code;
  try {
    await identityProviderClient.testIdentityProvider({
      identityProvider: editedIdentityProvider.value,
      oauth2Context: {
        code: code,
      },
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Request error occurred`,
      description: (error as ClientError).details,
    });
    return;
  }
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Test connection succeed",
  });
};

const copyRedirectUrl = () => {
  toClipboard(redirectUrl.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t("settings.sso.copy-redirect-url"),
    });
  });
};

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  try {
    const idp = await identityProviderStore.getOrFetchIdentityProviderByName(
      idpNamePrefix + resourceId,
      true /* silent */
    );
    if (idp) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t("resource.idp"),
          }),
        },
      ];
    }
  } catch (error) {
    if (getErrorCode(error) !== Status.NOT_FOUND) {
      throw error;
    }
  }
  return [];
};

const testConnection = () => {
  if (
    state.type === IdentityProviderType.OAUTH2 ||
    state.type === IdentityProviderType.OIDC
  ) {
    openWindowForSSO(editedIdentityProvider.value);
  }
};

const handleTemplateSelect = (template: IdentityProviderTemplate) => {
  if (template.type !== state.type) {
    return;
  }

  selectedTemplate.value = template;
  identityProvider.value.title = template.title;
  identityProvider.value.domain = template.domain;
  if (template.type === IdentityProviderType.OAUTH2) {
    configForOAuth2.value = {
      ...configForOAuth2.value,
      ...template.config,
    };
    scopesStringOfConfig.value = template.config.scopes.join(" ");
  }
};

const handleDeleteButtonClick = async () => {
  if (currentIdentityProvider.value) {
    await identityProviderStore.deleteIdentityProvider(
      currentIdentityProvider.value.name
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: "Archive SSO succeed",
    });
    router.push({
      name: "setting.workspace.sso",
    });
  }
};

const handleRestoreButtonClick = async () => {
  if (currentIdentityProvider.value) {
    await identityProviderStore.undeleteIdentityProvider(
      currentIdentityProvider.value.name
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: "Restore SSO succeed",
    });
  }
};

const handleCancelButtonClick = () => {
  router.push({
    name: "setting.workspace.sso",
  });
};

const updateEditState = (updatedIdentityProvider: IdentityProvider) => {
  const tempIdentityProvider = cloneDeep(updatedIdentityProvider);
  identityProvider.value = tempIdentityProvider;
  state.type = tempIdentityProvider.type;
  if (tempIdentityProvider.type === IdentityProviderType.OAUTH2) {
    configForOAuth2.value =
      tempIdentityProvider.config?.oauth2Config ||
      OAuth2IdentityProviderConfig.fromPartial({
        fieldMapping: FieldMapping.fromPartial({}),
      });
    scopesStringOfConfig.value = configForOAuth2.value.scopes.join(" ");
  } else if (tempIdentityProvider.type === IdentityProviderType.OIDC) {
    configForOIDC.value =
      tempIdentityProvider.config?.oidcConfig ||
      OIDCIdentityProviderConfig.fromPartial({
        fieldMapping: FieldMapping.fromPartial({}),
      });
  }
};

const handleDiscardChangesButtonClick = async () => {
  if (originIdentityProvider.value) {
    updateEditState(originIdentityProvider.value);
  }
};

const handleCreateButtonClick = async () => {
  const identityProviderCreate: IdentityProvider = {
    ...identityProvider.value,
    name: resourceIdField.value?.resourceId as string,
    type: state.type,
    config: {},
  };
  if (state.type === IdentityProviderType.OAUTH2) {
    configForOAuth2.value.scopes = scopesStringOfConfig.value.split(" ");
    identityProviderCreate.config!.oauth2Config = configForOAuth2.value;
  } else if (state.type === IdentityProviderType.OIDC) {
    identityProviderCreate.config!.oidcConfig = configForOIDC.value;
  } else {
    throw new Error(`identity provider type ${state.type} is invalid`);
  }

  await identityProviderStore.createIdentityProvider(identityProviderCreate);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Create SSO succeed",
  });
  router.push({
    name: "setting.workspace.sso",
  });
};

const handleUpdateButtonClick = async () => {
  const updatedIdentityProvider =
    await identityProviderStore.patchIdentityProvider(
      editedIdentityProvider.value
    );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Update SSO succeed",
  });
  updateEditState(updatedIdentityProvider);
};

watch(
  () => state.type,
  () => {
    if (!isCreating.value) {
      return;
    }
    if (state.type === IdentityProviderType.OAUTH2) {
      if (!selectedTemplate.value) {
        selectedTemplate.value = head(templateList.value);
      }
      if (selectedTemplate.value) {
        handleTemplateSelect(
          selectedTemplate.value as IdentityProviderTemplate
        );
      }
    } else if (state.type === IdentityProviderType.OIDC) {
      // NOTE: We do not yet have templates for OIDC providers so resetting
      // leftover values when we switch from other provider types to not confuse users.
      identityProvider.value.title = "";
      identityProvider.value.name = "";
      identityProvider.value.domain = "";
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => editedIdentityProvider.value.name,
  (newName, oldName) => {
    if (
      state.type === IdentityProviderType.OAUTH2 ||
      state.type === IdentityProviderType.OIDC
    ) {
      window.removeEventListener(
        `bb.oauth.signin.${oldName}`,
        loginWithIdentityProviderEventListener,
        false
      );
      window.addEventListener(
        `bb.oauth.signin.${newName}`,
        loginWithIdentityProviderEventListener,
        false
      );
    }
  },
  {
    immediate: true,
  }
);
</script>
