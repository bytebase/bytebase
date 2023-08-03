<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <template v-if="basicInfo.engine !== Engine.SPANNER">
    <CreateDataSourceExample
      class-name="sm:col-span-3 border-none mt-2"
      :create-instance-flag="isCreating"
      :engine="basicInfo.engine"
      :data-source-type="dataSource.type"
    />
    <div class="mt-2 sm:col-span-1 sm:col-start-1">
      <label for="username" class="textlabel block">
        {{ $t("common.username") }}
      </label>
      <!-- For mysql, username can be empty indicating anonymous user.
      But it's a very bad practice to use anonymous user for admin operation,
      thus we make it REQUIRED here.-->
      <input
        id="username"
        v-model="dataSource.username"
        name="username"
        type="text"
        class="textfield mt-1 w-full"
        :disabled="!allowEdit"
        :placeholder="
          basicInfo.engine === Engine.CLICKHOUSE ? $t('common.default') : ''
        "
      />
    </div>
    <div class="mt-2 sm:col-span-1 sm:col-start-1">
      <div class="flex flex-row items-center space-x-2">
        <label for="password" class="textlabel block">
          {{ $t("common.password") }}
        </label>
        <BBCheckbox
          v-if="!isCreating && allowUsingEmptyPassword"
          :title="$t('common.empty')"
          :value="dataSource.useEmptyPassword"
          :disabled="!allowEdit"
          @toggle="toggleUseEmptyPassword"
        />
      </div>
      <input
        id="password"
        name="password"
        type="text"
        class="textfield mt-1 w-full"
        autocomplete="off"
        :placeholder="
          dataSource.useEmptyPassword
            ? $t('instance.no-password')
            : $t('instance.password-write-only')
        "
        :disabled="!allowEdit || dataSource.useEmptyPassword"
        :value="dataSource.useEmptyPassword ? '' : dataSource.updatedPassword"
        @input="dataSource.updatedPassword = trimInputValue($event.target)"
      />
    </div>
  </template>
  <SpannerCredentialInput
    v-else
    v-model:value="dataSource.updatedPassword"
    :write-only="!isCreating"
    class="mt-2 sm:col-span-3 sm:col-start-1"
  />

  <template v-if="basicInfo.engine === Engine.ORACLE">
    <OracleSIDAndServiceNameInput
      v-model:sid="dataSource.sid"
      v-model:service-name="dataSource.serviceName"
      :allow-edit="allowEdit"
    />
  </template>

  <template v-if="showAuthenticationDatabase">
    <div class="sm:col-span-1 sm:col-start-1">
      <div class="flex flex-row items-center space-x-2">
        <label for="authenticationDatabase" class="textlabel block">
          {{ $t("instance.authentication-database") }}
        </label>
      </div>
      <input
        id="authenticationDatabase"
        name="authenticationDatabase"
        type="text"
        class="textfield mt-1 w-full"
        autocomplete="off"
        placeholder="admin"
        :value="dataSource.authenticationDatabase"
        @input="
          dataSource.authenticationDatabase = trimInputValue($event.target)
        "
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
      class="mt-2 sm:col-span-1 sm:col-start-1"
    >
      <div class="flex flex-row items-center space-x-2">
        <label for="host" class="textlabel block">
          {{ $t("data-source.read-replica-host") }}
        </label>
      </div>
      <input
        id="host"
        name="host"
        type="text"
        class="textfield mt-1 w-full"
        autocomplete="off"
        :value="dataSource.host"
        @input="handleHostInput"
      />
    </div>

    <div
      v-if="hasReadonlyReplicaPort"
      class="mt-2 sm:col-span-1 sm:col-start-1"
    >
      <div class="flex flex-row items-center space-x-2">
        <label for="port" class="textlabel block">
          {{ $t("data-source.read-replica-port") }}
        </label>
      </div>
      <input
        id="port"
        name="port"
        type="text"
        class="textfield mt-1 w-full"
        autocomplete="off"
        :value="dataSource.port"
        @input="handlePortInput"
      />
    </div>
  </template>

  <div v-if="showDatabase" class="mt-2 sm:col-span-1 sm:col-start-1">
    <label for="database" class="textlabel block">
      {{ $t("common.database") }}
    </label>
    <input
      id="database"
      v-model="dataSource.database"
      name="database"
      type="text"
      class="textfield mt-1 w-full"
      :disabled="!allowEdit"
      :placeholder="$t('common.database')"
    />
  </div>

  <div v-if="showSSL" class="mt-2 sm:col-span-3 sm:col-start-1">
    <div class="flex flex-row items-center">
      <label for="ssl" class="textlabel block">
        {{ $t("data-source.ssl-connection") }}
      </label>
    </div>
    <template v-if="dataSource.pendingCreate">
      <SslCertificateForm :value="dataSource" @change="handleSSLChange" />
    </template>
    <template v-else>
      <template v-if="dataSource.updateSsl">
        <SslCertificateForm :value="dataSource" @change="handleSSLChange" />
      </template>
      <template v-else>
        <button
          class="btn-normal mt-2"
          :disabled="!allowEdit"
          @click.prevent="handleEditSSL"
        >
          {{ $t("common.edit") }} - {{ $t("common.write-only") }}
        </button>
      </template>
    </template>
  </div>

  <div v-if="showSSH" class="mt-2 sm:col-span-3 sm:col-start-1">
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
        <button
          class="btn-normal mt-2"
          :disabled="!allowEdit"
          @click.prevent="handleEditSSH"
        >
          {{ $t("common.edit") }} - {{ $t("common.write-only") }}
        </button>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
/* eslint-disable vue/no-mutating-props */
import { Engine } from "@/types/proto/v1/common";
import { DataSource, DataSourceType } from "@/types/proto/v1/instance_service";
import { EditDataSource } from "../common";
import { useInstanceFormContext } from "../context";
import { useInstanceSpecs } from "../specs";
import { DataSourceOptions } from "@/types";

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
  showReadOnlyDataSourceFeatureModal,
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

const toggleUseEmptyPassword = (on: boolean) => {
  const ds = props.dataSource;
  ds.useEmptyPassword = on;
  if (on) {
    ds.updatedPassword = "";
  }
};
const handleHostInput = (event: Event) => {
  const ds = props.dataSource;
  if (ds.type === DataSourceType.READ_ONLY) {
    if (!hasReadonlyReplicaFeature.value) {
      if (ds.host || ds.port) {
        ds.host = adminDataSource.value.host;
        ds.port = adminDataSource.value.port;
        showReadOnlyDataSourceFeatureModal.value = true;
        return;
      }
    }
  }
  ds.host = trimInputValue(event.target);
};

const handlePortInput = (event: Event) => {
  const ds = props.dataSource;
  if (ds.type === DataSourceType.READ_ONLY) {
    if (!hasReadonlyReplicaFeature.value) {
      if (ds.host || ds.port) {
        ds.host = adminDataSource.value.host;
        ds.port = adminDataSource.value.port;
        showReadOnlyDataSourceFeatureModal.value = true;
        return;
      }
    }
  }
  ds.port = trimInputValue(event.target);
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

const trimInputValue = (target: Event["target"]) => {
  return ((target as HTMLInputElement)?.value ?? "").trim();
};
</script>
