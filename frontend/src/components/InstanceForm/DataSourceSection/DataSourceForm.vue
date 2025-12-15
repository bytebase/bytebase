<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <div class="grid grid-cols-1 gap-y-4 gap-x-4 border-none sm:grid-cols-3">
    <template
      v-if="
        basicInfo.engine !== Engine.SPANNER &&
        basicInfo.engine !== Engine.BIGQUERY &&
        basicInfo.engine !== Engine.DYNAMODB &&
        basicInfo.engine !== Engine.DATABRICKS
      "
    >
      <div
        v-if="
          basicInfo.engine === Engine.MYSQL ||
          basicInfo.engine === Engine.POSTGRES ||
          basicInfo.engine === Engine.COSMOSDB ||
          basicInfo.engine === Engine.MSSQL ||
          basicInfo.engine === Engine.ELASTICSEARCH
        "
        class="sm:col-span-3 sm:col-start-1"
      >
        <NRadioGroup
          v-model:value="dataSource.authenticationType"
          class="textlabel"
          :disabled="!allowEdit"
        >
          <NRadio
            v-for="item in supportedAuthenticationTypes"
            :key="item.value"
            :value="item.value"
          >
            {{ item.label }}
          </NRadio>
        </NRadioGroup>
      </div>
      <div
        v-else-if="basicInfo.engine === Engine.HIVE"
        class="sm:col-span-3 sm:col-start-1"
      >
        <NRadioGroup
          :value="hiveAuthentication"
          class="textlabel"
          :disabled="!allowEdit"
          @update:value="onHiveAuthenticationChange"
        >
          <NRadio value="PASSWORD"> Plain Password </NRadio>
          <NRadio value="KERBEROS"> Kerberos </NRadio>
        </NRadioGroup>
      </div>
      <CreateDataSourceExample
        class-name="sm:col-span-3 border-none"
        :create-instance-flag="isCreating"
        :engine="basicInfo.engine"
        :data-source-type="dataSource.type"
        :authentication-type="dataSource.authenticationType"
      />
      <template v-if="dataSource.saslConfig?.mechanism?.case === 'krbConfig'">
        <div class="sm:col-span-3 sm:col-start-1">
          <label class="textlabel block">
            Principal
            <RequiredStar />
          </label>
          <div class="mt-2 flex items-center gap-x-2">
            <NInput
              :value="
                dataSource.saslConfig?.mechanism?.case === 'krbConfig'
                  ? dataSource.saslConfig.mechanism.value.primary
                  : ''
              "
              :disabled="!allowEdit"
              placeholder="primary"
              @update:value="
                (val) => {
                  if (dataSource.saslConfig?.mechanism?.case === 'krbConfig') {
                    dataSource.saslConfig.mechanism.value.primary = val;
                  }
                }
              "
            />
            <span>/</span>
            <NInput
              :value="
                dataSource.saslConfig?.mechanism?.case === 'krbConfig'
                  ? dataSource.saslConfig.mechanism.value.instance
                  : ''
              "
              :disabled="!allowEdit"
              placeholder="instance, optional"
              @update:value="
                (val) => {
                  if (dataSource.saslConfig?.mechanism?.case === 'krbConfig') {
                    dataSource.saslConfig.mechanism.value.instance = val;
                  }
                }
              "
            />
            <span>@</span>
            <NInput
              :value="
                dataSource.saslConfig?.mechanism?.case === 'krbConfig'
                  ? dataSource.saslConfig.mechanism.value.realm
                  : ''
              "
              :disabled="!allowEdit"
              placeholder="realm"
              @update:value="
                (val) => {
                  if (dataSource.saslConfig?.mechanism?.case === 'krbConfig') {
                    dataSource.saslConfig.mechanism.value.realm = val;
                  }
                }
              "
            />
          </div>
        </div>
        <div class="sm:col-span-3 sm:col-start-1">
          <label class="textlabel block">
            KDC
            <RequiredStar />
          </label>
          <div class="flex items-center gap-x-2">
            <div class="w-fit">
              <NRadioGroup
                :value="
                  dataSource.saslConfig?.mechanism?.case === 'krbConfig'
                    ? dataSource.saslConfig.mechanism.value.kdcTransportProtocol
                    : 'tcp'
                "
                class="textlabel w-32"
                :disabled="!allowEdit"
                @update:value="
                  (val) => {
                    if (
                      dataSource.saslConfig?.mechanism?.case === 'krbConfig'
                    ) {
                      dataSource.saslConfig.mechanism.value.kdcTransportProtocol =
                        val;
                    }
                  }
                "
              >
                <NRadio value="tcp"> TCP </NRadio>
                <NRadio value="udp"> UDP </NRadio>
              </NRadioGroup>
            </div>
            <NInput
              :value="
                dataSource.saslConfig?.mechanism?.case === 'krbConfig'
                  ? dataSource.saslConfig.mechanism.value.kdcHost
                  : ''
              "
              :disabled="!allowEdit"
              placeholder="KDC host"
              @update:value="
                (val) => {
                  if (dataSource.saslConfig?.mechanism?.case === 'krbConfig') {
                    dataSource.saslConfig.mechanism.value.kdcHost = val;
                  }
                }
              "
            />
            <span>:</span>
            <NInput
              :value="
                dataSource.saslConfig?.mechanism?.case === 'krbConfig'
                  ? dataSource.saslConfig.mechanism.value.kdcPort
                  : ''
              "
              :disabled="!allowEdit"
              placeholder="KDC port, optional"
              :allow-input="onlyAllowNumber"
              @update:value="
                (val) => {
                  if (dataSource.saslConfig?.mechanism?.case === 'krbConfig') {
                    dataSource.saslConfig.mechanism.value.kdcPort = val;
                  }
                }
              "
            />
          </div>
        </div>
        <div class="sm:col-span-3 sm:col-start-1">
          <label class="textlabel block">
            Keytab File
            <RequiredStar />
          </label>
          <NUpload :max="1" @change="handleKeytabUpload">
            <NUploadDragger class="mt-3">
              <span class="textinfolabel"
                >Click or Drag your .keytab file here</span
              >
            </NUploadDragger>
          </NUpload>
        </div>
      </template>
      <template v-else>
        <div
          v-if="
            dataSource.authenticationType !==
            DataSource_AuthenticationType.AZURE_IAM
          "
          class="sm:col-span-3 sm:col-start-1"
        >
          <label for="username" class="textlabel block">
            {{ $t("common.username") }}
          </label>
          <!-- For mysql, username can be empty indicating anonymous user.
      But it's a very bad practice to use anonymous user for admin operation,
      thus we make it REQUIRED here.-->
          <NInput
            v-model:value="dataSource.username"
            class="mt-2 w-full"
            :disabled="!allowEdit"
            :placeholder="
              basicInfo.engine === Engine.CLICKHOUSE ? $t('common.default') : ''
            "
          />
        </div>
        <CredentialSourceForm
          v-if="
            dataSource.authenticationType ===
              DataSource_AuthenticationType.AZURE_IAM ||
            dataSource.authenticationType ===
              DataSource_AuthenticationType.AWS_RDS_IAM ||
            dataSource.authenticationType ===
              DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
          "
          :data-source="dataSource"
          :allow-edit="allowEdit"
        />
        <div
          v-if="
            dataSource.authenticationType ===
            DataSource_AuthenticationType.AWS_RDS_IAM
          "
          class="sm:col-span-3 sm:col-start-1"
        >
          <label for="username" class="textlabel block">
            {{ $t("instance.database-region") }}
            <RequiredStar />
          </label>
          <NInput
            v-model:value="dataSource.region"
            class="mt-2 w-full"
            :disabled="!allowEdit"
            :placeholder="'database region, for example, us-east-1'"
          />
        </div>
        <div
          v-if="
            dataSource.authenticationType ===
            DataSource_AuthenticationType.PASSWORD
          "
          class="sm:col-span-3 sm:col-start-1"
        >
          <div v-if="!hideAdvancedFeatures" class="mb-4">
            <NRadioGroup
              class="textlabel"
              :value="state.passwordType"
              :disabled="!allowEdit"
              @update:value="changeSecretType"
            >
              <NRadio
                :value="
                  DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED
                "
              >
                {{ $t("instance.password-type.password") }}
              </NRadio>
              <NRadio :value="DataSourceExternalSecret_SecretType.VAULT_KV_V2">
                <div class="flex items-center gap-x-1">
                  {{ $t("instance.password-type.external-secret-vault") }}
                  <FeatureBadge
                    :feature="PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER"
                  />
                </div>
              </NRadio>
              <NRadio
                :value="DataSourceExternalSecret_SecretType.AWS_SECRETS_MANAGER"
              >
                <div class="flex items-center gap-x-1">
                  {{ $t("instance.password-type.external-secret-aws") }}
                  <FeatureBadge
                    :feature="PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER"
                  />
                </div>
              </NRadio>
              <NRadio
                :value="DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER"
              >
                <div class="flex items-center gap-x-1">
                  {{ $t("instance.password-type.external-secret-gcp") }}
                  <FeatureBadge
                    :feature="PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER"
                  />
                </div>
              </NRadio>
              <NRadio
                :value="DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT"
              >
                <div class="flex items-center gap-x-1">
                  {{ $t("instance.password-type.external-secret-azure") }}
                  <FeatureBadge
                    :feature="PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER"
                  />
                </div>
              </NRadio>
            </NRadioGroup>
            <LearnMoreLink
              url="https://docs.bytebase.com/get-started/connect/overview#secret-manager-integration"
              class="text-sm"
            />
          </div>
          <div
            v-if="
              state.passwordType ===
              DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED
            "
          >
            <label class="textlabel block">
              {{ $t("common.password") }}
            </label>
            <div class="mt-2">
              <NCheckbox
                v-if="!isCreating && allowUsingEmptyPassword"
                :size="'small'"
                :checked="dataSource.useEmptyPassword"
                :disabled="!allowEdit"
                @update:checked="toggleUseEmptyPassword"
              >
                {{ $t("instance.no-password") }}
              </NCheckbox>
              <NInput
                type="password"
                show-password-on="click"
                class="w-full"
                :input-props="{ autocomplete: 'off' }"
                :placeholder="
                  dataSource.useEmptyPassword
                    ? $t('instance.no-password')
                    : $t('instance.password-write-only')
                "
                :disabled="!allowEdit || dataSource.useEmptyPassword"
                :value="
                  dataSource.useEmptyPassword ? '' : dataSource.updatedPassword
                "
                @update:value="dataSource.updatedPassword = $event.trim()"
              />
            </div>
          </div>
          <div
            v-else-if="dataSource.externalSecret"
            class="flex flex-col gap-y-4"
          >
            <div
              v-if="
                state.passwordType ===
                DataSourceExternalSecret_SecretType.VAULT_KV_V2
              "
              class="flex flex-col gap-y-4"
            >
              <div class="sm:col-span-2 sm:col-start-1">
                <label class="textlabel block">
                  {{ $t("instance.external-secret-vault.vault-url") }}
                  <RequiredStar />
                </label>
                <BBTextField
                  v-model:value="dataSource.externalSecret.url"
                  :required="true"
                  class="mt-2 w-full"
                  :disabled="!allowEdit"
                  :placeholder="$t('instance.external-secret-vault.vault-url')"
                />
              </div>
              <div class="sm:col-span-2 sm:col-start-1 flex flex-col gap-y-2">
                <label class="textlabel block">
                  {{
                    $t("instance.external-secret-vault.vault-auth-type.self")
                  }}
                </label>
                <NRadioGroup
                  class="textlabel"
                  :value="dataSource.externalSecret.authType"
                  @update:value="changeExternalSecretAuthType"
                >
                  <NRadio :value="DataSourceExternalSecret_AuthType.TOKEN">
                    {{
                      $t(
                        "instance.external-secret-vault.vault-auth-type.token.self"
                      )
                    }}
                  </NRadio>
                  <NRadio
                    :value="DataSourceExternalSecret_AuthType.VAULT_APP_ROLE"
                  >
                    {{
                      $t(
                        "instance.external-secret-vault.vault-auth-type.approle.self"
                      )
                    }}
                  </NRadio>
                </NRadioGroup>
              </div>
              <div
                v-if="
                  dataSource.externalSecret.authType ===
                  DataSourceExternalSecret_AuthType.TOKEN
                "
                class="sm:col-span-2 sm:col-start-1"
              >
                <label class="textlabel block">
                  {{
                    $t(
                      "instance.external-secret-vault.vault-auth-type.token.self"
                    )
                  }}
                  <RequiredStar />
                </label>
                <div class="flex gap-x-2 text-sm">
                  <div class="textinfolabel">
                    {{
                      $t(
                        "instance.external-secret-vault.vault-auth-type.token.tips"
                      )
                    }}
                  </div>
                  <LearnMoreLink
                    url="https://developer.hashicorp.com/vault/tutorials/operations/generate-root"
                    class="ml-1 text-sm"
                  />
                </div>
                <BBTextField
                  :value="
                    dataSource.externalSecret?.authOption?.case === 'token'
                      ? dataSource.externalSecret.authOption.value
                      : ''
                  "
                  class="mt-2 w-full"
                  :disabled="!allowEdit"
                  :placeholder="secretInputPlaceholder"
                  :required="isCreating"
                  @update:value="
                    (val: string) => {
                      const ds = dataSource;
                      if (ds.externalSecret) {
                        ds.externalSecret.authOption = {
                          case: 'token',
                          value: val,
                        };
                      }
                    }
                  "
                />
              </div>
              <div
                v-else-if="
                  dataSource.externalSecret?.authOption?.case === 'appRole'
                "
                class="flex flex-col gap-y-4"
              >
                <div class="sm:col-span-2 sm:col-start-1">
                  <label class="textlabel block">
                    {{
                      $t(
                        "instance.external-secret-vault.vault-auth-type.approle.role-id"
                      )
                    }}
                    <RequiredStar />
                  </label>
                  <BBTextField
                    :value="
                      dataSource.externalSecret?.authOption?.case === 'appRole'
                        ? dataSource.externalSecret.authOption.value.roleId
                        : ''
                    "
                    :required="isCreating"
                    class="mt-2 w-full"
                    :disabled="!allowEdit"
                    :placeholder="`${$t(
                      'instance.external-secret-vault.vault-auth-type.approle.role-id'
                    )} - ${$t('common.write-only')}`"
                    @update:value="
                      (val: string) => {
                        const ds = dataSource;
                        if (ds.externalSecret?.authOption?.case === 'appRole') {
                          ds.externalSecret.authOption.value.roleId = val;
                        }
                      }
                    "
                  />
                </div>
                <div class="sm:col-span-2 sm:col-start-1">
                  <label class="textlabel block">
                    {{
                      $t(
                        "instance.external-secret-vault.vault-auth-type.approle.secret-id"
                      )
                    }}
                    <RequiredStar />
                  </label>
                  <i18n-t
                    tag="div"
                    keypath="instance.external-secret-vault.vault-auth-type.approle.secret-tips"
                    class="textinfolabel text-sm"
                  >
                    <template #learn_more>
                      <LearnMoreLink
                        url="https://developer.hashicorp.com/vault/tutorials/auth-methods/approle"
                        class="ml-1 text-sm"
                      />
                    </template>
                  </i18n-t>
                  <NRadioGroup
                    :value="
                      dataSource.externalSecret?.authOption?.case === 'appRole'
                        ? dataSource.externalSecret.authOption.value.type
                        : DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN
                    "
                    class="textlabel my-1"
                    :disabled="!allowEdit"
                    @update:value="
                      (val) => {
                        if (
                          dataSource.externalSecret?.authOption?.case ===
                          'appRole'
                        ) {
                          dataSource.externalSecret.authOption.value.type = val;
                        }
                      }
                    "
                  >
                    <NRadio
                      :value="
                        DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN
                      "
                    >
                      {{
                        $t(
                          "instance.external-secret-vault.vault-auth-type.approle.secret-plain-text"
                        )
                      }}
                    </NRadio>
                    <NRadio
                      :value="
                        DataSourceExternalSecret_AppRoleAuthOption_SecretType.ENVIRONMENT
                      "
                    >
                      {{
                        $t(
                          "instance.external-secret-vault.vault-auth-type.approle.secret-env-name"
                        )
                      }}
                    </NRadio>
                  </NRadioGroup>
                  <BBTextField
                    :value="
                      dataSource.externalSecret?.authOption?.case === 'appRole'
                        ? dataSource.externalSecret.authOption.value.secretId
                        : ''
                    "
                    class="mt-2 w-full"
                    :disabled="!allowEdit"
                    :placeholder="secretInputPlaceholder"
                    @update:value="
                      (val: string) => {
                        const ds = dataSource;
                        if (ds.externalSecret?.authOption?.case === 'appRole') {
                          ds.externalSecret.authOption.value.secretId = val;
                        }
                      }
                    "
                  />
                </div>
              </div>
              <div class="sm:col-span-2 sm:col-start-1">
                <label class="textlabel block">
                  {{ $t("instance.external-secret-vault.vault-tls-config") }}
                </label>
                <SslCertificateFormV1
                  v-model:verify="verifyVaultTls"
                  v-model:ca="dataSource.externalSecret.vaultSslCa"
                  v-model:cert="dataSource.externalSecret.vaultSslCert"
                  v-model:private-key="dataSource.externalSecret.vaultSslKey"
                  :disabled="!allowEdit"
                  :show-tooltip="false"
                  :show-key-field="true"
                  :show-cert-field="true"
                  :verify-label="
                    $t(
                      'instance.external-secret-vault.verify-vault-certificate'
                    )
                  "
                  :ca-label="$t('instance.external-secret-vault.vault-ca-cert')"
                  :cert-label="
                    $t('instance.external-secret-vault.vault-client-cert')
                  "
                  :key-label="
                    $t('instance.external-secret-vault.vault-client-key')
                  "
                />
              </div>
              <div class="sm:col-span-2 sm:col-start-1">
                <label class="textlabel block">
                  {{
                    $t(
                      "instance.external-secret-vault.vault-secret-engine-name"
                    )
                  }}
                  <RequiredStar />
                </label>
                <div class="flex gap-x-2 text-sm textinfolabel">
                  {{
                    $t(
                      "instance.external-secret-vault.vault-secret-engine-tips"
                    )
                  }}
                </div>
                <BBTextField
                  v-model:value="dataSource.externalSecret.engineName"
                  :required="true"
                  class="mt-2 w-full"
                  :disabled="!allowEdit"
                  :placeholder="
                    $t(
                      'instance.external-secret-vault.vault-secret-engine-name'
                    )
                  "
                />
              </div>
            </div>
            <div
              v-if="
                state.passwordType ===
                DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT
              "
              class="flex flex-col gap-y-4"
            >
              <div class="sm:col-span-2 sm:col-start-1">
                <label class="textlabel block">
                  {{ $t("instance.external-secret-azure.vault-url") }}
                  <RequiredStar />
                </label>
                <div class="flex gap-x-2 text-sm textinfolabel">
                  {{ $t("instance.external-secret-azure.vault-url-tips") }}
                </div>
                <BBTextField
                  v-model:value="dataSource.externalSecret.url"
                  :required="true"
                  class="mt-2 w-full"
                  :disabled="!allowEdit"
                  :placeholder="$t('instance.external-secret-azure.vault-url')"
                />
              </div>
            </div>
            <div class="sm:col-span-2 sm:col-start-1">
              <label class="textlabel block">
                {{ secretNameLabel }}
                <RequiredStar />
              </label>
              <div
                v-if="
                  state.passwordType ===
                  DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER
                "
                class="flex gap-x-2 text-sm textinfolabel"
              >
                {{ $t("instance.external-secret-gcp.secret-name-tips") }}
              </div>
              <div
                v-else-if="
                  state.passwordType ===
                  DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT
                "
                class="flex gap-x-2 text-sm textinfolabel"
              >
                {{ $t("instance.external-secret-azure.secret-name-tips") }}
              </div>
              <BBTextField
                v-model:value="dataSource.externalSecret.secretName"
                :required="true"
                class="mt-2 w-full"
                :disabled="!allowEdit"
                :placeholder="secretNameLabel"
              />
            </div>
            <div
              v-if="
                state.passwordType !==
                  DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER &&
                state.passwordType !==
                  DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT
              "
              class="sm:col-span-2 sm:col-start-1"
            >
              <label class="textlabel block">
                {{ secretKeyLabel }}
                <RequiredStar />
              </label>
              <BBTextField
                v-model:value="dataSource.externalSecret.passwordKeyName"
                :required="true"
                class="mt-2 w-full"
                :disabled="!allowEdit"
                :placeholder="secretKeyLabel"
              />
            </div>
          </div>

          <template
            v-if="
              basicInfo.engine === Engine.REDIS &&
              dataSource.redisType === DataSource_RedisType.SENTINEL
            "
          >
            <div class="mt-2">
              <label class="textlabel"> Master Name <RequiredStar /></label>
              <NInput
                v-model:value="dataSource.masterName"
                class="mt-1 w-full"
                :disabled="!allowEdit"
                :placeholder="''"
              />
            </div>
            <div class="mt-2">
              <label class="textlabel"> Master Username </label>
              <NInput
                v-model:value="dataSource.masterUsername"
                class="mt-1 w-full"
                :disabled="!allowEdit"
                :placeholder="''"
              />
            </div>
            <div class="mt-2">
              <label class="textlabel block"> Master Password </label>
              <div class="mt-2">
                <NCheckbox
                  v-if="!isCreating && allowUsingEmptyPassword"
                  :size="'small'"
                  :checked="dataSource.useEmptyMasterPassword"
                  :disabled="!allowEdit"
                  @update:checked="toggleUseEmptyMasterPassword"
                >
                  {{ $t("instance.no-password") }}
                </NCheckbox>
                <NInput
                  type="password"
                  show-password-on="click"
                  class="w-full"
                  :input-props="{ autocomplete: 'off' }"
                  :placeholder="
                    dataSource.useEmptyMasterPassword
                      ? $t('instance.no-password')
                      : $t('instance.password-write-only')
                  "
                  :disabled="!allowEdit || dataSource.useEmptyMasterPassword"
                  :value="
                    dataSource.useEmptyMasterPassword
                      ? ''
                      : dataSource.updatedMasterPassword
                  "
                  @update:value="
                    dataSource.updatedMasterPassword = $event.trim()
                  "
                />
              </div>
            </div>
          </template>
        </div>
      </template>
    </template>
    <template
      v-if="
        basicInfo.engine === Engine.SPANNER ||
        basicInfo.engine === Engine.BIGQUERY
      "
    >
      <NRadioGroup
        v-model:value="dataSource.authenticationType"
        class="textlabel"
        :disabled="!allowEdit"
      >
        <NRadio
          v-for="item in supportedAuthenticationTypes"
          :key="item.value"
          :value="item.value"
        >
          {{ item.label }}
        </NRadio>
      </NRadioGroup>
      <CredentialSourceForm :data-source="dataSource" :allow-edit="allowEdit" />
    </template>

    <template v-if="basicInfo.engine === Engine.ORACLE">
      <OracleSIDAndServiceNameInput
        v-model:sid="dataSource.sid"
        v-model:service-name="dataSource.serviceName"
        :allow-edit="allowEdit"
      />
    </template>

    <template v-if="basicInfo.engine === Engine.SNOWFLAKE">
      <div class="sm:col-span-3 sm:col-start-1">
        <div class="textlabel block">
          {{ $t("data-source.ssh.private-key") }}
        </div>
        <div class="flex gap-x-2 text-sm">
          <div class="textinfolabel">
            {{ $t("data-source.snowflake-keypair-tip") }}
          </div>
          <LearnMoreLink
            url="https://docs.snowflake.com/en/user-guide/key-pair-auth"
            class="ml-1 text-sm"
          />
        </div>
        <DroppableTextarea
          v-model:value="dataSource.authenticationPrivateKey"
          :resizable="false"
          :disabled="!allowEdit"
          class="w-full h-32 mt-2 whitespace-pre-wrap"
          placeholder="-----BEGIN PRIVATE KEY-----
MIIEvQ...
-----END PRIVATE KEY-----"
          :allow-edit="allowEdit"
        />
      </div>
      <div class="sm:col-span-3 sm:col-start-1">
        <div class="textlabel block">
          {{ $t("data-source.private-key-passphrase") }}
        </div>
        <div class="textinfolabel text-sm">
          {{ $t("data-source.private-key-passphrase-tip") }}
        </div>
        <NInput
          v-model:value="dataSource.authenticationPrivateKeyPassphrase"
          type="password"
          class="mt-2 w-full"
          :disabled="!allowEdit"
          :placeholder="$t('data-source.private-key-passphrase-placeholder')"
        />
      </div>
    </template>

    <template v-if="basicInfo.engine === Engine.DATABRICKS">
      <div>
        <div class="textlabel black mt-2">
          Warehouse ID
          <RequiredStar />
        </div>
        <NInput
          v-model:value="dataSource.warehouseId"
          class="mt-2"
          :disabled="!allowEdit"
        />
      </div>

      <div>
        <div class="textlabel black mt-2">
          Token
          <RequiredStar />
        </div>
        <NInput
          v-model:value="dataSource.authenticationPrivateKey"
          class="mt-2 w-full"
          :disabled="!allowEdit"
          placeholder="personal access token"
        />
      </div>
    </template>

    <template v-if="showAuthenticationDatabase">
      <div class="sm:col-span-3 sm:col-start-1">
        <div class="flex flex-row items-center gap-x-2">
          <label for="authenticationDatabase" class="textlabel block">
            {{ $t("instance.authentication-database") }}
          </label>
        </div>
        <NInput
          class="mt-2 w-full"
          :input-props="{ autocomplete: 'off' }"
          placeholder="admin"
          :disabled="!allowEdit"
          :value="dataSource.authenticationDatabase"
          @update:value="dataSource.authenticationDatabase = $event.trim()"
        />
      </div>
    </template>

    <template
      v-if="
        dataSource.type === DataSourceType.READ_ONLY &&
        (hasReadonlyReplicaHost || hasReadonlyReplicaPort)
      "
    >
      <div v-if="hasReadonlyReplicaHost" class="sm:col-span-3 sm:col-start-1">
        <div class="flex flex-row items-center gap-x-2">
          <label for="host" class="textlabel block">
            {{ $t("data-source.read-replica-host") }}
          </label>
        </div>
        <NInput
          class="mt-2 w-full"
          :input-props="{ autocomplete: 'off' }"
          :value="dataSource.host"
          :disabled="!allowEdit"
          @update:value="handleHostInput"
        />
      </div>

      <div v-if="hasReadonlyReplicaPort" class="sm:col-span-3 sm:col-start-1">
        <div class="flex flex-row items-center gap-x-2">
          <label for="port" class="textlabel block">
            {{ $t("data-source.read-replica-port") }}
          </label>
        </div>
        <NInput
          class="mt-2 w-full"
          :input-props="{ autocomplete: 'off' }"
          :value="dataSource.port"
          :allow-input="onlyAllowNumber"
          :disabled="!allowEdit"
          @update:value="handlePortInput"
        />
      </div>
    </template>

    <div v-if="showDatabase" class="sm:col-span-3 sm:col-start-1">
      <label for="database" class="textlabel block">
        {{ $t("common.database") }}
      </label>
      <NInput
        v-model:value="dataSource.database"
        class="mt-2 w-full"
        :disabled="!allowEdit"
        :placeholder="$t('common.database')"
      />
    </div>

    <div v-if="hasExtraParameters" class="sm:col-span-3 sm:col-start-1">
      <div class="flex flex-row items-center justify-between">
        <label class="textlabel block">
          {{ $t("data-source.extra-params.self") }}
        </label>
      </div>
      <div class="textinfolabel text-sm mt-1 mb-2">
        {{ $t("data-source.extra-params.description") }}
      </div>

      <!-- Add parameter form -->
      <div
        v-if="allowEdit"
        class="flex mt-2 mb-2 gap-x-2 bg-gray-50 p-3 rounded-md"
      >
        <NInput
          v-model:value="newParam.key"
          class="w-full"
          :placeholder="$t('instance.parameter-name-placeholder')"
        />
        <NInput
          v-model:value="newParam.value"
          class="w-full"
          :placeholder="$t('instance.parameter-value-placeholder')"
        />
        <NButton
          type="primary"
          ghost
          size="small"
          :disabled="!newParam.key.trim()"
          @click="addNewParameter"
        >
          Add
        </NButton>
      </div>

      <!-- Existing parameters -->
      <div
        v-for="(param, index) in extraConnectionParamsList"
        :key="param.key"
        class="flex mt-2 gap-x-2"
      >
        <NInput
          class="w-full"
          :value="param.key"
          :disabled="!allowEdit"
          placeholder="Parameter name"
          @update:value="(v) => updateExtraConnectionParamKey(index, v)"
        />
        <NInput
          class="w-full"
          :value="param.value"
          :disabled="!allowEdit"
          placeholder="Parameter value"
          @update:value="(v) => updateExtraConnectionParamValue(index, v)"
        />
        <NButton
          v-if="allowEdit"
          type="error"
          secondary
          size="small"
          @click="removeExtraConnectionParam(index)"
          title="Remove parameter"
        >
          Remove
        </NButton>
      </div>

      <!-- Show a message when there are no parameters -->
      <div
        v-if="extraConnectionParamsList.length === 0"
        class="textinfolabel text-sm italic"
      >
        {{
          allowEdit
            ? $t("instance.no-params-yet-add-above")
            : $t("instance.no-extra-params-configured")
        }}
      </div>
    </div>

    <div
      v-if="
        showSSL &&
        dataSource.authenticationType === DataSource_AuthenticationType.PASSWORD
      "
      class="sm:col-span-3 sm:col-start-1"
    >
      <div for="ssl" class="flex items-center justify-start gap-x-2 textlabel">
        {{ $t("data-source.ssl-connection") }}
        <Switch
          :text="true"
          :value="dataSource.useSsl"
          @update:value="handleUseSslChanged"
        />
      </div>
      <template v-if="dataSource.useSsl">
        <SslCertificateFormV1
          v-if="dataSource.pendingCreate || dataSource.updateSsl"
          class="pt-1!"
          v-model:verify="dataSource.verifyTlsCertificate"
          v-model:ca="dataSource.sslCa"
          v-model:cert="dataSource.sslCert"
          v-model:private-key="dataSource.sslKey"
          :engine-type="basicInfo.engine"
          :disabled="!allowEdit"
        />
        <NButton
          v-else
          class="mt-2!"
          :disabled="!allowEdit"
          @click.prevent="handleEditSSL"
        >
          {{ $t("common.edit") }} - {{ $t("common.write-only") }}
        </NButton>
      </template>
    </div>

    <div
      v-if="
        !hideAdvancedFeatures &&
        showSSH &&
        dataSource.authenticationType === DataSource_AuthenticationType.PASSWORD
      "
      class="sm:col-span-3 sm:col-start-1"
    >
      <div class="flex flex-row items-center gap-x-1">
        <label for="ssh" class="textlabel block">
          {{ $t("data-source.ssh-connection") }}
        </label>
      </div>
      <SshConnectionForm
        :value="dataSource"
        :instance="instance"
        :disabled="!allowEdit"
        @change="handleSSHChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import {
  NButton,
  NCheckbox,
  NInput,
  NRadio,
  NRadioGroup,
  NUpload,
  NUploadDragger,
  type UploadFileInfo,
} from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBTextField } from "@/bbkit";
import { FeatureBadge } from "@/components/FeatureGuard";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import DroppableTextarea from "@/components/misc/DroppableTextarea.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Switch } from "@/components/v2";
import type { DataSourceOptions } from "@/types/dataSource";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceExternalSecret_AppRoleAuthOption_SecretType,
  DataSourceExternalSecret_AppRoleAuthOptionSchema,
  DataSourceExternalSecret_AuthType,
  DataSourceExternalSecret_SecretType,
  DataSourceExternalSecretSchema,
  DataSourceType,
  KerberosConfigSchema,
  SASLConfigSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { onlyAllowNumber } from "@/utils";
import type { EditDataSource } from "../common";
import { useInstanceFormContext } from "../context";
import CreateDataSourceExample from "./CreateDataSourceExample.vue";
import CredentialSourceForm from "./CredentialSourceForm.vue";
import OracleSIDAndServiceNameInput from "./OracleSIDAndServiceNameInput.vue";
import SshConnectionForm from "./SshConnectionForm.vue";
import SslCertificateFormV1 from "./SslCertificateFormV1.vue";

interface LocalState {
  passwordType: DataSourceExternalSecret_SecretType;
}

interface ExtraConnectionParam {
  key: string;
  value: string;
}

const props = defineProps<{
  dataSource: EditDataSource;
}>();

const {
  instance,
  specs,
  isCreating,
  allowEdit,
  basicInfo,
  adminDataSource,
  hasReadonlyReplicaFeature,
  missingFeature,
  hideAdvancedFeatures,
} = useInstanceFormContext();

const {
  showDatabase,
  showSSL,
  showSSH,
  allowUsingEmptyPassword,
  showAuthenticationDatabase,
  hasReadonlyReplicaHost,
  hasReadonlyReplicaPort,
  hasExtraParameters,
} = specs;

const state = reactive<LocalState>({
  passwordType: DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED,
});

// Use a simpler approach to track new parameters
const newParam = reactive({ key: "", value: "" });

// Helper computed to convert object to array for UI
const extraConnectionParamsList = computed<ExtraConnectionParam[]>(() => {
  // Ensure we're using a non-null object for the params
  const params = props.dataSource.extraConnectionParameters || {};

  // Convert to plain entries for display
  return Object.entries(params).map(([key, value]) => ({ key, value }));
});
const { t } = useI18n();

watch(
  () => props.dataSource.externalSecret,
  (externalSecret) => {
    if (externalSecret) {
      state.passwordType = externalSecret.secretType;
    } else {
      state.passwordType =
        DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED;
    }
  },
  { immediate: true, deep: true }
);

// Watch SSL fields and set updateSsl flag when they change
watch(
  () => [
    () => props.dataSource.verifyTlsCertificate,
    () => props.dataSource.sslCa,
    () => props.dataSource.sslCert,
    () => props.dataSource.sslKey,
  ],
  () => {
    if (!props.dataSource.pendingCreate) {
      // eslint-disable-next-line vue/no-mutating-props
      props.dataSource.updateSsl = true;
    }
  }
);

const hiveAuthentication = computed(() => {
  return props.dataSource.saslConfig?.mechanism?.case === "krbConfig"
    ? "KERBEROS"
    : "PASSWORD";
});

const onHiveAuthenticationChange = (val: "KERBEROS" | "PASSWORD") => {
  const ds = props.dataSource;
  if (val === "KERBEROS") {
    ds.saslConfig = create(SASLConfigSchema, {
      mechanism: {
        case: "krbConfig",
        value: create(KerberosConfigSchema, {
          kdcTransportProtocol: "tcp",
        }),
      },
    });
  } else {
    ds.saslConfig = undefined;
  }
};

const supportedAuthenticationTypes = computed(() => {
  switch (basicInfo.value.engine) {
    case Engine.COSMOSDB:
      return [
        {
          value: DataSource_AuthenticationType.AZURE_IAM,
          label: t("instance.password-type.azure-iam"),
        },
      ];
    case Engine.MSSQL:
      return [
        {
          value: DataSource_AuthenticationType.PASSWORD,
          label: t("instance.password-type.password"),
        },
        {
          value: DataSource_AuthenticationType.AZURE_IAM,
          label: t("instance.password-type.azure-iam"),
        },
      ];
    case Engine.ELASTICSEARCH:
      return [
        {
          value: DataSource_AuthenticationType.PASSWORD,
          label: t("instance.password-type.password"),
        },
        {
          value: DataSource_AuthenticationType.AWS_RDS_IAM,
          label: t("instance.password-type.aws-iam"),
        },
      ];
    case Engine.SPANNER:
    case Engine.BIGQUERY: {
      return [
        {
          value: DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM,
          label: t("instance.password-type.google-iam"),
        },
      ];
    }
    default:
      return [
        {
          value: DataSource_AuthenticationType.PASSWORD,
          label: t("instance.password-type.password"),
        },
        {
          value: DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM,
          label: t("instance.password-type.google-iam"),
        },
        {
          value: DataSource_AuthenticationType.AWS_RDS_IAM,
          label: t("instance.password-type.aws-iam"),
        },
      ];
  }
});

const secretInputPlaceholder = computed(() => {
  switch (state.passwordType) {
    case DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED:
      return `${t("common.password")} - ${t("common.write-only")}`;
    case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
      switch (props.dataSource.externalSecret?.authType) {
        case DataSourceExternalSecret_AuthType.TOKEN:
          return `${t(
            "instance.external-secret-vault.vault-auth-type.token.self"
          )} - ${t("common.write-only")}`;
        case DataSourceExternalSecret_AuthType.VAULT_APP_ROLE:
          switch (
            props.dataSource.externalSecret.authOption?.case === "appRole"
              ? props.dataSource.externalSecret.authOption.value.type
              : undefined
          ) {
            case DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN:
              return `${t(
                "instance.external-secret-vault.vault-auth-type.approle.secret-id-plain-text"
              )} - ${t("common.write-only")}`;
            case DataSourceExternalSecret_AppRoleAuthOption_SecretType.ENVIRONMENT:
              return `${t(
                "instance.external-secret-vault.vault-auth-type.approle.secret-id-environment"
              )} - ${t("common.write-only")}`;
          }
      }
  }

  return "";
});

const secretNameLabel = computed(() => {
  switch (state.passwordType) {
    case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
      return t("instance.external-secret-vault.vault-secret-path");
    case DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER:
      return t("instance.external-secret-gcp.secret-name");
    case DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT:
      return t("instance.external-secret-azure.secret-name");
    default:
      return t("instance.external-secret.secret-name");
  }
});

const secretKeyLabel = computed(() => {
  if (state.passwordType == DataSourceExternalSecret_SecretType.VAULT_KV_V2) {
    return t("instance.external-secret-vault.vault-secret-key");
  }
  return t("instance.external-secret.key-name");
});

const verifyVaultTls = computed({
  get: () => {
    return !props.dataSource.externalSecret?.skipVaultTlsVerification;
  },
  set: (value: boolean) => {
    if (props.dataSource.externalSecret) {
      // eslint-disable-next-line vue/no-mutating-props
      props.dataSource.externalSecret.skipVaultTlsVerification = !value;
    }
  },
});

const changeSecretType = (secretType: DataSourceExternalSecret_SecretType) => {
  const ds = props.dataSource;
  switch (secretType) {
    case DataSourceExternalSecret_SecretType.SECRET_TYPE_UNSPECIFIED:
      ds.externalSecret = undefined;
      break;
    case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
      ds.externalSecret = create(DataSourceExternalSecretSchema, {
        authType: DataSourceExternalSecret_AuthType.TOKEN,
        secretType: secretType,
        authOption: { case: "token", value: "" },
        secretName: ds.externalSecret?.secretName ?? "",
        passwordKeyName: ds.externalSecret?.passwordKeyName ?? "",
      });
      break;
    case DataSourceExternalSecret_SecretType.AWS_SECRETS_MANAGER:
      ds.externalSecret = create(DataSourceExternalSecretSchema, {
        authType: DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED,
        secretType: secretType,
        authOption: { case: "token", value: "" },
        secretName: ds.externalSecret?.secretName ?? "",
        passwordKeyName: ds.externalSecret?.passwordKeyName ?? "",
      });
      break;
    case DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER:
      ds.externalSecret = create(DataSourceExternalSecretSchema, {
        authType: DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED,
        secretType: secretType,
        authOption: { case: "token", value: "" },
        secretName: ds.externalSecret?.secretName ?? "",
        passwordKeyName: "",
      });
      break;
    case DataSourceExternalSecret_SecretType.AZURE_KEY_VAULT:
      ds.externalSecret = create(DataSourceExternalSecretSchema, {
        authType: DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED,
        secretType: secretType,
        authOption: { case: "token", value: "" },
        url: ds.externalSecret?.url ?? "",
        secretName: ds.externalSecret?.secretName ?? "",
        passwordKeyName: "",
      });
      break;
  }

  state.passwordType = secretType;
};

const changeExternalSecretAuthType = (
  authType: DataSourceExternalSecret_AuthType
) => {
  const ds = props.dataSource;
  if (!ds.externalSecret) {
    return;
  }
  if (authType === DataSourceExternalSecret_AuthType.VAULT_APP_ROLE) {
    ds.externalSecret.authOption = {
      case: "appRole",
      value: create(DataSourceExternalSecret_AppRoleAuthOptionSchema, {
        type: DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN,
      }),
    };
  } else {
    ds.externalSecret.authOption = {
      case: "token",
      value: "",
    };
  }
  ds.externalSecret.authType = authType;
};

const toggleUseEmptyPassword = (on: boolean) => {
  const ds = props.dataSource;
  ds.useEmptyPassword = on;
  if (on) {
    ds.updatedPassword = "";
  }
};

const toggleUseEmptyMasterPassword = (on: boolean) => {
  const ds = props.dataSource;
  ds.useEmptyMasterPassword = on;
  if (on) {
    ds.updatedMasterPassword = "";
  }
};

const handleHostInput = (value: string) => {
  const ds = props.dataSource;
  if (ds.type === DataSourceType.READ_ONLY) {
    if (!hasReadonlyReplicaFeature.value) {
      if (ds.host || ds.port) {
        ds.host = adminDataSource.value.host;
        ds.port = adminDataSource.value.port;
        missingFeature.value =
          PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION;
        return;
      }
    }
  }
  ds.host = value.trim();
};

const handlePortInput = (value: string) => {
  const ds = props.dataSource;
  if (ds.type === DataSourceType.READ_ONLY) {
    if (!hasReadonlyReplicaFeature.value) {
      if (ds.host || ds.port) {
        ds.host = adminDataSource.value.host;
        ds.port = adminDataSource.value.port;
        missingFeature.value =
          PlanFeature.FEATURE_INSTANCE_READ_ONLY_CONNECTION;
        return;
      }
    }
  }
  ds.port = value.trim();
};

const handleUseSslChanged = (useSSL: boolean) => {
  const ds = props.dataSource;
  ds.useSsl = useSSL;
  ds.updateSsl = true;
};

const handleEditSSL = () => {
  const ds = props.dataSource;
  ds.sslCa = "";
  ds.sslCert = "";
  ds.sslKey = "";
  ds.updateSsl = true;
};

const handleSSHChange = (
  value: Partial<
    Pick<
      DataSourceOptions,
      "sshHost" | "sshPort" | "sshUser" | "sshPassword" | "sshPrivateKey"
    >
  >
) => {
  const ds = props.dataSource;
  Object.assign(ds, value);
};

watch(
  () => props.dataSource.authenticationPrivateKey,
  (privateKey) => {
    const ds = props.dataSource;
    ds.useEmptyPassword = !!privateKey;
  }
);

const handleKeytabUpload = (options: { file: UploadFileInfo }) => {
  const reader = new FileReader();
  reader.onload = function () {
    const arrayBuffer = reader.result as ArrayBuffer;
    const data = new Uint8Array(arrayBuffer);
    const ds = props.dataSource;
    if (ds.saslConfig?.mechanism?.case === "krbConfig") {
      ds.saslConfig.mechanism.value.keytab = data;
    }
  };
  reader.readAsArrayBuffer(options.file.file as Blob);
};

// Extra connection parameters management
const addNewParameter = () => {
  // Skip if key is empty
  if (!newParam.key.trim()) return;

  const ds = props.dataSource;

  // Get plain params object using our helper
  const plainParams = createPlainParamsObject(ds.extraConnectionParameters);

  // Add the new parameter
  const trimmedKey = newParam.key.trim();
  plainParams[trimmedKey] = newParam.value;

  // Create a fresh object to ensure we're using a plain JS object
  const freshParams: Record<string, string> = {};
  Object.keys(plainParams).forEach((key) => {
    freshParams[key] = plainParams[key];
  });

  // Set the object directly using a brand new object
  ds.extraConnectionParameters = freshParams;

  // Clear the form
  newParam.key = "";
  newParam.value = "";
};

// Helper function to create plain parameters object from existing parameters
const createPlainParamsObject = (
  existingParams: Record<string, string> | undefined
): Record<string, string> => {
  const plainParams: Record<string, string> = {};

  if (existingParams) {
    // Copy all properties from the potentially proxied object
    Object.entries(existingParams).forEach(([key, value]) => {
      plainParams[key] = value;
    });
  }

  return plainParams;
};

const updateExtraConnectionParamKey = (index: number, newKey: string) => {
  const ds = props.dataSource;
  const params = extraConnectionParamsList.value;
  if (index >= params.length) return;

  // Get plain params object
  const plainParams = createPlainParamsObject(ds.extraConnectionParameters);

  const oldKey = params[index].key;
  const value = params[index].value;

  // Skip if the key hasn't changed
  if (oldKey === newKey) return;

  // Delete the old key
  delete plainParams[oldKey];

  // Only add if the key is not empty
  if (newKey.trim()) {
    plainParams[newKey] = value;
  }

  // Set the object directly
  ds.extraConnectionParameters = plainParams;
};

const updateExtraConnectionParamValue = (index: number, newValue: string) => {
  const ds = props.dataSource;
  const params = extraConnectionParamsList.value;
  if (index >= params.length) return;

  const key = params[index].key;

  // Get plain params object
  const plainParams = createPlainParamsObject(ds.extraConnectionParameters);

  // Update the value
  plainParams[key] = newValue;

  // Set the object directly
  ds.extraConnectionParameters = plainParams;
};

const removeExtraConnectionParam = (index: number) => {
  const ds = props.dataSource;
  const params = extraConnectionParamsList.value;
  if (index >= params.length) return;

  // Get plain params object
  const plainParams = createPlainParamsObject(ds.extraConnectionParameters);

  // Remove the parameter
  delete plainParams[params[index].key];

  // Set the object directly
  ds.extraConnectionParameters = plainParams;
};
</script>
