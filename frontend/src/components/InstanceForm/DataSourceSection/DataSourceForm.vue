<template>
  <!-- eslint-disable vue/no-mutating-props -->
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
      class="mt-2 sm:col-span-3 sm:col-start-1"
    >
      <template v-if="basicInfo.engine === Engine.COSMOSDB">
        <NRadioGroup
          v-model:value="dataSource.authenticationType"
          class="textlabel"
          :disabled="!allowEdit"
        >
          <NRadio :value="DataSource_AuthenticationType.AZURE_IAM">
            {{ $t("instance.password-type.azure-iam") }}
          </NRadio>
        </NRadioGroup>
      </template>
      <template v-else-if="basicInfo.engine === Engine.MSSQL">
        <NRadioGroup
          v-model:value="dataSource.authenticationType"
          class="textlabel"
          :disabled="!allowEdit"
        >
          <NRadio :value="DataSource_AuthenticationType.PASSWORD">
            {{ $t("instance.password-type.password") }}
          </NRadio>
          <NRadio :value="DataSource_AuthenticationType.AZURE_IAM">
            {{ $t("instance.password-type.azure-iam") }}
          </NRadio>
        </NRadioGroup>
      </template>
      <template v-else-if="basicInfo.engine === Engine.ELASTICSEARCH">
        <NRadioGroup
          v-model:value="dataSource.authenticationType"
          class="textlabel"
          :disabled="!allowEdit"
        >
          <NRadio :value="DataSource_AuthenticationType.PASSWORD">
            {{ $t("instance.password-type.password") }}
          </NRadio>
          <NRadio :value="DataSource_AuthenticationType.AWS_RDS_IAM">
            {{ $t("instance.password-type.aws-iam") }}
          </NRadio>
        </NRadioGroup>
      </template>
      <template v-else>
        <NRadioGroup
          v-model:value="dataSource.authenticationType"
          class="textlabel"
          :disabled="!allowEdit"
        >
          <NRadio :value="DataSource_AuthenticationType.PASSWORD">
            {{ $t("instance.password-type.password") }}
          </NRadio>
          <NRadio :value="DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM">
            {{ $t("instance.password-type.google-iam") }}
          </NRadio>
          <NRadio :value="DataSource_AuthenticationType.AWS_RDS_IAM">
            {{ $t("instance.password-type.aws-iam") }}
          </NRadio>
        </NRadioGroup>
      </template>
    </div>
    <div
      v-else-if="basicInfo.engine === Engine.HIVE"
      class="mt-2 sm:col-span-3 sm:col-start-1"
    >
      <NRadioGroup
        :value="hiveAuthentication"
        class="textlabel"
        :disabled="!allowEdit"
        @update:value="onHiveAuthenticationChange"
      >
        <NRadio value="PASSWORD"> Plain password </NRadio>
        <NRadio value="KERBEROS"> Kerberos </NRadio>
      </NRadioGroup>
    </div>
    <CreateDataSourceExample
      class-name="sm:col-span-3 border-none mt-2"
      :create-instance-flag="isCreating"
      :engine="basicInfo.engine"
      :data-source-type="dataSource.type"
      :authentication-type="dataSource.authenticationType"
    />
    <div
      v-if="dataSource.saslConfig?.mechanism?.case === 'krbConfig'"
      class="sm:col-span-3 sm:col-start-1"
    >
      <div class="mt-4 sm:col-span-3 sm:col-start-1">
        <label class="textlabel block">
          Principal
          <span class="text-red-600">*</span>
        </label>
        <div class="flex mt-2 items-center space-x-2">
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
      <div class="mt-4 sm:col-span-3 sm:col-start-1">
        <label class="textlabel block">
          KDC
          <span class="text-red-600">*</span>
        </label>
        <div class="flex items-center space-x-2">
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
                  if (dataSource.saslConfig?.mechanism?.case === 'krbConfig') {
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
      <div class="mt-4 sm:col-span-3 sm:col-start-1">
        <label class="textlabel block">
          Keytab File
          <span class="text-red-600">*</span>
        </label>

        <NUpload :max="1" @change="handleKeytabUpload">
          <NUploadDragger class="mt-3">
            <span class="text-gray-400"
              >Click or Drag your .keytab file here</span
            >
          </NUploadDragger>
        </NUpload>
      </div>
    </div>
    <div v-else class="sm:col-span-3 sm:col-start-1">
      <div
        v-if="
          dataSource.authenticationType !==
          DataSource_AuthenticationType.AZURE_IAM
        "
        class="mt-4 sm:col-span-3 sm:col-start-1"
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
      <div
        v-if="
          dataSource.authenticationType ===
            DataSource_AuthenticationType.AZURE_IAM ||
          dataSource.authenticationType ===
            DataSource_AuthenticationType.AWS_RDS_IAM ||
          dataSource.authenticationType ===
            DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
        "
        class="mt-4 sm:col-span-3 sm:col-start-1"
      >
        <label for="credential-source" class="textlabel block">
          {{ $t("instance.iam-extension.credential-source") }}
        </label>
        <NRadioGroup
          v-model:value="state.credentialSource"
          class="textlabel"
          :disabled="!allowEdit"
        >
          <NRadio
            v-for="option in getIAMExtensionOptions(
              DataSource_AuthenticationType.AZURE_IAM
            )"
            :value="option.value"
            :key="option.value"
            :label="option.label"
          />
        </NRadioGroup>
        <template v-if="state.credentialSource === 'specific-credential'">
          <div
            v-if="
              dataSource.authenticationType ===
              DataSource_AuthenticationType.AZURE_IAM
            "
            class="mt-4 sm:col-span-3 sm:col-start-1"
          >
            <label for="tenant-id" class="textlabel block mt-2">
              {{ $t("instance.iam-extension.tenant-id") }}
            </label>
            <NInput
              v-model:value="
                (dataSource.iamExtension?.case === 'azureCredential'
                  ? dataSource.iamExtension.value
                  : {}
                ).tenantId
              "
              class="mt-2 w-full"
              :disabled="!allowEdit"
              :placeholder="''"
              @update:value="
                (val) => {
                  if (dataSource.iamExtension?.case === 'azureCredential') {
                    dataSource.iamExtension.value.tenantId = val;
                  }
                }
              "
            />
            <label for="client-id" class="textlabel block mt-2">
              {{ $t("instance.iam-extension.client-id") }}
            </label>
            <NInput
              v-model:value="
                (dataSource.iamExtension?.case === 'azureCredential'
                  ? dataSource.iamExtension.value
                  : {}
                ).clientId
              "
              class="mt-2 w-full"
              :disabled="!allowEdit"
              :placeholder="''"
              @update:value="
                (val) => {
                  if (dataSource.iamExtension?.case === 'azureCredential') {
                    dataSource.iamExtension.value.clientId = val;
                  }
                }
              "
            />
            <label for="client-secret" class="textlabel block mt-2">
              {{ $t("instance.iam-extension.client-secret") }}
            </label>
            <NInput
              type="password"
              show-password-on="click"
              v-model:value="
                (dataSource.iamExtension?.case === 'azureCredential'
                  ? dataSource.iamExtension.value
                  : {}
                ).clientSecret
              "
              class="mt-2 w-full"
              :disabled="!allowEdit"
              :placeholder="$t('instance.type-or-paste-credentials-write-only')"
              @update:value="
                (val) => {
                  if (dataSource.iamExtension?.case === 'azureCredential') {
                    dataSource.iamExtension.value.clientSecret = val;
                  }
                }
              "
            />
          </div>
          <div
            v-else-if="
              dataSource.authenticationType ===
              DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
            "
            class="mt-4 sm:col-span-3 sm:col-start-1"
          >
            <label class="textlabel block mt-2"> Credential File Content</label>
            <NInput
              v-model:value="
                (dataSource.iamExtension?.case === 'gcpCredential'
                  ? dataSource.iamExtension.value
                  : {}
                ).content
              "
              type="textarea"
              class="mt-2 w-full"
              :disabled="!allowEdit"
              :placeholder="$t('instance.type-or-paste-credentials-write-only')"
              @update:value="
                (val) => {
                  if (dataSource.iamExtension?.case === 'gcpCredential') {
                    dataSource.iamExtension.value.content = val;
                  }
                }
              "
            />
          </div>
          <div
            v-else-if="
              dataSource.authenticationType ===
              DataSource_AuthenticationType.AWS_RDS_IAM
            "
            class="mt-4 sm:col-span-3 sm:col-start-1"
          >
            <label class="textlabel block mt-2"> Access Key ID </label>
            <NInput
              v-model:value="
                (dataSource.iamExtension?.case === 'awsCredential'
                  ? dataSource.iamExtension.value
                  : {}
                ).accessKeyId
              "
              class="mt-2 w-full"
              :disabled="!allowEdit"
              :placeholder="$t('common.write-only')"
              @update:value="
                (val) => {
                  if (dataSource.iamExtension?.case === 'awsCredential') {
                    dataSource.iamExtension.value.accessKeyId = val;
                  }
                }
              "
            />
            <label class="textlabel block mt-2"> Secret Access Key </label>
            <NInput
              v-model:value="
                (dataSource.iamExtension?.case === 'awsCredential'
                  ? dataSource.iamExtension.value
                  : {}
                ).secretAccessKey
              "
              class="mt-2 w-full"
              :disabled="!allowEdit"
              :placeholder="$t('common.write-only')"
              @update:value="
                (val) => {
                  if (dataSource.iamExtension?.case === 'awsCredential') {
                    dataSource.iamExtension.value.secretAccessKey = val;
                  }
                }
              "
            />
            <label class="textlabel block mt-2"> Session Token </label>
            <NInput
              v-model:value="
                (dataSource.iamExtension?.case === 'awsCredential'
                  ? dataSource.iamExtension.value
                  : {}
                ).sessionToken
              "
              class="mt-2 w-full"
              :disabled="!allowEdit"
              :placeholder="$t('common.write-only')"
              @update:value="
                (val) => {
                  if (dataSource.iamExtension?.case === 'awsCredential') {
                    dataSource.iamExtension.value.sessionToken = val;
                  }
                }
              "
            />
          </div>
        </template>
        <div
          v-else-if="state.credentialSource === 'default'"
          class="mt-1 sm:col-span-3 sm:col-start-1 textinfolabel !leading-6 credential"
        >
          <span
            v-if="
              dataSource.authenticationType ===
              DataSource_AuthenticationType.AZURE_IAM
            "
          >
            Bytebase will read the credential from environment variables
            <code class="code">AZURE_CLIENT_ID</code>/
            <code class="code">AZURE_TENANT_ID</code>/
            <code class="code">AZURE_CLIENT_SECRET</code>
            or
            <code class="code">AZURE_CLIENT_CERTIFICATE_PATH</code>, and
            fallback to attached users in Azure VM
          </span>
          <span
            v-else-if="
              dataSource.authenticationType ===
              DataSource_AuthenticationType.AWS_RDS_IAM
            "
          >
            Bytebase will read the credential from environment variables
            <code class="code">AWS_ACCESS_KEY_ID</code>/
            <code class="code">AWS_SECRET_ACCESS_KEY</code>/
            <code class="code">AWS_SESSION_TOKEN</code>, fallback to shared
            credentials file <code class="code">~/.aws/credentials</code> or IAM
            role in AWS ECS
          </span>
          <span
            v-else-if="
              dataSource.authenticationType ===
              DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
            "
          >
            Bytebase will read the credential from environment variable
            <code class="code">GOOGLE_APPLICATION_CREDENTIALS</code>, fallback
            to the attached service account in GCP GCE
          </span>
        </div>
      </div>
      <div
        v-if="
          dataSource.authenticationType ===
          DataSource_AuthenticationType.AWS_RDS_IAM
        "
        class="mt-4 sm:col-span-3 sm:col-start-1"
      >
        <label for="username" class="textlabel block">
          {{ $t("instance.database-region") }}
          <span class="text-red-600">*</span>
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
        class="mt-4 sm:col-span-3 sm:col-start-1"
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
                DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED
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
          </NRadioGroup>
          <LearnMoreLink
            url="https://docs.bytebase.com/get-started/connect/overview#secret-manager-integration"
            class="text-sm"
          />
        </div>
        <div
          v-if="
            state.passwordType ===
            DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED
          "
        >
          <label class="textlabel block">
            {{ $t("common.password") }}
          </label>
          <div v-if="!hideAdvancedFeatures" class="flex space-x-2 text-sm">
            <div class="text-gray-400">
              {{ $t("instance.password-type.password-tip") }}
            </div>
            <LearnMoreLink
              url="https://docs.bytebase.com/get-started/connect/overview/#use-secret-manager?source=console"
              class="ml-1 text-sm"
            />
            <FeatureBadge
              :feature="PlanFeature.FEATURE_EXTERNAL_SECRET_MANAGER"
            />
          </div>
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
        <div v-else-if="dataSource.externalSecret" class="space-y-4">
          <div
            v-if="
              state.passwordType ===
              DataSourceExternalSecret_SecretType.VAULT_KV_V2
            "
            class="space-y-4"
          >
            <div class="sm:col-span-2 sm:col-start-1">
              <label class="textlabel block">
                {{ $t("instance.external-secret-vault.vault-url") }}
                <span class="text-red-600">*</span>
              </label>
              <BBTextField
                v-model:value="dataSource.externalSecret.url"
                :required="true"
                class="mt-2 w-full"
                :disabled="!allowEdit"
                :placeholder="$t('instance.external-secret-vault.vault-url')"
              />
            </div>
            <div class="sm:col-span-2 sm:col-start-1 space-y-2">
              <label class="textlabel block">
                {{ $t("instance.external-secret-vault.vault-auth-type.self") }}
              </label>
              <NRadioGroup
                class="textlabel mb-2"
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
                <span class="text-red-600">*</span>
              </label>
              <div class="flex space-x-2 text-sm">
                <div class="text-gray-400">
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
              class="space-y-4"
            >
              <div class="sm:col-span-2 sm:col-start-1">
                <label class="textlabel block">
                  {{
                    $t(
                      "instance.external-secret-vault.vault-auth-type.approle.role-id"
                    )
                  }}
                  <span class="text-red-600">*</span>
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
                  <span class="text-red-600">*</span>
                </label>
                <i18n-t
                  tag="div"
                  keypath="instance.external-secret-vault.vault-auth-type.approle.secret-tips"
                  class="text-gray-400 text-sm"
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
                {{
                  $t("instance.external-secret-vault.vault-secret-engine-name")
                }}
                <span class="text-red-600">*</span>
              </label>
              <div class="flex space-x-2 text-sm text-gray-400">
                {{
                  $t("instance.external-secret-vault.vault-secret-engine-tips")
                }}
              </div>
              <BBTextField
                v-model:value="dataSource.externalSecret.engineName"
                :required="true"
                class="mt-2 w-full"
                :disabled="!allowEdit"
                :placeholder="
                  $t('instance.external-secret-vault.vault-secret-engine-name')
                "
              />
            </div>
          </div>
          <div class="sm:col-span-2 sm:col-start-1">
            <label class="textlabel block">
              {{ secretNameLabel }}
              <span class="text-red-600">*</span>
            </label>
            <div
              v-if="
                state.passwordType ===
                DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER
              "
              class="flex space-x-2 text-sm text-gray-400"
            >
              {{ $t("instance.external-secret-gcp.secret-name-tips") }}
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
              DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER
            "
            class="sm:col-span-2 sm:col-start-1"
          >
            <label class="textlabel block">
              {{ secretKeyLabel }}
              <span class="text-red-600">*</span>
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

        <div
          v-if="
            basicInfo.engine === Engine.REDIS &&
            dataSource.redisType === DataSource_RedisType.SENTINEL
          "
        >
          <div class="mt-4">
            <label class="textlabel"> Master Name </label>
            <span class="text-red-600 mr-2">*</span>
            <NInput
              v-model:value="dataSource.masterName"
              class="mt-1 w-full"
              :disabled="!allowEdit"
              :placeholder="''"
            />
          </div>
          <div class="mt-4">
            <label class="textlabel"> Master Username </label>
            <NInput
              v-model:value="dataSource.masterUsername"
              class="mt-1 w-full"
              :disabled="!allowEdit"
              :placeholder="''"
            />
          </div>
          <div class="mt-4">
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
                @update:value="dataSource.updatedMasterPassword = $event.trim()"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </template>
  <template
    v-if="
      basicInfo.engine === Engine.SPANNER ||
      basicInfo.engine === Engine.BIGQUERY
    "
  >
    <div class="mt-2 sm:col-span-3 sm:col-start-1">
      <NRadioGroup
        v-model:value="dataSource.authenticationType"
        class="textlabel"
        :disabled="!allowEdit"
      >
        <NRadio :value="DataSource_AuthenticationType.PASSWORD">
          {{ $t("common.credentials") }}
        </NRadio>
        <NRadio :value="DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM">
          {{ $t("instance.password-type.google-iam") }}
        </NRadio>
      </NRadioGroup>
    </div>

    <GcpCredentialInput
      v-if="
        dataSource.authenticationType === DataSource_AuthenticationType.PASSWORD
      "
      v-model:value="dataSource.updatedPassword"
      :write-only="!isCreating"
      class="mt-4 sm:col-span-3 sm:col-start-1"
    />
  </template>

  <template v-if="basicInfo.engine === Engine.ORACLE">
    <OracleSIDAndServiceNameInput
      v-model:sid="dataSource.sid"
      v-model:service-name="dataSource.serviceName"
      :allow-edit="allowEdit"
    />
  </template>

  <template v-if="basicInfo.engine === Engine.SNOWFLAKE">
    <div class="mt-4 sm:col-span-3 sm:col-start-1">
      <div class="textlabel block">
        {{ $t("data-source.ssh.private-key") }}
      </div>
      <div class="flex space-x-2 text-sm">
        <div class="text-gray-400">
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
  </template>

  <template v-if="basicInfo.engine === Engine.DATABRICKS">
    <div>
      <div class="textlabel black mt-4">
        Warehouse ID
        <span class="text-red-600">*</span>
      </div>
      <NInput
        v-model:value="dataSource.warehouseId"
        class="mt-2"
        :disabled="!allowEdit"
      />
    </div>

    <div>
      <div class="textlabel black mt-4">
        Token
        <span class="text-red-600">*</span>
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
    <div class="mt-4 sm:col-span-3 sm:col-start-1">
      <div class="flex flex-row items-center space-x-2">
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
    <div
      v-if="hasReadonlyReplicaHost"
      class="mt-4 sm:col-span-3 sm:col-start-1"
    >
      <div class="flex flex-row items-center space-x-2">
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

    <div
      v-if="hasReadonlyReplicaPort"
      class="mt-4 sm:col-span-3 sm:col-start-1"
    >
      <div class="flex flex-row items-center space-x-2">
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

  <div v-if="showDatabase" class="mt-4 sm:col-span-3 sm:col-start-1">
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

  <div v-if="hasExtraParameters" class="mt-4 sm:col-span-3 sm:col-start-1">
    <div class="flex flex-row items-center justify-between">
      <label class="textlabel block">
        {{ $t("data-source.extra-params.self") }}
      </label>
    </div>
    <div class="text-gray-400 text-sm mt-1 mb-2">
      {{ $t("data-source.extra-params.description") }}
    </div>

    <!-- Add parameter form -->
    <div
      v-if="allowEdit"
      class="flex mt-2 mb-4 space-x-2 bg-gray-50 p-3 rounded-md"
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
      class="flex mt-2 space-x-2"
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
      class="text-gray-500 text-sm mt-2 italic"
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
    class="mt-4 sm:col-span-3 sm:col-start-1"
  >
    <div class="flex flex-row items-center gap-2">
      <NSwitch
        :value="dataSource.useSsl"
        size="small"
        @update:value="handleUseSslChanged"
      />
      <label for="ssl" class="textlabel block">
        {{ $t("data-source.ssl-connection") }}
      </label>
    </div>
    <template v-if="dataSource.useSsl">
      <template v-if="dataSource.pendingCreate">
        <SslCertificateFormV1
          :value="dataSource"
          :engine-type="basicInfo.engine"
          :disabled="!allowEdit"
          @change="handleSSLChange"
        />
      </template>
      <template v-else>
        <template v-if="dataSource.updateSsl">
          <SslCertificateFormV1
            :value="dataSource"
            :engine-type="basicInfo.engine"
            :disabled="!allowEdit"
            @change="handleSSLChange"
          />
        </template>
        <template v-else>
          <NButton
            class="!mt-2"
            :disabled="!allowEdit"
            @click.prevent="handleEditSSL"
          >
            {{ $t("common.edit") }} - {{ $t("common.write-only") }}
          </NButton>
        </template>
      </template>
    </template>
  </div>

  <div
    v-if="
      !hideAdvancedFeatures &&
      showSSH &&
      dataSource.authenticationType === DataSource_AuthenticationType.PASSWORD
    "
    class="mt-4 sm:col-span-3 sm:col-start-1"
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
</template>

<script setup lang="ts">
/* eslint-disable vue/no-mutating-props */
import { create } from "@bufbuild/protobuf";
import {
  NButton,
  NCheckbox,
  NInput,
  NRadio,
  NRadioGroup,
  NSwitch,
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
import type { DataSourceOptions } from "@/types/dataSource";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DataSource } from "@/types/proto-es/v1/instance_service_pb";
import {
  DataSourceExternalSecret_AppRoleAuthOption_SecretType,
  DataSourceExternalSecret_AuthType,
  DataSourceExternalSecret_SecretType,
  DataSourceType,
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceExternalSecretSchema,
  DataSourceExternalSecret_AppRoleAuthOptionSchema,
  DataSource_AzureCredentialSchema,
  DataSource_AWSCredentialSchema,
  DataSource_GCPCredentialSchema,
  KerberosConfigSchema,
  SASLConfigSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { onlyAllowNumber } from "@/utils";
import type { EditDataSource } from "../common";
import { useInstanceFormContext } from "../context";
import CreateDataSourceExample from "./CreateDataSourceExample.vue";
import GcpCredentialInput from "./GcpCredentialInput.vue";
import OracleSIDAndServiceNameInput from "./OracleSIDAndServiceNameInput.vue";
import SshConnectionForm from "./SshConnectionForm.vue";
import SslCertificateFormV1 from "./SslCertificateFormV1.vue";

type credentialSource = "default" | "specific-credential";

interface LocalState {
  passwordType: DataSourceExternalSecret_SecretType;
  credentialSource: credentialSource;
}

interface IAMExtensionOptions {
  label: string;
  value: credentialSource;
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
  passwordType: DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED,
  credentialSource: "default",
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
        DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED;
    }
  },
  { immediate: true, deep: true }
);

watch(
  [
    () => props.dataSource.iamExtension?.case === "azureCredential",
    () => props.dataSource.iamExtension?.case === "awsCredential",
    () => props.dataSource.iamExtension?.case === "gcpCredential",
  ],
  (credentials) => {
    if (credentials.some((c) => c === true)) {
      state.credentialSource = "specific-credential";
    } else {
      state.credentialSource = "default";
    }
  },
  { immediate: true, deep: true }
);

watch(
  () => props.dataSource.authenticationType,
  () => {
    state.credentialSource = "default";
  }
);

watch(
  () => state.credentialSource,
  (source) => {
    switch (props.dataSource.authenticationType) {
      case DataSource_AuthenticationType.AWS_RDS_IAM:
        if (source === "default") {
          props.dataSource.iamExtension = { case: undefined };
        } else {
          props.dataSource.iamExtension = {
            case: "awsCredential",
            value: create(
              DataSource_AWSCredentialSchema,
              props.dataSource.iamExtension?.case === "awsCredential"
                ? props.dataSource.iamExtension.value
                : {}
            ),
          };
        }
        break;
      case DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM:
        if (source === "default") {
          props.dataSource.iamExtension = { case: undefined };
        } else {
          props.dataSource.iamExtension = {
            case: "gcpCredential",
            value: create(
              DataSource_GCPCredentialSchema,
              props.dataSource.iamExtension?.case === "gcpCredential"
                ? props.dataSource.iamExtension.value
                : {}
            ),
          };
        }
        break;
      case DataSource_AuthenticationType.AZURE_IAM:
        if (source === "default") {
          props.dataSource.iamExtension = { case: undefined };
        } else {
          props.dataSource.iamExtension = {
            case: "azureCredential",
            value: create(
              DataSource_AzureCredentialSchema,
              props.dataSource.iamExtension?.case === "azureCredential"
                ? props.dataSource.iamExtension.value
                : {}
            ),
          };
        }
        break;
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

const secretInputPlaceholder = computed(() => {
  switch (state.passwordType) {
    case DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED:
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

const changeSecretType = (secretType: DataSourceExternalSecret_SecretType) => {
  const ds = props.dataSource;
  switch (secretType) {
    case DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED:
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

const handleSSLChange = (
  value: Partial<Pick<DataSource, "sslCa" | "sslCert" | "sslKey">>
) => {
  const ds = props.dataSource;
  Object.assign(ds, value);
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
  () => {
    const ds = props.dataSource;
    ds.updateAuthenticationPrivateKey = true;
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

// IAM Extension Options
const getIAMExtensionOptions = (
  authenticationType: DataSource_AuthenticationType
): IAMExtensionOptions[] => {
  switch (authenticationType) {
    case DataSource_AuthenticationType.AWS_RDS_IAM:
    case DataSource_AuthenticationType.AZURE_IAM:
    case DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM:
      return [
        {
          label: t("common.default"),
          value: "default",
        },
        {
          label: t("instance.iam-extension.specific-credential"),
          value: "specific-credential",
        },
      ];
  }
  return [];
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

<style lang="postcss" scoped>
.credential :deep(.code) {
  @apply bg-gray-100 p-1 rounded-sm mr-1;
}
</style>
