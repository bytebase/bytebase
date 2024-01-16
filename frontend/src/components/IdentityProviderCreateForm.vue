<template>
  <div
    class="w-full flex flex-col justify-start items-start overflow-x-hidden px-1"
  >
    <div
      v-if="isCreating"
      class="w-full flex flex-col justify-start items-start"
    >
      <div class="w-full flex flex-row items-center space-x-2">
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
        <NRadioGroup
          v-for="item in identityProviderTypeList"
          :key="item"
          v-model:value="state.type"
        >
          <NRadio :value="item">
            {{ identityProviderTypeToString(item) }}
          </NRadio>
        </NRadioGroup>
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
        <NRadioGroup
          v-for="template in templateList"
          :key="template.title"
          :value="selectedTemplate?.title"
        >
          <div
            class="w-24 h-12 border rounded-md flex items-center justify-center"
          >
            <NRadio
              :value="template.title"
              @change="handleTemplateSelect(template)"
            >
              {{ template.title }}
            </NRadio>
          </div>
        </NRadioGroup>
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
        <BBTextField
          v-model:value="identityProvider.title"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <BBTextField
          v-model:value="identityProvider.domain"
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <BBTextField
          v-model:value="configForOAuth2.clientId"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. 6655asd77895265aa110ac0d3"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Client secret
          <span class="text-red-600">*</span>
        </p>
        <BBTextField
          v-model:value="configForOAuth2.clientSecret"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <BBTextField
          v-model:value="configForOAuth2.authUrl"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <BBTextField
          v-model:value="scopesStringOfConfig"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <BBTextField
          v-model:value="configForOAuth2.tokenUrl"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <BBTextField
          v-model:value="configForOAuth2.userInfoUrl"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
          <NCheckbox
            v-model:checked="configForOAuth2.skipTlsVerify"
            :label="$t('settings.sso.form.connection-security-skip-tls-verify')"
            :disabled="!allowEdit"
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
        <BBTextField
          v-model:value="configForOAuth2.fieldMapping!.identifier"
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. login"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
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
      <div class="w-full grid grid-cols-[256px_1fr]">
        <BBTextField
          v-model:value="configForOAuth2.fieldMapping!.displayName"
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <BBTextField
          v-model:value="configForOAuth2.fieldMapping!.phone"
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <BBTextField
          v-model:value="configForOIDC.issuer"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. https://acme.okta.com"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Client ID
          <span class="text-red-600">*</span>
        </p>
        <BBTextField
          v-model:value="configForOIDC.clientId"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. 6655asd77895265aa110ac0d3"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Client secret
          <span class="text-red-600">*</span>
        </p>
        <BBTextField
          v-model:value="configForOIDC.clientSecret"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
          <NCheckbox
            v-model:checked="configForOIDC.skipTlsVerify"
            :label="$t('settings.sso.form.connection-security-skip-tls-verify')"
            :disabled="!allowEdit"
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
        <BBTextField
          v-model:value="configForOIDC.fieldMapping!.identifier"
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. preferred_username"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
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
      <div class="w-full grid grid-cols-[256px_1fr]">
        <BBTextField
          v-model:value="configForOIDC.fieldMapping!.displayName"
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <BBTextField
          v-model:value="configForOIDC.fieldMapping!.phone"
          :disabled="!allowEdit"
          class="mt-1 w-full"
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

    <!-- LDAP form group -->
    <div
      v-else-if="state.type === IdentityProviderType.LDAP"
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
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Host
          <span class="text-red-600">*</span>
        </p>
        <BBTextField
          v-model:value="configForLDAP.host"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. ldap.example.com"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Port
          <span class="text-red-600">*</span>
        </p>
        <NInputNumber
          v-model:value="configForLDAP.port"
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. 389 or 636"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Bind DN
          <span class="text-red-600">*</span>
        </p>
        <BBTextField
          v-model:value="configForLDAP.bindDn"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. uid=system,ou=Users,dc=example,dc=com"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Bind Password
          <span class="text-red-600">*</span>
        </p>
        <BBTextField
          v-model:value="configForLDAP.bindPassword"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
          :placeholder="$t('common.sensitive-placeholder')"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          Base DN
          <span class="text-red-600">*</span>
        </p>
        <BBTextField
          v-model="configForLDAP.baseDn"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. ou=users,dc=example,dc=com"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          User Filter
          <span class="text-red-600">*</span>
        </p>
        <BBTextField
          v-model="configForLDAP.userFilter"
          required
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. (uid=%s)"
        />
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.security-protocol") }}
          <span class="text-red-600">*</span>
        </p>
        <p class="textinfolabel mt-1">
          <NRadioGroup v-model:value="configForLDAP.securityProtocol">
            <NRadio value="starttls" label="StartTLS" />
            <NRadio value="ldaps" label="LDAPS" />
          </NRadioGroup>
        </p>
      </div>
      <div class="w-full flex flex-col justify-start items-start">
        <p class="textlabel">
          {{ $t("settings.sso.form.connection-security") }}
        </p>
        <p class="textinfolabel mt-1">
          <NCheckbox
            v-model:checked="configForLDAP.skipTlsVerify"
            :label="$t('settings.sso.form.connection-security-skip-tls-verify')"
            :disabled="!allowEdit"
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
            href="https://www.bytebase.com/docs/administration/sso/ldap#configuration?source=console"
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
        <BBTextField
          v-model:value="configForLDAP.fieldMapping!.identifier"
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. mail"
        />
        <div class="w-full flex flex-row justify-start items-center text-sm">
          <heroicons-outline:arrow-right
            class="mx-1 h-auto w-4 text-gray-300"
          />
          <p class="flex flex-row justify-start items-center">
            {{ $t("settings.sso.form.identifier")
            }}<span class="text-red-600">*</span>
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
        <BBTextField
          v-model:value="configForLDAP.fieldMapping!.displayName"
          :disabled="!allowEdit"
          class="mt-1 w-full"
          placeholder="e.g. displayName"
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
        <BBTextField
          v-model:value="configForLDAP.fieldMapping!.phone"
          :disabled="!allowEdit"
          class="mt-1 w-full"
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
        <NButton
          v-if="!isDeleted"
          :disabled="!allowTestConnection"
          @click="testConnection"
        >
          {{ $t("identity-provider.test-connection") }}
        </NButton>
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
          <NButton @click="handleCancelButtonClick">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!allowCreate"
            @click="handleCreateButtonClick"
          >
            {{ $t("common.create") }}
          </NButton>
        </template>
        <template v-else>
          <NButton
            :disabled="!allowUpdate"
            @click="handleDiscardChangesButtonClick"
          >
            {{ $t("common.discard-changes") }}
          </NButton>
          <NButton
            class="primary"
            :disabled="!allowUpdate"
            @click="handleUpdateButtonClick"
          >
            {{ $t("common.update") }}
          </NButton>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, head, isEqual } from "lodash-es";
import {
  NRadioGroup,
  NCheckbox,
  NRadio,
  NTooltip,
  NInputNumber,
} from "naive-ui";
import { ClientError, Status } from "nice-grpc-common";
import { computed, reactive, ref, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { identityProviderClient } from "@/grpcweb";
import { SETTING_ROUTE_WORKSPACE_SSO } from "@/router/dashboard/workspaceSetting";
import {
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
} from "@/store";
import { useIdentityProviderStore } from "@/store/modules/idp";
import {
  getIdentityProviderResourceId,
  idpNamePrefix,
} from "@/store/modules/v1/common";
import { OAuthWindowEventPayload, ResourceId, ValidatedMessage } from "@/types";
import { State } from "@/types/proto/v1/common";
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
import {
  IdentityProviderTemplate,
  hasWorkspacePermissionV2,
  identityProviderTemplateList,
  identityProviderTypeToString,
  openWindowForSSO,
  toClipboard,
} from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";

interface LocalState {
  type: IdentityProviderType;
}

const props = defineProps<{
  identityProviderName?: string;
}>();

const { t } = useI18n();
const router = useRouter();
const currentUser = useCurrentUserV1();
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
const configForLDAP = ref<LDAPIdentityProviderConfig>(
  LDAPIdentityProviderConfig.fromPartial({
    port: 389,
    securityProtocol: "starttls",
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
  return [
    IdentityProviderType.OAUTH2,
    IdentityProviderType.OIDC,
    IdentityProviderType.LDAP,
  ];
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
  } else if (state.type === IdentityProviderType.LDAP) {
    return "https://www.bytebase.com/docs/administration/sso/ldap?source=console";
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
  } else if (state.type === IdentityProviderType.LDAP) {
    if (
      !configForLDAP.value.host ||
      !configForLDAP.value.port ||
      !configForLDAP.value.bindDn ||
      !configForLDAP.value.baseDn ||
      !configForLDAP.value.userFilter ||
      !configForLDAP.value.securityProtocol ||
      !configForLDAP.value.fieldMapping?.identifier ||
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
    return hasWorkspacePermissionV2(
      currentUser.value,
      "bb.identityProviders.create"
    );
  } else if (!isDeleted.value) {
    return hasWorkspacePermissionV2(
      currentUser.value,
      "bb.identityProviders.update"
    );
  } else {
    return false;
  }
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
    config: IdentityProviderConfig.fromPartial({}),
  };
  if (tempIdentityProvider.type === IdentityProviderType.OAUTH2) {
    tempIdentityProvider.config!.oauth2Config = {
      ...configForOAuth2.value,
      scopes: scopesStringOfConfig.value.split(" "),
    };
  } else if (tempIdentityProvider.type === IdentityProviderType.OIDC) {
    tempIdentityProvider.config!.oidcConfig = configForOIDC.value;
  } else if (tempIdentityProvider.type === IdentityProviderType.LDAP) {
    tempIdentityProvider.config!.ldapConfig = configForLDAP.value;
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
    } else if (tempIdentityProvider.type === IdentityProviderType.LDAP) {
      configForLDAP.value =
        tempIdentityProvider.config?.ldapConfig ||
        LDAPIdentityProviderConfig.fromPartial({
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

const testConnection = async () => {
  if (
    state.type === IdentityProviderType.OAUTH2 ||
    state.type === IdentityProviderType.OIDC
  ) {
    openWindowForSSO(editedIdentityProvider.value);
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
    scopesStringOfConfig.value = template.config.scopes.join(" ");
  }
};

const handleDeleteButtonClick = async () => {
  if (!currentIdentityProvider.value) {
    return;
  }
  if (
    !hasWorkspacePermissionV2(currentUser.value, "bb.identityProviders.delete")
  ) {
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
  router.push({
    name: SETTING_ROUTE_WORKSPACE_SSO,
  });
};

const handleRestoreButtonClick = async () => {
  if (!currentIdentityProvider.value) {
    return;
  }
  if (
    !hasWorkspacePermissionV2(
      currentUser.value,
      "bb.identityProviders.undelete"
    )
  ) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: "Permission denied",
    });
    return;
  }

  await identityProviderStore.undeleteIdentityProvider(
    currentIdentityProvider.value.name
  );
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: "Restore SSO succeed",
  });
};

const handleCancelButtonClick = () => {
  router.push({
    name: SETTING_ROUTE_WORKSPACE_SSO,
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
  } else if (tempIdentityProvider.type === IdentityProviderType.LDAP) {
    configForLDAP.value =
      tempIdentityProvider.config?.ldapConfig ||
      LDAPIdentityProviderConfig.fromPartial({
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
  } else if (state.type === IdentityProviderType.LDAP) {
    identityProviderCreate.config!.ldapConfig = configForLDAP.value;
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
    name: SETTING_ROUTE_WORKSPACE_SSO,
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
    } else if (
      state.type === IdentityProviderType.OIDC ||
      state.type === IdentityProviderType.LDAP
    ) {
      // NOTE: We do not yet have templates for OIDC nor LDAP providers so resetting
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
