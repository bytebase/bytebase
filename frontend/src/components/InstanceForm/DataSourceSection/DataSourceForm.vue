<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <template v-if="basicInfo.engine !== Engine.SPANNER">
    <CreateDataSourceExample
      class-name="sm:col-span-3 border-none mt-2"
      :create-instance-flag="isCreating"
      :engine="basicInfo.engine"
      :data-source-type="dataSource.type"
    />
    <div class="mt-4 sm:col-span-2 sm:col-start-1">
      <label for="username" class="textlabel block">
        {{ $t("common.username") }}
      </label>
      <!-- For mysql, username can be empty indicating anonymous user.
      But it's a very bad practice to use anonymous user for admin operation,
      thus we make it REQUIRED here.-->
      <NInput
        v-model:value="dataSource.username"
        class="mt-1 w-full"
        :disabled="!allowEdit"
        :placeholder="
          basicInfo.engine === Engine.CLICKHOUSE ? $t('common.default') : ''
        "
      />
    </div>
    <div class="mt-4 sm:col-span-2 sm:col-start-1">
      <NRadioGroup
        v-if="isDev()"
        class="textlabel mb-2"
        :value="state.passwordType"
        @update:value="changePasswordType"
      >
        <NRadio value="PLAIN">
          {{ $t("instance.password-type.password") }}
        </NRadio>
        <NRadio value="EXTERNAL_SECRET">
          <div class="flex items-center gap-x-1">
            <FeatureBadge feature="bb.feature.external-secret-manager" />
            {{ $t("instance.password-type.external-secret") }}
          </div>
        </NRadio>
      </NRadioGroup>
      <div v-if="state.passwordType === 'PLAIN'">
        <label class="textlabel block">
          {{ $t("common.password") }}
        </label>
        <div class="flex space-x-2 text-sm mb-1">
          <div class="text-gray-400">
            {{ $t("instance.password-type.password-tip") }}
          </div>
          <LearnMoreLink
            url="https://www.bytebase.com/docs/get-started/instance/#use-secret-manager?source=console"
            class="ml-1 text-sm"
          />
          <FeatureBadge feature="bb.feature.external-secret-manager" />
        </div>
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
          class="w-full"
          :input-props="{ autocomplete: 'off' }"
          :placeholder="
            dataSource.useEmptyPassword
              ? $t('instance.no-password')
              : $t('instance.password-write-only')
          "
          :disabled="!allowEdit || dataSource.useEmptyPassword"
          :value="dataSource.useEmptyPassword ? '' : dataSource.updatedPassword"
          @update:value="dataSource.updatedPassword = $event.trim()"
        />
      </div>
      <div v-else-if="dataSource.externalSecret" class="space-y-4">
        <div class="sm:col-span-2 sm:col-start-1">
          <label class="textlabel block">
            {{ $t("instance.external-secret.vault-url") }}
          </label>
          <BBTextField
            v-model:value="dataSource.externalSecret.url"
            :required="true"
            class="mt-1 w-full"
            :disabled="!allowEdit"
            :placeholder="$t('instance.external-secret.vault-url')"
          />
        </div>
        <!-- App role is not enabled -->
        <!-- <div class="sm:col-span-2 sm:col-start-1 space-y-2">
          <label class="textlabel block">
            {{ $t("instance.external-secret.vault-auth-type.self") }}
          </label>
          <NRadioGroup
            class="textlabel mb-2"
            :value="dataSource.externalSecret.authType"
            @update:value="changeExternalSecretAuthType"
          >
            <NRadio :value="DataSourceExternalSecret_AuthType.TOKEN">
              {{ $t("instance.external-secret.vault-auth-type.token.self") }}
            </NRadio>
            <NRadio :value="DataSourceExternalSecret_AuthType.APP_ROLE">
              {{ $t("instance.external-secret.vault-auth-type.app-role.self") }}
            </NRadio>
          </NRadioGroup>
        </div> -->
        <div
          v-if="
            dataSource.externalSecret.authType ===
            DataSourceExternalSecret_AuthType.TOKEN
          "
          class="sm:col-span-2 sm:col-start-1"
        >
          <label class="textlabel block">
            {{ $t("instance.external-secret.vault-auth-type.token.api-token") }}
          </label>
          <div class="flex space-x-2 text-sm mb-1">
            <div class="text-gray-400">
              {{ $t("instance.external-secret.vault-auth-type.token.tips") }}
            </div>
            <LearnMoreLink
              url="https://developer.hashicorp.com/vault/tutorials/operations/generate-root"
              class="ml-1 text-sm"
            />
          </div>
          <BBTextField
            :value="dataSource.externalSecret.token ?? ''"
            class="mt-1 w-full"
            :disabled="!allowEdit"
            :placeholder="secretInputPlaceholder"
            @update:value="
              (val: string) => {
                const ds = dataSource;
                ds.externalSecret!.token = val;
              }
            "
          />
        </div>
        <div v-else-if="dataSource.externalSecret.appRole" class="space-y-4">
          <div class="sm:col-span-2 sm:col-start-1">
            <label class="textlabel block">
              {{
                $t("instance.external-secret.vault-auth-type.app-role.role-id")
              }}
            </label>
            <BBTextField
              :value="dataSource.externalSecret.appRole.roleId"
              :required="true"
              class="mt-1 w-full"
              :disabled="!allowEdit"
              :placeholder="`${$t(
                'instance.external-secret.vault-auth-type.app-role.role-id'
              )} - ${$t('common.write-only')}`"
              @update:value="
                (val: string) => {
                  const ds = dataSource;
                  ds.externalSecret!.appRole!.roleId = val;
                }
              "
            />
          </div>
          <div class="sm:col-span-2 sm:col-start-1">
            <label class="textlabel block">
              {{
                $t(
                  "instance.external-secret.vault-auth-type.app-role.secret-id"
                )
              }}
            </label>
            <i18n-t
              tag="div"
              keypath="instance.external-secret.vault-auth-type.app-role.secret-tips"
              class="text-gray-400 text-sm mb-1"
            >
              <template #learn_more>
                <LearnMoreLink
                  url="https://developer.hashicorp.com/vault/tutorials/auth-methods/approle"
                  class="ml-1 text-sm"
                />
              </template>
            </i18n-t>
            <NRadioGroup
              v-model:value="dataSource.externalSecret.appRole.type"
              class="textlabel my-1"
            >
              <NRadio
                :value="
                  DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN
                "
              >
                {{
                  $t(
                    "instance.external-secret.vault-auth-type.app-role.secret-plain-text"
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
                    "instance.external-secret.vault-auth-type.app-role.secret-env-name"
                  )
                }}
              </NRadio>
            </NRadioGroup>
            <BBTextField
              :value="dataSource.externalSecret.appRole.secretId"
              class="mt-1 w-full"
              :disabled="!allowEdit"
              :placeholder="secretInputPlaceholder"
              @update:value="
                (val: string) => {
                  const ds = dataSource;
                  ds.externalSecret!.appRole!.secretId = val;
                }
              "
            />
          </div>
        </div>
        <div class="sm:col-span-2 sm:col-start-1">
          <label class="textlabel block">
            {{ $t("instance.external-secret.vault-secret-engine-name") }}
          </label>
          <div class="flex space-x-2 text-sm mb-1 text-gray-400">
            {{ $t("instance.external-secret.vault-secret-engine-tips") }}
          </div>
          <BBTextField
            v-model:value="dataSource.externalSecret.engineName"
            :required="true"
            class="mt-1 w-full"
            :disabled="!allowEdit"
            :placeholder="
              $t('instance.external-secret.vault-secret-engine-name')
            "
          />
        </div>
        <div class="sm:col-span-2 sm:col-start-1">
          <label class="textlabel block">
            {{ $t("instance.external-secret.vault-secret-name") }}
          </label>
          <BBTextField
            v-model:value="dataSource.externalSecret.secretName"
            :required="true"
            class="mt-1 w-full"
            :disabled="!allowEdit"
            :placeholder="$t('instance.external-secret.vault-secret-name')"
          />
        </div>
        <div class="sm:col-span-2 sm:col-start-1">
          <label class="textlabel block">
            {{ $t("instance.external-secret.vault-key-name") }}
          </label>
          <BBTextField
            v-model:value="dataSource.externalSecret.passwordKeyName"
            :required="true"
            class="mt-1 w-full"
            :disabled="!allowEdit"
            :placeholder="$t('instance.external-secret.vault-key-name')"
          />
        </div>
      </div>
    </div>
  </template>
  <SpannerCredentialInput
    v-else
    v-model:value="dataSource.updatedPassword"
    :write-only="!isCreating"
    class="mt-4 sm:col-span-3 sm:col-start-1"
  />

  <template v-if="basicInfo.engine === Engine.ORACLE">
    <OracleSIDAndServiceNameInput
      v-model:sid="dataSource.sid"
      v-model:service-name="dataSource.serviceName"
      :allow-edit="allowEdit"
    />
  </template>

  <template v-if="basicInfo.engine === Engine.SNOWFLAKE">
    <div class="mt-4 sm:col-span-2 sm:col-start-1">
      <div class="textlabel block">
        {{ $t("data-source.ssh.private-key") }}
      </div>
      <div class="flex space-x-2 text-sm mb-1">
        <div class="text-gray-400">
          {{ $t("data-source.snowflake-keypair-tip") }}
        </div>
        <LearnMoreLink
          url="https://docs.snowflake.com/en/user-guide/key-pair-auth"
          class="ml-1 text-sm"
        />
      </div>
      <div>
        <DroppableTextarea
          v-model:value="dataSource.authenticationPrivateKey"
          :resizable="false"
          class="w-full h-32 whitespace-pre-wrap"
          placeholder="-----BEGIN PRIVATE KEY-----
MIIEvQ...
-----END PRIVATE KEY-----"
          :allow-edit="allowEdit"
        />
      </div>
    </div>
  </template>

  <template v-if="showAuthenticationDatabase">
    <div class="sm:col-span-2 sm:col-start-1">
      <div class="flex flex-row items-center space-x-2">
        <label for="authenticationDatabase" class="textlabel block">
          {{ $t("instance.authentication-database") }}
        </label>
      </div>
      <NInput
        class="mt-1 w-full"
        :input-props="{ autocomplete: 'off' }"
        placeholder="admin"
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
      class="mt-4 sm:col-span-2 sm:col-start-1"
    >
      <div class="flex flex-row items-center space-x-2">
        <label for="host" class="textlabel block">
          {{ $t("data-source.read-replica-host") }}
        </label>
      </div>
      <NInput
        class="mt-1 w-full"
        :input-props="{ autocomplete: 'off' }"
        :value="dataSource.host"
        @update:value="handleHostInput"
      />
    </div>

    <div
      v-if="hasReadonlyReplicaPort"
      class="mt-4 sm:col-span-2 sm:col-start-1"
    >
      <div class="flex flex-row items-center space-x-2">
        <label for="port" class="textlabel block">
          {{ $t("data-source.read-replica-port") }}
        </label>
      </div>
      <NInput
        class="mt-1 w-full"
        :input-props="{ autocomplete: 'off' }"
        :value="dataSource.port"
        :allow-input="onlyAllowNumber"
        @update:value="handlePortInput"
      />
    </div>
  </template>

  <div v-if="showDatabase" class="mt-4 sm:col-span-2 sm:col-start-1">
    <label for="database" class="textlabel block">
      {{ $t("common.database") }}
    </label>
    <NInput
      v-model:value="dataSource.database"
      class="mt-1 w-full"
      :disabled="!allowEdit"
      :placeholder="$t('common.database')"
    />
  </div>

  <div v-if="showSSL" class="mt-4 sm:col-span-3 sm:col-start-1">
    <div class="flex flex-row items-center">
      <label for="ssl" class="textlabel block">
        {{ $t("data-source.ssl-connection") }}
      </label>
    </div>
    <template v-if="dataSource.pendingCreate">
      <SslCertificateForm
        :value="dataSource"
        :engine-type="basicInfo.engine"
        @change="handleSSLChange"
      />
    </template>
    <template v-else>
      <template v-if="dataSource.updateSsl">
        <SslCertificateForm
          :value="dataSource"
          :engine-type="basicInfo.engine"
          @change="handleSSLChange"
        />
      </template>
      <template v-else>
        <NButton
          class="!mt-4"
          :disabled="!allowEdit"
          @click.prevent="handleEditSSL"
        >
          {{ $t("common.edit") }} - {{ $t("common.write-only") }}
        </NButton>
      </template>
    </template>
  </div>

  <div v-if="showSSH" class="mt-4 sm:col-span-3 sm:col-start-1">
    <div class="flex flex-row items-center gap-x-1">
      <label for="ssh" class="textlabel block">
        {{ $t("data-source.ssh-connection") }}
      </label>
      <FeatureBadge
        feature="bb.feature.instance-ssh-connection"
        :instance="instance"
      />
    </div>
    <template v-if="dataSource.pendingCreate">
      <SshConnectionForm
        :value="dataSource"
        :instance="instance"
        @change="handleSSHChange"
      />
    </template>
    <template v-else>
      <template v-if="dataSource.updateSsh">
        <SshConnectionForm
          :value="dataSource"
          :instance="instance"
          @change="handleSSHChange"
        />
      </template>
      <template v-else>
        <NButton
          class="!mt-4"
          :disabled="!allowEdit"
          @click.prevent="handleEditSSH"
        >
          {{ $t("common.edit") }} - {{ $t("common.write-only") }}
        </NButton>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
/* eslint-disable vue/no-mutating-props */
import { NButton, NRadioGroup, NRadio, NCheckbox, NInput } from "naive-ui";
import { watch, reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import type { DataSourceOptions } from "@/types/dataSource";
import { Engine } from "@/types/proto/v1/common";
import type { DataSource } from "@/types/proto/v1/instance_service";
import {
  DataSourceType,
  DataSourceExternalSecret,
  DataSourceExternalSecret_AuthType,
  DataSourceExternalSecret_SecretType,
  DataSourceExternalSecret_AppRoleAuthOption_SecretType,
} from "@/types/proto/v1/instance_service";
import { onlyAllowNumber, isDev } from "@/utils";
import type { EditDataSource } from "../common";
import { useInstanceFormContext } from "../context";
import { useInstanceSpecs } from "../specs";
import SpannerCredentialInput from "./SpannerCredentialInput.vue";
import SshConnectionForm from "./SshConnectionForm.vue";
import SslCertificateForm from "./SslCertificateForm.vue";

interface LocalState {
  passwordType: "PLAIN" | "EXTERNAL_SECRET";
}

const props = defineProps<{
  dataSource: EditDataSource;
}>();

const {
  instance,
  isCreating,
  allowEdit,
  basicInfo,
  adminDataSource,
  hasReadonlyReplicaFeature,
  missingFeature,
} = useInstanceFormContext();

const {
  showDatabase,
  showSSL,
  showSSH,
  allowUsingEmptyPassword,
  showAuthenticationDatabase,
  hasReadonlyReplicaHost,
  hasReadonlyReplicaPort,
} = useInstanceSpecs();

const state = reactive<LocalState>({
  passwordType: "PLAIN",
});
const { t } = useI18n();

watch(
  () => props.dataSource.externalSecret,
  (externalSecret) => {
    if (externalSecret) {
      state.passwordType = "EXTERNAL_SECRET";
    } else {
      state.passwordType = "PLAIN";
    }
  },
  { immediate: true, deep: true }
);

const secretInputPlaceholder = computed(() => {
  switch (state.passwordType) {
    case "PLAIN":
      return `${t("common.password")} - ${t("common.write-only")}`;
    case "EXTERNAL_SECRET":
      switch (props.dataSource.externalSecret?.authType) {
        case DataSourceExternalSecret_AuthType.TOKEN:
          return `${t(
            "instance.external-secret.vault-auth-type.token.api-token"
          )} - ${t("common.write-only")}`;
        case DataSourceExternalSecret_AuthType.APP_ROLE:
          switch (props.dataSource.externalSecret.appRole?.type) {
            case DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN:
              return `${t(
                "instance.external-secret.vault-auth-type.app-role.secret-id-plain-text"
              )} - ${t("common.write-only")}`;
            case DataSourceExternalSecret_AppRoleAuthOption_SecretType.ENVIRONMENT:
              return `${t(
                "instance.external-secret.vault-auth-type.app-role.secret-id-environment"
              )} - ${t("common.write-only")}`;
          }
      }
  }

  return "";
});

const changePasswordType = (passwordType: "PLAIN" | "EXTERNAL_SECRET") => {
  const ds = props.dataSource;
  if (passwordType === "EXTERNAL_SECRET") {
    ds.externalSecret = DataSourceExternalSecret.fromPartial(
      ds.externalSecret ?? {
        authType: DataSourceExternalSecret_AuthType.TOKEN,
        secretType: DataSourceExternalSecret_SecretType.VAULT_KV_V2,
        token: "",
      }
    );
  } else {
    ds.externalSecret = undefined;
  }
  state.passwordType = passwordType;
};

// TODO: support change auth type.
// const changeExternalSecretAuthType = (
//   authType: DataSourceExternalSecret_AuthType
// ) => {
//   const ds = props.dataSource;
//   if (!ds.externalSecret) {
//     return;
//   }
//   if (authType === DataSourceExternalSecret_AuthType.APP_ROLE) {
//     ds.externalSecret.appRole =
//       DataSourceExternalSecret_AppRoleAuthOption.fromPartial({
//         type: DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN,
//       });
//     ds.externalSecret.token = undefined;
//   } else {
//     ds.externalSecret.token = "";
//     ds.externalSecret.appRole = undefined;
//   }
//   ds.externalSecret.authType = authType;
// };

const toggleUseEmptyPassword = (on: boolean) => {
  const ds = props.dataSource;
  ds.useEmptyPassword = on;
  if (on) {
    ds.updatedPassword = "";
  }
};
const handleHostInput = (value: string) => {
  const ds = props.dataSource;
  if (ds.type === DataSourceType.READ_ONLY) {
    if (!hasReadonlyReplicaFeature.value) {
      if (ds.host || ds.port) {
        ds.host = adminDataSource.value.host;
        ds.port = adminDataSource.value.port;
        missingFeature.value = "bb.feature.read-replica-connection";
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
        missingFeature.value = "bb.feature.read-replica-connection";
        return;
      }
    }
  }
  ds.port = value.trim();
};
const handleEditSSL = () => {
  const ds = props.dataSource;
  ds.sslCa = "";
  ds.sslCert = "";
  ds.sslKey = "";
  ds.updateSsl = true;
};

const handleEditSSH = () => {
  const ds = props.dataSource;
  if (!ds) return;
  ds.sshHost = "";
  ds.sshPort = "";
  ds.sshUser = "";
  ds.sshPassword = "";
  ds.sshPrivateKey = "";
  ds.updateSsh = true;
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
  ds.updateSsh = true;
};

watch(
  () => props.dataSource.authenticationPrivateKey,
  () => {
    const ds = props.dataSource;
    ds.updateAuthenticationPrivateKey = true;
  }
);
</script>
