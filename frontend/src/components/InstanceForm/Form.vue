<template>
  <div class="flex flex-col gap-y-6 pb-2">
    <div class="w-full flex flex-col gap-y-6">
      <div
        v-if="isCreating"
        class="rounded-lg border border-block-border bg-white"
      >
        <button
          type="button"
          class="w-full flex items-center justify-between gap-x-3 px-4 py-3 text-left transition-colors hover:bg-gray-50"
          @click="toggleEngineSelector"
        >
          <div class="min-w-0">
            <p
              class="text-[11px] font-medium uppercase tracking-[0.14em] text-control-light"
            >
              {{ $t("database.engine") }}
            </p>
            <div class="mt-1 flex items-center gap-x-1.5">
              <RichEngineName
                :engine="basicInfo.engine"
                class="text-sm font-medium text-main"
              />
              <NTag
                v-if="isEngineBeta(basicInfo.engine)"
                round
                size="small"
                type="info"
              >
                Beta
              </NTag>
            </div>
          </div>
          <div class="shrink-0 text-control-light">
            <ChevronDownIcon
              v-if="!isEngineSelectorCollapsed"
              class="w-4 h-4"
            />
            <ChevronRightIcon v-else class="w-4 h-4" />
          </div>
        </button>

        <Transition name="engine-selector">
          <div
            v-show="!isEngineSelectorCollapsed"
            class="border-t border-block-border px-4 py-4"
          >
            <InstanceEngineRadioGrid
              :engine="basicInfo.engine"
              :engine-list="supportedEngineV1List()"
              class="w-full grid-cols-2 sm:grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-2"
              @update:engine="
                (newEngine: Engine) => handleSelectInstanceEngine(newEngine)
              "
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
          </div>
        </Transition>
      </div>

      <!-- Basic Info Card -->
      <div class="border border-block-border rounded-lg p-5">
        <h3 class="text-base font-medium text-main">
          {{ $t("instance.section.basic-info") }}
        </h3>

        <div class="mt-3 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
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
              class="mt-1 w-full max-w-[40rem]"
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
                :to="autoSubscriptionRoute()"
                class="accent-link"
              >
                {{
                  $t("subscription.instance-assignment.n-license-remain", {
                    n: availableLicenseCountText,
                  })
                }}</router-link
              >)
            </label>
            <div class="h-8.5 flex flex-row items-center mt-1">
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
              class="mt-1 w-full max-w-[40rem]"
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

          <!--Do not show external link on create to reduce cognitive load-->
          <div v-if="!isCreating" class="sm:col-span-3 sm:col-start-1">
            <label for="external-link" class="textlabel inline-flex">
              <span>
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
        </div>
      </div>

      <!-- Connection Card -->
      <div class="border border-block-border rounded-lg p-5">
        <h3 class="text-base font-medium text-main">
          {{ $t("instance.section.connection") }}
        </h3>

        <div class="mt-3 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
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
                  <InfoTrigger
                    v-if="isCreating && infoPanel && hasHostInfo"
                    @click="openInfoPanel('host')"
                  />
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
            <div
              class="mt-1 grid grid-cols-1 gap-y-1 gap-x-4 sm:grid-cols-12"
            >
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
                <div class="h-8.5 flex flex-row items-center self-end">
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
        </div>

        <!-- Credentials (auth method, username, password) -->
        <template v-if="basicInfo.engine !== Engine.DYNAMODB">
          <DataSourceSection hide-options />

          <BBAttention
            v-if="actuatorStore.isSaaSMode"
            class="mt-4 border-none"
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
        </template>
      </div>

      <!-- Connection Options Card (Extra Params, SSL, SSH) -->
      <div
        v-if="basicInfo.engine !== Engine.DYNAMODB && editingDataSource"
        class="border border-block-border rounded-lg bg-white"
      >
        <button
          type="button"
          class="w-full flex items-center justify-between gap-x-3 px-5 py-4 text-left transition-colors hover:bg-gray-50"
          @click="toggleConnectionOptions"
        >
          <h3 class="text-base font-medium text-main">
            {{ $t("instance.connection-options") }}
          </h3>
          <div class="shrink-0 text-control-light">
            <ChevronDownIcon
              v-if="!isConnectionOptionsCollapsed"
              class="w-4 h-4"
            />
            <ChevronRightIcon v-else class="w-4 h-4" />
          </div>
        </button>
        <Transition name="connection-options">
          <div
            v-show="!isConnectionOptionsCollapsed"
            class="border-t border-block-border px-5 py-4"
          >
            <DataSourceForm :data-source="editingDataSource" options-only />
          </div>
        </Transition>
      </div>

      <div
        v-if="
          isCreating &&
          !!editingDataSource
        "
        class="flex justify-start"
      >
        <NButton
          tertiary
          type="primary"
          :loading="state.isTestingConnection"
          :disabled="!allowTestConnection"
          @click.prevent="testConnectionForCurrentEditingDS"
        >
          {{ $t("instance.test-connection") }}
        </NButton>
      </div>

      <!-- Sync Databases Card (create only) -->
      <div
        v-if="basicInfo.engine !== Engine.DYNAMODB && isCreating"
        class="border border-block-border rounded-lg p-5 flex flex-col gap-y-1"
      >
        <p class="w-full text-lg leading-6 font-medium text-gray-900">
          {{ $t("instance.sync-databases.self") }}
        </p>

        <SyncDatabases
          class="mt-2"
          :is-creating="true"
          :show-label="false"
          :allow-edit="allowEdit && !!allowCreate"
          :sync-databases="basicInfo.syncDatabases"
          @update:sync-databases="handleChangeSyncDatabases"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { ChevronDownIcon, ChevronRightIcon, TrashIcon } from "lucide-vue-next";
import {
  NButton,
  NCheckbox,
  NInput,
  NRadio,
  NRadioGroup,
  NSwitch,
  NTag,
} from "naive-ui";
import { computed, inject, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBAttention } from "@/bbkit";
import { LabelListEditor } from "@/components/Label";
import RequiredStar from "@/components/RequiredStar.vue";
import {
  EnvironmentSelect,
  InstanceEngineRadioGrid,
  InstanceV1EngineIcon,
  MiniActionButton,
  RichEngineName,
} from "@/components/v2";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
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
  isValidSpannerHost,
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
import DataSourceForm from "./DataSourceSection/DataSourceForm.vue";
import DataSourceSection from "./DataSourceSection/DataSourceSection.vue";
import InfoTrigger from "./InfoTrigger.vue";
import { hasInfoContent, type InfoSection } from "./info-content";
import ScanIntervalInput from "./ScanIntervalInput.vue";
import SpannerHostInput from "./SpannerHostInput.vue";
import SyncDatabases from "./SyncDatabases.vue";

const context = useInstanceFormContext();
const {
  events,
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
  checkDataSource,
  testConnection,
} = context;
const { isEngineBeta, defaultPort, instanceLink, allowEditPort } = specs;

const infoPanel = inject<
  | {
      open: (section: InfoSection) => void;
      close: () => void;
      setEngine: (engine: Engine) => void;
    }
  | undefined
>("infoPanel", undefined);

watch(
  () => basicInfo.value.engine,
  (engine) => {
    infoPanel?.setEngine(engine);
  },
  { immediate: true }
);

const openInfoPanel = (section: InfoSection) => {
  if (!hasInfoContent(basicInfo.value.engine, section)) {
    return;
  }
  infoPanel?.open(section);
};

const hasHostInfo = computed(() =>
  hasInfoContent(basicInfo.value.engine, "host")
);

const { t } = useI18n();
const instanceV1Store = useInstanceV1Store();
const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();
const scanIntervalInputRef = ref<InstanceType<typeof ScanIntervalInput>>();
const isEngineSelectorCollapsed = ref(false);
const isConnectionOptionsCollapsed = ref(true);

const showConnectionOptionsCard = computed(() => {
  return (
    basicInfo.value.engine !== Engine.DYNAMODB && !!editingDataSource.value
  );
});

const hasConfiguredConnectionOptions = computed(() => {
  const ds = editingDataSource.value;
  if (!ds) {
    return false;
  }

  const hasExtraParameters =
    Object.keys(ds.extraConnectionParameters ?? {}).length > 0;
  const hasSslConfig = !!(ds.useSsl || ds.sslCa || ds.sslCert || ds.sslKey);
  const hasSshConfig = !!(
    ds.sshHost ||
    ds.sshPort ||
    ds.sshUser ||
    ds.sshPassword ||
    ds.sshPrivateKey
  );

  return hasExtraParameters || hasSslConfig || hasSshConfig;
});

watch(
  () => ({
    show: showConnectionOptionsCard.value,
    creating: isCreating.value,
    hasConfigured: hasConfiguredConnectionOptions.value,
  }),
  (next, prev) => {
    if (!next.show) {
      return;
    }

    const becameVisible = !prev?.show;
    if (becameVisible) {
      isConnectionOptionsCollapsed.value = next.creating
        ? true
        : !next.hasConfigured;
      return;
    }

    if (!next.creating && next.hasConfigured && !prev?.hasConfigured) {
      isConnectionOptionsCollapsed.value = false;
    }
  },
  { immediate: true }
);

useEmitteryEventListener(events, "show-connection-options", () => {
  if (!showConnectionOptionsCard.value) {
    return;
  }
  isConnectionOptionsCollapsed.value = false;
});

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

const handleSelectInstanceEngine = (engine: Engine) => {
  changeInstanceEngine(engine);
  isEngineSelectorCollapsed.value = true;
};

const toggleEngineSelector = () => {
  isEngineSelectorCollapsed.value = !isEngineSelectorCollapsed.value;
};

const toggleConnectionOptions = () => {
  isConnectionOptionsCollapsed.value = !isConnectionOptionsCollapsed.value;
};

const allowTestConnection = computed(() => {
  if (
    !allowEdit.value ||
    state.value.isRequesting ||
    state.value.isTestingConnection
  ) {
    return false;
  }

  const ds = editingDataSource.value;
  if (!ds) {
    return false;
  }

  if (basicInfo.value.engine === Engine.SPANNER) {
    return isValidSpannerHost(ds.host);
  }
  if (basicInfo.value.engine === Engine.BIGQUERY) {
    return ds.host !== "";
  }
  if (basicInfo.value.engine !== Engine.DYNAMODB && ds.host === "") {
    return false;
  }

  return checkDataSource([ds]);
});

const testConnectionForCurrentEditingDS = async () => {
  const ds = editingDataSource.value;
  if (!ds) {
    return;
  }

  const result = await testConnection(ds, /* !silent */ false);
  if (!result.success && hasConfiguredConnectionOptions.value) {
    events.emit("show-connection-options");
  }
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
</script>

<style lang="postcss" scoped>
.engine-selector-enter-active,
.engine-selector-leave-active {
  overflow: hidden;
  transition: max-height 0.2s ease, opacity 0.2s ease, transform 0.2s ease;
}

.engine-selector-enter-from,
.engine-selector-leave-to {
  max-height: 0;
  opacity: 0;
  transform: translateY(-0.25rem);
}

.engine-selector-enter-to,
.engine-selector-leave-from {
  max-height: 32rem;
  opacity: 1;
  transform: translateY(0);
}

.connection-options-enter-active,
.connection-options-leave-active {
  overflow: hidden;
  transition: max-height 0.2s ease, opacity 0.2s ease, transform 0.2s ease;
}

.connection-options-enter-from,
.connection-options-leave-to {
  max-height: 0;
  opacity: 0;
  transform: translateY(-0.25rem);
}

.connection-options-enter-to,
.connection-options-leave-from {
  max-height: 80rem;
  opacity: 1;
  transform: translateY(0);
}
</style>
