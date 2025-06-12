<template>
  <div class="w-full space-y-0 pt-4">
    <div class="divide-y divide-block-border">
      <!-- Provider Type Section -->
      <div v-if="isCreating" class="pb-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold">
            {{ $t("settings.sso.form.type") }}
          </h1>
          <a
            v-if="userDocLink"
            :href="userDocLink"
            class="normal-link text-sm inline-flex flex-row items-center mt-2"
            target="_blank"
          >
            {{ $t("settings.sso.form.learn-more-with-user-doc") }}
            <heroicons-outline:external-link class="w-4 h-4 ml-1" />
          </a>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <div class="space-y-3">
            <NRadioGroup
              v-model:value="state.type"
              size="large"
              class="space-y-3"
            >
              <NRadio
                v-for="item in identityProviderTypeList"
                :key="item.type"
                :value="item.type"
                :disabled="!subscriptionStore.hasFeature(item.feature)"
              >
                <div class="flex items-center space-x-2">
                  <FeatureBadge :feature="item.feature" />
                  {{ identityProviderTypeToString(item.type) }}
                </div>
              </NRadio>
            </NRadioGroup>

            <BBAttention
              v-if="
                !externalUrl &&
                (state.type === IdentityProviderType.OAUTH2 ||
                  state.type === IdentityProviderType.OIDC)
              "
              class="mt-4 border-none"
              type="error"
              :title="$t('banner.external-url')"
              :description="
                $t('settings.general.workspace.external-url.description')
              "
            >
              <template #action>
                <NButton type="primary" @click="configureSetting">
                  {{ $t("common.configure-now") }}
                </NButton>
              </template>
            </BBAttention>
          </div>
        </div>
      </div>

      <!-- OAuth2 Templates Section -->
      <div
        v-if="isCreating && state.type === IdentityProviderType.OAUTH2"
        class="py-6 lg:flex"
      >
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold">
            {{ $t("settings.sso.form.use-template") }}
          </h1>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <NRadioGroup
            :value="selectedTemplate?.title"
            size="large"
            class="space-y-3"
          >
            <NRadio
              v-for="template in templateList"
              :key="template.title"
              :value="template.title"
              :disabled="!subscriptionStore.hasFeature(template.feature)"
              @change="handleTemplateSelect(template)"
            >
              <div class="flex items-center space-x-2">
                <FeatureBadge :feature="template.feature" />
                {{ template.title }}
              </div>
            </NRadio>
          </NRadioGroup>
        </div>
      </div>

      <!-- Basic Information Section -->
      <div class="py-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold">
            {{ $t("settings.sso.form.basic-information") }}
          </h1>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0 space-y-4">
          <div>
            <p class="text-base font-semibold text-gray-800">
              {{ $t("settings.sso.form.name") }}
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="identityProvider.title"
              required
              size="large"
              :disabled="!allowEdit"
              class="mt-1 w-full"
              :placeholder="$t('settings.sso.form.name-description')"
            />
            <ResourceIdField
              ref="resourceIdField"
              class="mt-1.5"
              editing-class="mt-3"
              resource-type="idp"
              :readonly="!isCreating"
              :value="resourceId"
              :suffix="true"
              :resource-title="identityProvider.title"
              :fetch-resource="
                (id) =>
                  identityProviderStore.getOrFetchIdentityProviderByName(
                    `${idpNamePrefix}${id}`,
                    true /* silent */
                  )
              "
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              {{ $t("settings.sso.form.domain") }}
            </p>
            <BBTextField
              size="large"
              v-model:value="identityProvider.domain"
              :disabled="!allowEdit"
              class="mt-1 w-full"
              :placeholder="$t('settings.sso.form.domain-description')"
            />
          </div>
        </div>
      </div>

      <!-- OAuth2 Configuration Section -->
      <div
        v-if="state.type === IdentityProviderType.OAUTH2"
        class="py-6 lg:flex"
      >
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold text-gray-900">
            {{ $t("settings.sso.form.identity-provider-information") }}
          </h1>
          <p class="textinfolabel mt-2">
            {{
              $t("settings.sso.form.identity-provider-information-description")
            }}
          </p>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0 space-y-4">
          <IdentityProviderExternalURL v-if="isCreating" :type="state.type" />
          <div>
            <p class="text-base font-semibold text-gray-800">
              Client ID
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForOAuth2.clientId"
              required
              size="large"
              :disabled="!allowEdit"
              class="mt-1 w-full"
              placeholder="e.g. 6655asd77895265aa110ac0d3"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Client secret
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForOAuth2.clientSecret"
              :required="isCreating"
              size="large"
              :disabled="!allowEdit"
              class="mt-1 w-full"
              :placeholder="
                isCreating
                  ? 'e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8'
                  : $t('common.sensitive-placeholder')
              "
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Auth URL
              <span class="text-red-600">*</span>
            </p>
            <p class="textinfolabel">
              {{ $t("settings.sso.form.auth-url-description") }}
            </p>
            <BBTextField
              v-model:value="configForOAuth2.authUrl"
              required
              size="large"
              :disabled="!allowEdit"
              class="mt-1 w-full"
              placeholder="e.g. https://github.com/login/oauth/authorize"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Scopes
              <span class="text-red-600">*</span>
            </p>
            <p class="textinfolabel">
              {{ $t("settings.sso.form.scopes-description") }}
            </p>
            <BBTextField
              v-model:value="scopesStringOfConfig"
              required
              size="large"
              :disabled="!allowEdit"
              class="mt-1 w-full"
              placeholder="e.g. user"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Token URL
              <span class="text-red-600">*</span>
            </p>
            <p class="textinfolabel">
              {{ $t("settings.sso.form.token-url-description") }}
            </p>
            <BBTextField
              v-model:value="configForOAuth2.tokenUrl"
              required
              size="large"
              :disabled="!allowEdit"
              class="mt-1 w-full"
              placeholder="e.g. https://github.com/login/oauth/access_token"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              User information URL
              <span class="text-red-600">*</span>
            </p>
            <p class="textinfolabel">
              {{ $t("settings.sso.form.user-info-url-description") }}
            </p>
            <BBTextField
              size="large"
              v-model:value="configForOAuth2.userInfoUrl"
              required
              :disabled="!allowEdit"
              class="mt-1 w-full"
              placeholder="e.g. https://api.github.com/user"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              {{ $t("settings.sso.form.auth-style.self") }}
              <span class="text-red-600">*</span>
            </p>
            <div class="mt-1">
              <NRadioGroup
                size="large"
                v-model:value="configForOAuth2.authStyle"
              >
                <NRadio
                  :value="OAuth2AuthStyle.IN_PARAMS"
                  :label="$t('settings.sso.form.auth-style.in-params.self')"
                />
                <NRadio
                  :value="OAuth2AuthStyle.IN_HEADER"
                  :label="$t('settings.sso.form.auth-style.in-header.self')"
                />
              </NRadioGroup>
            </div>
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              {{ $t("settings.sso.form.connection-security") }}
            </p>
            <div class="mt-1">
              <NCheckbox
                size="large"
                v-model:checked="configForOAuth2.skipTlsVerify"
                :label="
                  $t('settings.sso.form.connection-security-skip-tls-verify')
                "
                :disabled="!allowEdit"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- OIDC Configuration Section -->
      <div
        v-else-if="state.type === IdentityProviderType.OIDC"
        class="py-6 lg:flex"
      >
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold text-gray-900">
            {{ $t("settings.sso.form.identity-provider-information") }}
          </h1>
          <p class="text-base text-gray-600 mt-3">
            {{
              $t("settings.sso.form.identity-provider-information-description")
            }}
          </p>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0 space-y-6">
          <IdentityProviderExternalURL v-if="isCreating" :type="state.type" />
          <div>
            <p class="text-base font-semibold text-gray-800">
              Issuer
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForOIDC.issuer"
              required
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              placeholder="e.g. https://acme.okta.com"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Client ID
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForOIDC.clientId"
              required
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              placeholder="e.g. 6655asd77895265aa110ac0d3"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Client secret
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForOIDC.clientSecret"
              required
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              :placeholder="
                isCreating
                  ? 'e.g. 5bbezxc3972ca304de70c5d70a6aa932asd8'
                  : $t('common.sensitive-placeholder')
              "
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Scopes
              <span class="text-red-600">*</span>
            </p>
            <p class="text-sm text-gray-600">
              {{ $t("settings.sso.form.scopes-description") }}
            </p>
            <BBTextField
              v-model:value="scopesStringOfConfig"
              required
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              placeholder="e.g. user"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              {{ $t("settings.sso.form.auth-style.self") }}
              <span class="text-red-600">*</span>
            </p>
            <NRadioGroup
              v-model:value="configForOIDC.authStyle"
              size="large"
              class="space-y-3"
            >
              <NRadio
                :value="OAuth2AuthStyle.IN_PARAMS"
                :label="$t('settings.sso.form.auth-style.in-params.self')"
                class="text-base font-medium text-gray-800"
              />
              <NRadio
                :value="OAuth2AuthStyle.IN_HEADER"
                :label="$t('settings.sso.form.auth-style.in-header.self')"
                class="text-base font-medium text-gray-800"
              />
            </NRadioGroup>
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800 mb-3">
              {{ $t("settings.sso.form.connection-security") }}
            </p>
            <div class="mt-2">
              <NCheckbox
                v-model:checked="configForOIDC.skipTlsVerify"
                :label="
                  $t('settings.sso.form.connection-security-skip-tls-verify')
                "
                :disabled="!allowEdit"
                size="large"
                class="text-base font-medium text-gray-800"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- LDAP Configuration Section -->
      <div
        v-else-if="state.type === IdentityProviderType.LDAP"
        class="py-6 lg:flex"
      >
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold text-gray-900">
            {{ $t("settings.sso.form.identity-provider-information") }}
          </h1>
          <p class="text-base text-gray-600 mt-3">
            {{
              $t("settings.sso.form.identity-provider-information-description")
            }}
          </p>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0 space-y-6">
          <div>
            <p class="text-base font-semibold text-gray-800">
              Host
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForLDAP.host"
              required
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              placeholder="e.g. ldap.example.com"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Port
              <span class="text-red-600">*</span>
            </p>
            <NInputNumber
              v-model:value="configForLDAP.port"
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              placeholder="e.g. 389 or 636"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Bind DN
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForLDAP.bindDn"
              required
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              placeholder="e.g. uid=system,ou=Users,dc=example,dc=com"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Bind Password
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForLDAP.bindPassword"
              required
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              :placeholder="$t('common.sensitive-placeholder')"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              Base DN
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForLDAP.baseDn"
              required
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              placeholder="e.g. ou=users,dc=example,dc=com"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              User Filter
              <span class="text-red-600">*</span>
            </p>
            <BBTextField
              v-model:value="configForLDAP.userFilter"
              required
              :disabled="!allowEdit"
              class="mt-2 w-full text-base"
              size="large"
              placeholder="e.g. (uid=%s)"
            />
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800">
              {{ $t("settings.sso.form.security-protocol") }}
              <span class="text-red-600">*</span>
            </p>
            <NRadioGroup
              v-model:value="configForLDAP.securityProtocol"
              size="large"
              class="space-y-3"
            >
              <NRadio
                value="starttls"
                label="StartTLS"
                class="text-base font-medium text-gray-800"
              />
              <NRadio
                value="ldaps"
                label="LDAPS"
                class="text-base font-medium text-gray-800"
              />
              <NRadio
                value=""
                label="None"
                class="text-base font-medium text-gray-800"
              />
            </NRadioGroup>
          </div>
          <div>
            <p class="text-base font-semibold text-gray-800 mb-3">
              {{ $t("settings.sso.form.connection-security") }}
            </p>
            <NCheckbox
              v-model:checked="configForLDAP.skipTlsVerify"
              :label="
                $t('settings.sso.form.connection-security-skip-tls-verify')
              "
              :disabled="!allowEdit"
              size="large"
              class="text-base font-medium text-gray-800"
            />
          </div>
        </div>
      </div>

      <!-- User Information Mapping Section -->
      <div class="py-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-3xl font-bold">
            {{ $t("settings.sso.form.user-information-mapping") }}
          </h1>
          <p class="textinfolabel mt-2">
            {{ $t("settings.sso.form.user-information-mapping-description") }}
            <a
              href="https://docs.bytebase.com/administration/sso/oauth2#user-information-field-mapping?source=console"
              class="normal-link text-sm inline-flex flex-row items-center"
              target="_blank"
            >
              {{ $t("common.learn-more") }}
              <heroicons-outline:external-link class="w-4 h-4 ml-1" />
            </a>
          </p>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0 space-y-4">
          <div class="grid grid-cols-[256px_1fr] gap-4 items-center">
            <BBTextField
              v-model:value="state.fieldMapping.identifier"
              :disabled="!allowEdit"
              size="large"
              class="w-full"
              placeholder="e.g. login"
            />
            <div class="flex flex-row justify-start items-center">
              <ArrowRightIcon class="mx-2 h-auto w-4 text-gray-300" />
              <p class="flex flex-row justify-start items-center">
                {{ $t("settings.sso.form.identifier") }}
                <span class="text-red-600">*</span>
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
          <div class="grid grid-cols-[256px_1fr] gap-4 items-center">
            <BBTextField
              v-model:value="state.fieldMapping.displayName"
              :disabled="!allowEdit"
              size="large"
              class="w-full"
              placeholder="e.g. name"
            />
            <div class="flex flex-row justify-start items-center">
              <ArrowRightIcon class="mx-2 h-auto w-4 text-gray-300" />
              <p>
                {{ $t("settings.sso.form.display-name") }}
              </p>
            </div>
          </div>
          <div class="grid grid-cols-[256px_1fr] gap-4 items-center">
            <BBTextField
              v-model:value="state.fieldMapping.phone"
              :disabled="!allowEdit"
              size="large"
              class="w-full"
              placeholder="e.g. phone"
            />
            <div class="flex flex-row justify-start items-center">
              <ArrowRightIcon class="mx-2 h-auto w-4 text-gray-300" />
              <p>
                {{ $t("settings.sso.form.phone") }}
              </p>
            </div>
          </div>
          <div
            v-if="state.type === IdentityProviderType.OIDC"
            class="grid grid-cols-[256px_1fr] gap-4 items-center"
          >
            <BBTextField
              v-model:value="state.fieldMapping.groups"
              :disabled="!allowEdit"
              size="large"
              class="w-full"
              placeholder="e.g. groups"
            />
            <div class="flex flex-row justify-start items-center">
              <ArrowRightIcon class="mx-2 h-auto w-4 text-gray-300" />
              <p>
                {{ $t("settings.sso.form.groups") }}
              </p>
            </div>
          </div>
          <div>
            <NButton :disabled="!allowTestConnection" @click="testConnection">
              {{ $t("identity-provider.test-connection") }}
            </NButton>
          </div>
        </div>
      </div>

      <!-- Actions Section -->
      <div class="py-6">
        <div class="flex flex-row justify-between items-center">
          <div class="space-x-4 flex flex-row justify-start items-center">
            <template v-if="!isCreating">
              <BBButtonConfirm
                :type="'ARCHIVE'"
                :button-text="$t('settings.sso.delete')"
                :ok-text="$t('common.delete')"
                :confirm-title="$t('settings.sso.delete')"
                :confirm-description="$t('common.cannot-undo-this-action')"
                :require-confirm="true"
                @confirm="handleDeleteButtonClick"
              />
            </template>
          </div>
          <div class="space-x-3 flex flex-row justify-end items-center">
            <template v-if="isCreating">
              <NButton
                @click="handleCancelButtonClick"
                size="large"
                class="text-base"
              >
                {{ $t("common.cancel") }}
              </NButton>
              <NButton
                type="primary"
                :disabled="!allowCreate"
                @click="handleCreateButtonClick"
                size="large"
                class="text-base"
              >
                {{ $t("common.create") }}
              </NButton>
            </template>
            <template v-else>
              <NButton
                v-if="allowUpdate"
                @click="handleDiscardChangesButtonClick"
                size="large"
                class="text-base"
              >
                {{ $t("common.discard-changes") }}
              </NButton>
              <NButton
                type="primary"
                :disabled="!allowUpdate"
                @click="handleUpdateButtonClick"
                size="large"
                class="text-base"
              >
                {{ $t("common.update") }}
              </NButton>
            </template>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, head, isEqual } from "lodash-es";
import { ArrowRightIcon } from "lucide-vue-next";
import {
  NRadioGroup,
  NCheckbox,
  NRadio,
  NTooltip,
  NInputNumber,
  NButton,
} from "naive-ui";
import type { ClientError } from "nice-grpc-common";
import { computed, reactive, ref, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import { BBAttention, BBButtonConfirm, BBTextField } from "@/bbkit";
import { FeatureBadge } from "@/components/FeatureGuard";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { identityProviderClient } from "@/grpcweb";
import { WORKSPACE_ROUTE_SSO } from "@/router/dashboard/workspaceRoutes";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import { SQL_EDITOR_SETTING_GENERAL_MODULE } from "@/router/sqlEditor";
import {
  pushNotification,
  useActuatorV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import {
  getIdentityProviderResourceId,
  idpNamePrefix,
} from "@/store/modules/v1/common";
import type { OAuthWindowEventPayload } from "@/types";
import {
  FieldMapping,
  IdentityProvider,
  IdentityProviderConfig,
  IdentityProviderType,
  OAuth2AuthStyle,
  OAuth2IdentityProviderConfig,
  OIDCIdentityProviderConfig,
  LDAPIdentityProviderConfig,
} from "@/types/proto/v1/idp_service";
import { PlanFeature } from "@/types/proto/v1/subscription_service";
import type { OAuth2IdentityProviderTemplate } from "@/utils";
import {
  hasWorkspacePermissionV2,
  identityProviderTypeToString,
  openWindowForSSO,
} from "@/utils";
import IdentityProviderExternalURL from "./IdentityProviderExternalURL.vue";

interface IdentityProviderTemplate extends OAuth2IdentityProviderTemplate {
  feature: PlanFeature;
}

interface LocalState {
  type: IdentityProviderType;
  fieldMapping: FieldMapping;
}

const props = defineProps<{
  identityProviderName?: string;
  onCreated?: (sso: IdentityProvider) => void;
  onUpdated?: (sso: IdentityProvider) => void;
  onDeleted?: (sso: IdentityProvider) => void;
  onCanceled?: () => void;
}>();

const router = useRouter();
const identityProviderStore = useIdentityProviderStore();
const subscriptionStore = useSubscriptionV1Store();

const state = reactive<LocalState>({
  type: IdentityProviderType.OAUTH2,
  fieldMapping: FieldMapping.fromPartial({}),
});

const identityProvider = ref<IdentityProvider>(
  IdentityProvider.fromPartial({})
);
const configForOAuth2 = ref<OAuth2IdentityProviderConfig>(
  OAuth2IdentityProviderConfig.fromPartial({})
);
const scopesStringOfConfig = ref<string>("");
const configForOIDC = ref<OIDCIdentityProviderConfig>(
  OIDCIdentityProviderConfig.fromPartial({})
);
const configForLDAP = ref<LDAPIdentityProviderConfig>(
  LDAPIdentityProviderConfig.fromPartial({
    port: 389,
    securityProtocol: "starttls",
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
  return [
    {
      type: IdentityProviderType.OAUTH2,
      feature: PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO,
    },
    {
      type: IdentityProviderType.OIDC,
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
    },
    {
      type: IdentityProviderType.LDAP,
      feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
    },
  ];
});

const configureSetting = () => {
  router.push({
    name: router.currentRoute.value.name?.toString().startsWith("sql-editor")
      ? SQL_EDITOR_SETTING_GENERAL_MODULE
      : SETTING_ROUTE_WORKSPACE_GENERAL,
  });
};

const externalUrl = computed(
  () => useActuatorV1Store().serverInfo?.externalUrl ?? ""
);

const isCreating = computed(() => {
  return currentIdentityProvider.value === undefined;
});

const resourceId = computed(() => {
  return isCreating.value
    ? ""
    : getIdentityProviderResourceId(identityProvider.value.name);
});

const userDocLink = computed(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    return "https://docs.bytebase.com/administration/sso/oauth2?source=console";
  } else if (state.type === IdentityProviderType.OIDC) {
    return "https://docs.bytebase.com/administration/sso/oidc?source=console";
  } else if (state.type === IdentityProviderType.LDAP) {
    return "https://docs.bytebase.com/administration/sso/ldap?source=console";
  }
  return "";
});

const identityProviderTemplateList = computed(
  (): IdentityProviderTemplate[] => {
    return [
      {
        title: "Google",
        name: "",
        domain: "google.com",
        type: IdentityProviderType.OAUTH2,
        feature: PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO,
        config: {
          clientId: "",
          clientSecret: "",
          authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
          tokenUrl: "https://oauth2.googleapis.com/token",
          userInfoUrl: "https://www.googleapis.com/oauth2/v2/userinfo",
          scopes: [
            "https://www.googleapis.com/auth/userinfo.email",
            "https://www.googleapis.com/auth/userinfo.profile",
          ],
          skipTlsVerify: false,
          authStyle: OAuth2AuthStyle.IN_PARAMS,
          fieldMapping: {
            identifier: "email",
            displayName: "name",
            phone: "",
            groups: "",
          },
        },
      },
      {
        title: "GitHub",
        name: "",
        domain: "github.com",
        type: IdentityProviderType.OAUTH2,
        feature: PlanFeature.FEATURE_GOOGLE_AND_GITHUB_SSO,
        config: {
          clientId: "",
          clientSecret: "",
          authUrl: "https://github.com/login/oauth/authorize",
          tokenUrl: "https://github.com/login/oauth/access_token",
          userInfoUrl: "https://api.github.com/user",
          scopes: ["user"],
          skipTlsVerify: false,
          authStyle: OAuth2AuthStyle.IN_PARAMS,
          fieldMapping: {
            identifier: "email",
            displayName: "name",
            phone: "",
            groups: "",
          },
        },
      },
      {
        title: "GitLab",
        name: "",
        domain: "gitlab.com",
        type: IdentityProviderType.OAUTH2,
        feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
        config: {
          clientId: "",
          clientSecret: "",
          authUrl: "https://gitlab.com/oauth/authorize",
          tokenUrl: "https://gitlab.com/oauth/token",
          userInfoUrl: "https://gitlab.com/api/v4/user",
          scopes: ["read_user"],
          skipTlsVerify: false,
          authStyle: OAuth2AuthStyle.IN_PARAMS,
          fieldMapping: {
            identifier: "email",
            displayName: "name",
            phone: "",
            groups: "",
          },
        },
      },
      {
        title: "Microsoft Entra",
        name: "",
        domain: "",
        type: IdentityProviderType.OAUTH2,
        feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
        config: {
          clientId: "",
          clientSecret: "",
          authUrl:
            "https://login.microsoftonline.com/{uuid}/oauth2/v2.0/authorize",
          tokenUrl:
            "https://login.microsoftonline.com/{uuid}/oauth2/v2.0/token",
          userInfoUrl: "https://graph.microsoft.com/v1.0/me",
          scopes: ["user.read"],
          skipTlsVerify: false,
          authStyle: OAuth2AuthStyle.IN_PARAMS,
          fieldMapping: {
            identifier: "userPrincipalName",
            displayName: "displayName",
            phone: "",
            groups: "",
          },
        },
      },
      {
        title: "Custom",
        name: "",
        domain: "",
        type: IdentityProviderType.OAUTH2,
        feature: PlanFeature.FEATURE_ENTERPRISE_SSO,
        config: {
          clientId: "",
          clientSecret: "",
          authUrl: "",
          tokenUrl: "",
          userInfoUrl: "",
          scopes: [],
          skipTlsVerify: false,
          authStyle: OAuth2AuthStyle.IN_PARAMS,
          fieldMapping: {
            identifier: "",
            displayName: "",
            phone: "",
            groups: "",
          },
        },
      },
    ];
  }
);

const templateList = computed(() => {
  if (state.type === IdentityProviderType.OAUTH2) {
    return identityProviderTemplateList.value.filter(
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
  if (!state.fieldMapping.identifier) {
    return false;
  }

  if (state.type === IdentityProviderType.OAUTH2) {
    if (
      !configForOAuth2.value.clientId ||
      !configForOAuth2.value.authUrl ||
      !configForOAuth2.value.tokenUrl ||
      !configForOAuth2.value.userInfoUrl ||
      // Only request client secret when creating.
      (isCreating.value && !configForOAuth2.value.clientSecret)
    ) {
      return false;
    }
  } else if (state.type === IdentityProviderType.OIDC) {
    if (
      !configForOIDC.value.clientId ||
      !configForOIDC.value.issuer ||
      (isCreating.value && !configForOIDC.value.clientSecret)
    ) {
      return false;
    }
  } else if (state.type === IdentityProviderType.LDAP) {
    if (
      !configForLDAP.value.host ||
      !configForLDAP.value.port ||
      !configForLDAP.value.bindDn ||
      !configForLDAP.value.baseDn ||
      !configForLDAP.value.userFilter ||
      (isCreating.value && !configForLDAP.value.bindPassword)
    ) {
      return false;
    }
  } else {
    return false;
  }

  return true;
});

const allowEdit = computed(() => {
  if (isCreating.value) {
    return hasWorkspacePermissionV2("bb.identityProviders.create");
  }
  return hasWorkspacePermissionV2("bb.identityProviders.update");
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
    state.type === IdentityProviderType.OIDC ||
    state.type === IdentityProviderType.LDAP
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
    name:
      identityProvider.value.name ||
      (resourceIdField.value?.resourceId as string),
    config: IdentityProviderConfig.fromPartial({}),
  };
  if (tempIdentityProvider.type === IdentityProviderType.OAUTH2) {
    tempIdentityProvider.config!.oauth2Config = {
      ...configForOAuth2.value,
      scopes: scopesStringOfConfig.value.split(" "),
      fieldMapping: FieldMapping.fromPartial(state.fieldMapping),
    };
  } else if (tempIdentityProvider.type === IdentityProviderType.OIDC) {
    tempIdentityProvider.config!.oidcConfig = {
      ...configForOIDC.value,
      scopes: scopesStringOfConfig.value.split(" "),
      fieldMapping: FieldMapping.fromPartial(state.fieldMapping),
    };
  } else if (tempIdentityProvider.type === IdentityProviderType.LDAP) {
    tempIdentityProvider.config!.ldapConfig = {
      ...configForLDAP.value,
      fieldMapping: FieldMapping.fromPartial(state.fieldMapping),
    };
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
    updateEditState(originIdentityProvider.value);
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

const testConnection = async () => {
  if (
    state.type === IdentityProviderType.OAUTH2 ||
    state.type === IdentityProviderType.OIDC
  ) {
    let idpForTesting = editedIdentityProvider.value;
    // For OIDC, we need to obtain the auth endpoint from the issuer in backend.
    if (isCreating.value && idpForTesting.type === IdentityProviderType.OIDC) {
      idpForTesting = await identityProviderClient.createIdentityProvider({
        identityProviderId: idpForTesting.name,
        identityProvider: idpForTesting,
        validateOnly: true,
      });
    }
    try {
      await openWindowForSSO(idpForTesting);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `Request error occurred`,
        description: (error as any).message,
      });
    }
  } else if (state.type === IdentityProviderType.LDAP) {
    try {
      await identityProviderClient.testIdentityProvider({
        identityProvider: editedIdentityProvider.value,
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
    state.fieldMapping = FieldMapping.fromPartial(
      template.config.fieldMapping || {}
    );
    scopesStringOfConfig.value = template.config.scopes.join(" ");
  }
};

const handleDeleteButtonClick = async () => {
  if (!currentIdentityProvider.value) {
    return;
  }
  if (!hasWorkspacePermissionV2("bb.identityProviders.delete")) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Permission denied",
    });
    return;
  }

  await identityProviderStore.deleteIdentityProvider(
    currentIdentityProvider.value.name
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Archive SSO succeed",
  });
  if (props.onDeleted) {
    props.onDeleted(currentIdentityProvider.value);
    return;
  }
  router.push({
    name: WORKSPACE_ROUTE_SSO,
  });
};

const handleCancelButtonClick = () => {
  if (props.onCanceled) {
    props.onCanceled();
    return;
  }
  router.push({
    name: WORKSPACE_ROUTE_SSO,
  });
};

const updateEditState = (updatedIdentityProvider: IdentityProvider) => {
  const tempIdentityProvider = cloneDeep(updatedIdentityProvider);
  identityProvider.value = tempIdentityProvider;
  state.type = tempIdentityProvider.type;
  if (tempIdentityProvider.type === IdentityProviderType.OAUTH2) {
    state.fieldMapping =
      tempIdentityProvider.config?.oauth2Config?.fieldMapping ??
      FieldMapping.fromPartial({});
    configForOAuth2.value =
      tempIdentityProvider.config?.oauth2Config ||
      OAuth2IdentityProviderConfig.fromPartial({});
    scopesStringOfConfig.value = configForOAuth2.value.scopes.join(" ");
  } else if (tempIdentityProvider.type === IdentityProviderType.OIDC) {
    state.fieldMapping =
      tempIdentityProvider.config?.oidcConfig?.fieldMapping ??
      FieldMapping.fromPartial({});
    configForOIDC.value =
      tempIdentityProvider.config?.oidcConfig ||
      OIDCIdentityProviderConfig.fromPartial({});
    scopesStringOfConfig.value = configForOIDC.value.scopes.join(" ");
  } else if (tempIdentityProvider.type === IdentityProviderType.LDAP) {
    state.fieldMapping =
      tempIdentityProvider.config?.ldapConfig?.fieldMapping ??
      FieldMapping.fromPartial({});
    configForLDAP.value =
      tempIdentityProvider.config?.ldapConfig ||
      LDAPIdentityProviderConfig.fromPartial({});
  }
};

const handleDiscardChangesButtonClick = async () => {
  if (originIdentityProvider.value) {
    updateEditState(originIdentityProvider.value);
  }
};

const handleCreateButtonClick = async () => {
  const created = await identityProviderStore.createIdentityProvider(
    editedIdentityProvider.value
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Create SSO succeed",
  });
  if (props.onCreated) {
    props.onCreated(created);
    return;
  }
  router.push({
    name: WORKSPACE_ROUTE_SSO,
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
  if (props.onUpdated) {
    props.onUpdated(updatedIdentityProvider);
  }
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
    } else if (
      state.type === IdentityProviderType.OIDC ||
      state.type === IdentityProviderType.LDAP
    ) {
      // NOTE: We do not yet have templates for OIDC nor LDAP providers so resetting
      // leftover values when we switch from other provider types to not confuse users.
      identityProvider.value.title = "";
      identityProvider.value.name = "";
      identityProvider.value.domain = "";

      if (state.type === IdentityProviderType.OIDC) {
        // Default is a list of scopes that are part of OIDC standard claims, see
        // https://auth0.com/docs/get-started/apis/scopes/openid-connect-scopes#standard-claims.
        scopesStringOfConfig.value = "openid profile email";
      }
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
