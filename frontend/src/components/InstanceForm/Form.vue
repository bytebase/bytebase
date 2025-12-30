<template>
  <div class="flex flex-col gap-y-6 pb-2">
    <div class="max-w-[850px]">
      <InstanceEngineRadioGrid
        v-if="isCreating"
        :engine="basicInfo.engine"
        :engine-list="supportedEngineV1List()"
        class="w-full grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2"
        @update:engine="(newEngine: Engine) => changeInstanceEngine(newEngine)"
      >
        <template #suffix="{ engine }">
          <NTag
            v-if="isEngineBeta(engine as Engine)"
            round
            size="small"
            type="info"
          >
            Beta
          </NTag>
        </template>
      </InstanceEngineRadioGrid>

      <NDivider />

      <!-- Instance Name -->
      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="name" class="textlabel flex flex-row items-center">
            {{ $t("instance.instance-name") }}
            <RequiredStar class="ml-0.5" />
            <div v-if="instance" class="ml-2 flex items-center">
              <InstanceV1EngineIcon :instance="instance" :tooltip="false" />
              <span class="ml-1">{{ instance.engineVersion }}</span>
            </div>
          </label>
          <NInput
            v-model:value="basicInfo.title"
            required
            class="mt-1 w-full"
            :disabled="!allowEdit"
            :maxlength="200"
          />
        </div>

        <div
          v-if="subscriptionStore.currentPlan !== PlanType.FREE && allowEdit"
          class="sm:col-span-2 ml-0 sm:ml-3"
        >
          <label for="activation" class="textlabel block">
            {{ $t("subscription.instance-assignment.assign-license") }}
            (<router-link
              :to="autoSubscriptionRoute($router)"
              class="accent-link"
            >
              {{
                $t("subscription.instance-assignment.n-license-remain", {
                  n: availableLicenseCountText,
                })
              }}</router-link
            >)
          </label>
          <div class="h-[34px] flex flex-row items-center mt-1">
            <NSwitch
              :value="basicInfo.activation"
              :disabled="!basicInfo.activation && availableLicenseCount === 0"
              @update:value="changeInstanceActivation"
            />
          </div>
        </div>

        <div
          :key="basicInfo.environment"
          class="sm:col-span-3 sm:col-start-1 -mt-4"
        >
          <ResourceIdField
            ref="resourceIdField"
            v-model:value="resourceId"
            class="max-w-full flex-nowrap"
            editing-class="mt-4"
            resource-type="instance"
            :readonly="!isCreating"
            :resource-title="basicInfo.title"
            :fetch-resource="
              (id) =>
                instanceV1Store.getOrFetchInstanceByName(
                  `${instanceNamePrefix}${id}`,
                  true /* silent */
                )
            "
          />
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="environment" class="textlabel">
            {{ $t("common.environment") }}
          </label>
          <EnvironmentSelect
            class="mt-1 w-full"
            required="true"
            :value="
              isValidEnvironmentName(
                `${environmentNamePrefix}${environment.id}`
              )
                ? `${environmentNamePrefix}${environment.id}`
                : undefined
            "
            :disabled="!allowEdit"
            @update:value="handleSelectEnvironment($event as (string | undefined))"
          />
        </div>

        <div class="sm:col-span-3 sm:col-start-1">
          <template v-if="basicInfo.engine === Engine.SPANNER">
            <SpannerHostInput
              v-model:host="adminDataSource.host"
              :allow-edit="allowEdit"
            />
          </template>
          <template v-else-if="basicInfo.engine === Engine.BIGQUERY">
            <BigQueryHostInput
              v-model:host="adminDataSource.host"
              :allow-edit="allowEdit"
            />
          </template>
          <template v-else>
            <label for="host" class="textlabel block">
              <template v-if="basicInfo.engine === Engine.SNOWFLAKE">
                {{ $t("instance.account-locator") }}
                <RequiredStar class="mr-2" />
                <LearnMoreLink
                  url="https://docs.snowflake.com/en/user-guide/admin-account-identifier#using-an-account-locator-as-an-identifier"
                  class="text-sm"
                />
              </template>
              <template v-else-if="basicInfo.engine === Engine.COSMOSDB">
                {{ $t("instance.endpoint") }}
                <RequiredStar />
              </template>
              <div
                v-else-if="
                  adminDataSource.authenticationType ===
                  DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
                "
              >
                <span>
                  {{ $t("instance.sentence.google-cloud-sql.instance-name") }}
                  <RequiredStar />
                </span>
                <i18n-t
                  tag="div"
                  class="textinfolabel mb-1"
                  keypath="instance.sentence.google-cloud-sql.instance-name-tips"
                >
                  <template #instance>
                    <span class="font-bold">
                      {project-id}:{region}:{instance-name}
                    </span>
                  </template>
                </i18n-t>
              </div>
              <template v-else>
                {{ $t("instance.host-or-socket") }}
                <RequiredStar v-if="basicInfo.engine !== Engine.DYNAMODB" />
              </template>
            </label>
            <NInput
              v-model:value="adminDataSource.host"
              required
              :placeholder="
                basicInfo.engine === Engine.SNOWFLAKE
                  ? $t('instance.your-snowflake-account-locator')
                  : $t('instance.sentence.host.none-snowflake')
              "
              class="mt-1 w-full"
              :disabled="!allowEdit"
            />
            <div
              v-if="basicInfo.engine === Engine.SNOWFLAKE"
              class="mt-2 textinfolabel"
            >
              {{ $t("instance.sentence.proxy.snowflake") }}
            </div>
          </template>
        </div>

        <template
          v-if="
            basicInfo.engine !== Engine.SPANNER &&
            basicInfo.engine !== Engine.BIGQUERY &&
            basicInfo.engine !== Engine.DATABRICKS &&
            basicInfo.engine !== Engine.COSMOSDB &&
            adminDataSource.authenticationType !==
              DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
          "
        >
          <div class="sm:col-span-1">
            <label for="port" class="textlabel block">
              {{ $t("instance.port") }}
            </label>
            <NInput
              v-model:value="adminDataSource.port"
              class="mt-1 w-full"
              :placeholder="defaultPort"
              :disabled="!allowEdit || !allowEditPort"
              :allow-input="onlyAllowNumber"
            />
          </div>
        </template>

        <div
          v-if="basicInfo.engine === Engine.MONGODB"
          class="sm:col-span-4 sm:col-start-1"
        >
          <label
            for="connectionStringSchema"
            class="textlabel flex flex-row items-center"
          >
            {{ $t("data-source.connection-string-schema") }}
          </label>
          <NRadioGroup
            :value="currentMongoDBConnectionSchema"
            @update:value="handleMongodbConnectionStringSchemaChange"
          >
            <NRadio
              v-for="type in MongoDBConnectionStringSchemaList"
              :key="type"
              :value="type"
            >
              {{ type }}
            </NRadio>
          </NRadioGroup>
        </div>

        <div
          v-if="basicInfo.engine === Engine.REDIS"
          class="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2"
        >
          <label
            for="connectionStringSchema"
            class="textlabel flex flex-row items-center"
          >
            {{ $t("data-source.connection-type") }}
          </label>
          <NRadioGroup
            :value="currentRedisConnectionType"
            @update:value="handleRedisConnectionTypeChange"
          >
            <NRadio
              v-for="type in RedisConnectionType"
              :key="type"
              :value="type"
            >
              {{ type }}
            </NRadio>
          </NRadioGroup>
        </div>

        <div
          v-if="showAdditionalAddresses"
          class="sm:col-span-4 sm:col-start-1"
        >
          <label
            for="additionalAddresses"
            class="textlabel flex flex-row items-center"
          >
            {{ $t("data-source.additional-node-addresses") }}
          </label>
          <div class="mt-1 grid grid-cols-1 gap-y-1 gap-x-4 sm:grid-cols-12">
            <template
              v-for="(_, index) in adminDataSource.additionalAddresses"
              :key="index"
            >
              <div class="sm:col-span-8 sm:col-start-1">
                <label
                  v-if="index === 0"
                  for="additionalAddressesHost"
                  class="textlabel font-normal! flex flex-row items-center"
                >
                  {{ $t("instance.host-or-socket") }}
                </label>
                <NInput
                  v-model:value="
                    adminDataSource.additionalAddresses[index].host
                  "
                  required
                  class="mt-1 w-full"
                  :disabled="!allowEdit"
                />
              </div>
              <div class="sm:col-span-3">
                <label
                  v-if="index === 0"
                  for="additionalAddressesPort"
                  class="textlabel font-normal! flex flex-row items-center"
                >
                  {{ $t("instance.port") }}
                </label>
                <NInput
                  v-model:value="
                    adminDataSource.additionalAddresses[index].port
                  "
                  class="mt-1 w-full"
                  :placeholder="defaultPort"
                  :disabled="!allowEdit || !allowEditPort"
                  :allow-input="onlyAllowNumber"
                />
              </div>
              <div class="h-[34px] flex flex-row items-center self-end">
                <MiniActionButton
                  :disabled="!allowEdit"
                  @click.stop="removeDSAdditionalAddress(index)"
                >
                  <TrashIcon class="w-4 h-4" />
                </MiniActionButton>
              </div>
            </template>
            <div class="mt-1 sm:col-span-12 sm:col-start-1">
              <NButton
                class="ml-auto w-12!"
                size="small"
                @click.prevent="addDSAdditionalAddress"
              >
                {{ $t("common.add") }}
              </NButton>
            </div>
          </div>
        </div>

        <div
          v-if="basicInfo.engine === Engine.MONGODB && !adminDataSource.srv"
          class="sm:col-span-2 sm:col-start-1"
        >
          <label for="replicaSet" class="textlabel">
            {{ $t("data-source.replica-set") }}
          </label>
          <NInput
            v-model:value="adminDataSource.replicaSet"
            required
            class="mt-1 w-full"
            :disabled="!allowEdit"
          />
        </div>

        <div
          v-if="
            basicInfo.engine === Engine.MONGODB &&
            !adminDataSource.srv &&
            adminDataSource.additionalAddresses.length === 0
          "
          class="sm:col-span-4 sm:col-start-1"
        >
          <NCheckbox
            :checked="adminDataSource.directConnection"
            :disabled="!allowEdit"
            style="--n-label-padding: 0 0 0 1rem"
            @update:checked="
              (on: boolean) => {
                adminDataSource.directConnection = on;
              }
            "
          >
            {{ $t("data-source.direct-connection") }}
          </NCheckbox>
        </div>

        <ScanIntervalInput
          v-if="!isCreating && instance"
          ref="scanIntervalInputRef"
          :scan-interval="basicInfo.syncInterval"
          :allow-edit="allowEdit"
          :instance="instance"
          @update:scan-interval="changeScanInterval"
        />

        <SyncDatabases
          v-if="!isCreating"
          :is-creating="false"
          :show-label="true"
          :allow-edit="allowEdit"
          :sync-databases="basicInfo.syncDatabases"
          @update:sync-databases="handleChangeSyncDatabases"
        />

        <!--Do not show external link on create to reduce cognitive load-->
        <div v-if="!isCreating" class="sm:col-span-3 sm:col-start-1">
          <label for="external-link" class="textlabel inline-flex">
            <span class>
              {{
                basicInfo.engine === Engine.SNOWFLAKE
                  ? $t("instance.snowflake-web-console")
                  : $t("instance.external-link")
              }}
            </span>
            <button
              v-if="instanceLink.trim().length > 0"
              class="ml-1 btn-icon"
              @click.prevent="window.open(urlfy(instanceLink), '_blank')"
            >
              <heroicons-outline:external-link class="w-4 h-4" />
            </button>
          </label>
          <template v-if="basicInfo.engine === Engine.SNOWFLAKE">
            <NInput
              required
              class="mt-1 w-full"
              :disabled="true"
              :value="instanceLink"
            />
          </template>
          <template v-else>
            <div class="mt-1 textinfolabel">
              {{ $t("instance.sentence.console.snowflake") }}
            </div>
            <NInput
              v-model:value="basicInfo.externalLink"
              required
              class="textfield mt-1 w-full"
              :disabled="!allowEdit"
              :placeholder="SnowflakeExtraLinkPlaceHolder"
            />
          </template>
        </div>

        <!-- Labels -->
        <div class="sm:col-span-3 sm:col-start-1">
          <label for="labels" class="textlabel">
            {{ $t("common.labels") }}
          </label>
          <div class="mt-1">
            <LabelListEditor
              ref="labelListEditorRef"
              v-model:kv-list="labelKVList"
              :readonly="!allowEdit"
              :show-errors="true"
            />
          </div>
        </div>
      </div>

      <NDivider />

      <!-- Connection Info -->
      <template v-if="basicInfo.engine !== Engine.DYNAMODB">
        <p class="w-full text-lg leading-6 font-medium text-gray-900">
          {{ $t("instance.connection-info") }}
        </p>

        <DataSourceSection />
      </template>

      <BBAttention
        v-if="actuatorStore.isSaaSMode"
        class="my-4 border-none"
        type="info"
      >
        <a
          href="https://docs.bytebase.com/get-started/cloud#prerequisites"
          target="_blank"
          rel="noopener noreferrer"
          class="normal-link"
        >
          {{ $t("instance.sentence.firewall-info") }}
        </a>
      </BBAttention>

      <div class="mt-6 pt-0 border-none">
        <div class="flex flex-row gap-x-2">
          <NButton
            tertiary
            type="primary"
            class="whitespace-nowrap flex items-center"
            :loading="state.isTestingConnection"
            :disabled="!allowCreate || state.isRequesting || !allowEdit"
            @click.prevent="testConnectionForCurrentEditingDS"
          >
            <span>{{ $t("instance.test-connection") }}</span>
          </NButton>
        </div>
      </div>

      <NDivider />

      <div
        v-if="basicInfo.engine !== Engine.DYNAMODB && isCreating"
        class="flex flex-col gap-y-1"
      >
        <p class="w-full text-lg leading-6 font-medium text-gray-900">
          {{ $t("instance.sync-databases.self") }}
        </p>

        <SyncDatabases
          :is-creating="true"
          :show-label="false"
          :allow-edit="allowEdit && !!allowCreate"
          :sync-databases="basicInfo.syncDatabases"
          @update:sync-databases="handleChangeSyncDatabases"
        />
      </div>
    </div>

    <InstanceArchiveRestoreButton
      v-if="!hideArchiveRestore && !isCreating && instance"
      :instance="instance"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { TrashIcon } from "lucide-vue-next";
import {
  NButton,
  NCheckbox,
  NDivider,
  NInput,
  NRadio,
  NRadioGroup,
  NSwitch,
  NTag,
} from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention } from "@/bbkit";
import { InstanceArchiveRestoreButton } from "@/components/Instance";
import { LabelListEditor } from "@/components/Label";
import RequiredStar from "@/components/RequiredStar.vue";
import {
  EnvironmentSelect,
  InstanceEngineRadioGrid,
  InstanceV1EngineIcon,
  MiniActionButton,
} from "@/components/v2";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import {
  pushNotification,
  useActuatorV1Store,
  useDatabaseV1Store,
  useInstanceV1Store,
  useSubscriptionV1Store,
} from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { isValidEnvironmentName, UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AddressSchema,
  DataSource_AuthenticationType,
  DataSource_RedisType,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import {
  autoSubscriptionRoute,
  extractInstanceResourceName,
  isDev,
  onlyAllowNumber,
  supportedEngineV1List,
  urlfy,
} from "@/utils";
import LearnMoreLink from "../LearnMoreLink.vue";
import BigQueryHostInput from "./BigQueryHostInput.vue";
import {
  MongoDBConnectionStringSchemaList,
  RedisConnectionType,
  SnowflakeExtraLinkPlaceHolder,
} from "./constants";
import { useInstanceFormContext } from "./context";
import DataSourceSection from "./DataSourceSection/DataSourceSection.vue";
import ScanIntervalInput from "./ScanIntervalInput.vue";
import SpannerHostInput from "./SpannerHostInput.vue";
import SyncDatabases from "./SyncDatabases.vue";

defineProps<{
  hideArchiveRestore?: boolean;
}>();

const context = useInstanceFormContext();
const {
  state,
  specs,
  instance,
  environment,
  isCreating,
  allowEdit,
  allowCreate,
  resourceIdField,
  basicInfo,
  labelListEditorRef,
  labelKVList,
  adminDataSource,
  editingDataSource,
  testConnection,
} = context;
const { isEngineBeta, defaultPort, instanceLink, allowEditPort } = specs;

const { t } = useI18n();
const instanceV1Store = useInstanceV1Store();
const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();
const scanIntervalInputRef = ref<InstanceType<typeof ScanIntervalInput>>();

const availableLicenseCount = computed(() => {
  return Math.max(
    0,
    subscriptionStore.instanceLicenseCount -
      actuatorStore.activatedInstanceCount
  );
});

const availableLicenseCountText = computed((): string => {
  if (subscriptionStore.instanceLicenseCount === Number.MAX_VALUE) {
    return t("common.unlimited");
  }
  return `${availableLicenseCount.value}`;
});

const resourceId = computed({
  get() {
    const id = extractInstanceResourceName(basicInfo.value.name);
    if (id === String(UNKNOWN_ID)) return "";
    return id;
  },
  set(id) {
    basicInfo.value.name = `instances/${id}`;
  },
});

const currentMongoDBConnectionSchema = computed(() => {
  return adminDataSource.value.srv === false
    ? MongoDBConnectionStringSchemaList[0]
    : MongoDBConnectionStringSchemaList[1];
});

const currentRedisConnectionType = computed(() => {
  switch (adminDataSource.value.redisType) {
    case DataSource_RedisType.STANDALONE:
      return RedisConnectionType[0];
    case DataSource_RedisType.SENTINEL:
      return RedisConnectionType[1];
    case DataSource_RedisType.CLUSTER:
      return RedisConnectionType[2];
    default:
      return RedisConnectionType[0];
  }
});

const showAdditionalAddresses = computed(() => {
  if (basicInfo.value.engine === Engine.CASSANDRA) {
    return true;
  }
  if (basicInfo.value.engine === Engine.MONGODB && !adminDataSource.value.srv) {
    return true;
  }
  if (
    basicInfo.value.engine === Engine.REDIS &&
    (adminDataSource.value.redisType === DataSource_RedisType.CLUSTER ||
      adminDataSource.value.redisType === DataSource_RedisType.SENTINEL)
  ) {
    return true;
  }
  return false;
});

// The default host name is 127.0.0.1 or host.docker.internal which is not applicable to Snowflake, so we change
// the host name between 127.0.0.1/host.docker.internal and "" if user hasn't changed default yet.
const changeInstanceEngine = (engine: Engine) => {
  context.resetDataSource();
  switch (engine) {
    case Engine.SNOWFLAKE:
    case Engine.DYNAMODB: {
      if (
        adminDataSource.value.host === "127.0.0.1" ||
        adminDataSource.value.host === "host.docker.internal"
      ) {
        adminDataSource.value.host = "";
      }
      break;
    }
    case Engine.SPANNER:
    case Engine.BIGQUERY: {
      adminDataSource.value.authenticationType =
        DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM;
      if (
        adminDataSource.value.host === "127.0.0.1" ||
        adminDataSource.value.host === "host.docker.internal"
      ) {
        adminDataSource.value.host = "";
      }
      break;
    }
    case Engine.COSMOSDB: {
      // Cosmos DB supports Azure IAM only.
      adminDataSource.value.authenticationType =
        DataSource_AuthenticationType.AZURE_IAM;
      break;
    }
    default: {
      if (!adminDataSource.value.host) {
        adminDataSource.value.host = isDev()
          ? "127.0.0.1"
          : "host.docker.internal";
      }
      break;
    }
  }
  basicInfo.value.engine = engine;
};

const handleChangeSyncDatabases = (databases: string[]) => {
  basicInfo.value.syncDatabases = [...databases];
};

const changeScanInterval = (duration: Duration | undefined) => {
  basicInfo.value.syncInterval = duration;
};

const handleRedisConnectionTypeChange = (type: string) => {
  const ds = editingDataSource.value;
  if (!ds) return;
  switch (type) {
    case RedisConnectionType[0]:
      ds.redisType = DataSource_RedisType.STANDALONE;
      break;
    case RedisConnectionType[1]:
      ds.redisType = DataSource_RedisType.SENTINEL;
      break;
    case RedisConnectionType[2]:
      ds.redisType = DataSource_RedisType.CLUSTER;
      break;
    default:
      ds.redisType = DataSource_RedisType.STANDALONE;
      break;
  }
};

const handleMongodbConnectionStringSchemaChange = (type: string) => {
  const ds = editingDataSource.value;
  if (!ds) return;
  switch (type) {
    case MongoDBConnectionStringSchemaList[0]:
      ds.srv = false;
      break;
    case MongoDBConnectionStringSchemaList[1]:
      // MongoDB doesn't support specify port if using srv record.
      ds.port = "";
      ds.additionalAddresses = [];
      ds.replicaSet = "";
      ds.directConnection = false;
      ds.srv = true;
      break;
    default:
      ds.srv = false;
  }
};

const removeDSAdditionalAddress = (i: number) => {
  adminDataSource.value.additionalAddresses.splice(i, 1);
  if (adminDataSource.value.additionalAddresses.length === 0) {
    adminDataSource.value.directConnection = false;
  }
};

const addDSAdditionalAddress = () => {
  editingDataSource.value?.additionalAddresses.push(
    create(DataSource_AddressSchema, {
      host: "",
      port: "",
    })
  );
  if (adminDataSource.value.additionalAddresses.length !== 0) {
    adminDataSource.value.directConnection = false;
  }
};

const changeInstanceActivation = async (on: boolean) => {
  basicInfo.value.activation = on;
  if (instance.value) {
    const instancePatch = {
      ...instance.value,
      activation: on,
    };
    const updated = await instanceV1Store.updateInstance(instancePatch, [
      "activation",
    ]);
    useDatabaseV1Store().updateDatabaseInstance(updated);
    // refresh activatedInstanceCount
    await actuatorStore.fetchServerInfo();

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
};

const handleSelectEnvironment = (name: string | undefined) => {
  basicInfo.value.environment = name;
};

const testConnectionForCurrentEditingDS = () => {
  const editingDS = editingDataSource.value;
  if (!editingDS) return;
  testConnection(editingDS, /* !silent */ false);
};
</script>

<style lang="postcss" scoped>
.instance-engine-button :deep(.n-button__content) {
  width: 100%;
  justify-content: flex-start;
}
</style>
