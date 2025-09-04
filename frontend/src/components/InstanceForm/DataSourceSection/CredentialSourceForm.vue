<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <div class="sm:col-span-3 sm:col-start-1">
    <label for="credential-source" class="textlabel block">
      {{ $t("instance.iam-extension.credential-source") }}
    </label>
    <NRadioGroup
      v-model:value="credentialSource"
      class="textlabel mt-2"
      :disabled="!allowEdit"
    >
      <NRadio
        v-for="option in iamExtensionOptions"
        :value="option.value"
        :key="option.value"
        :label="option.label"
      />
    </NRadioGroup>
    <template v-if="credentialSource === 'specific-credential'">
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
        class="mt-2 sm:col-span-3 sm:col-start-1"
      >
        <GcpCredentialInput
          v-model:value="
            (dataSource.iamExtension?.case === 'gcpCredential'
              ? dataSource.iamExtension.value
              : {}
            ).content
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
      v-else-if="credentialSource === 'default'"
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
        <code class="code">AZURE_CLIENT_CERTIFICATE_PATH</code>, and fallback to
        attached users in Azure VM
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
        <code class="code">GOOGLE_APPLICATION_CREDENTIALS</code>, fallback to
        the attached service account in GCP GCE
      </span>
    </div>
  </div>
</template>

<script lang="tsx" setup>
/* eslint-disable vue/no-mutating-props */
import { create } from "@bufbuild/protobuf";
import { NRadioGroup, NRadio, NInput } from "naive-ui";
import { computed, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  DataSource_AuthenticationType,
  DataSource_AzureCredentialSchema,
  DataSource_AWSCredentialSchema,
  DataSource_GCPCredentialSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import type { EditDataSource } from "../common";
import GcpCredentialInput from "./GcpCredentialInput.vue";

type credentialSource = "default" | "specific-credential";

const props = defineProps<{
  allowEdit: boolean;
  dataSource: EditDataSource;
}>();

const credentialSource = ref<credentialSource>("default");

const { t } = useI18n();

const iamExtensionOptions = computed(() => {
  return [
    {
      label: t("common.Default"),
      value: "default",
    },
    {
      label: t("instance.iam-extension.specific-credential"),
      value: "specific-credential",
    },
  ];
});

watch(
  [
    () => props.dataSource.iamExtension?.case === "azureCredential",
    () => props.dataSource.iamExtension?.case === "awsCredential",
    () => props.dataSource.iamExtension?.case === "gcpCredential",
  ],
  (credentials) => {
    if (credentials.some((c) => c === true)) {
      credentialSource.value = "specific-credential";
    } else {
      credentialSource.value = "default";
    }
  },
  { immediate: true, deep: true }
);

watch(
  () => props.dataSource.authenticationType,
  () => {
    credentialSource.value = "default";
  }
);

watch(
  () => credentialSource.value,
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
</script>

<style lang="postcss" scoped>
.credential :deep(.code) {
  @apply bg-gray-100 p-1 rounded-sm mr-1;
}
</style>
