<template>
  <div class="space-y-6 divide-y divide-block-border">
    <div class="divide-y divide-block-border w-[850px]">
      <div v-if="isCreating" class="w-full mt-4 mb-6 grid grid-cols-4 gap-2">
        <template v-for="engine in engineList" :key="engine">
          <div
            class="flex relative justify-start p-2 border rounded cursor-pointer hover:bg-control-bg-hover"
            :class="
              basicInformation.engine === engine &&
              'font-medium bg-control-bg-hover'
            "
            @click.capture="changeInstanceEngine(engine)"
          >
            <div class="flex flex-row justify-start items-center">
              <input
                type="radio"
                class="btn mr-2"
                :checked="basicInformation.engine == engine"
              />
              <img
                class="w-5 h-auto max-h-[20px] object-contain mr-1"
                :src="EngineIconPath[engine]"
              />
              <p class="text-center text-sm">
                {{ engineName(engine) }}
              </p>
              <template v-if="isEngineBeta(engine)">
                <BBBetaBadge
                  class="absolute -top-px -right-px rounded text-xs !bg-gray-500 px-1 !py-0"
                />
              </template>
            </div>
          </div>
        </template>
      </div>

      <!-- Instance Name -->
      <div class="pt-4 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="name" class="textlabel flex flex-row items-center">
            {{ $t("instance.instance-name") }}
            <span class="text-red-600 mr-2">*</span>
            <template v-if="props.instance">
              <InstanceEngineIcon :instance="props.instance" />
              <span class="ml-1">{{ props.instance.engineVersion }}</span>
            </template>
          </label>
          <input
            id="name"
            required
            name="name"
            type="text"
            class="textfield mt-1 w-full"
            :disabled="!allowEdit"
            :value="basicInformation.name"
            @input="handleInstanceNameInput"
          />
        </div>

        <div
          :key="basicInformation.environmentId"
          class="sm:col-span-3 sm:col-start-1 -mt-4"
        >
          <ResourceIdField
            ref="resourceIdField"
            class="max-w-full flex-nowrap"
            resource-type="instance"
            :readonly="!isCreating"
            :value="basicInformation.resourceId"
            :resource-title="basicInformation.name"
            :validate="validateResourceId"
          />
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="environment" class="textlabel">
            {{ $t("common.environment") }}
          </label>
          <EnvironmentSelect
            id="environment"
            class="mt-1 w-full"
            name="environment"
            :disabled="!isCreating"
            :selected-id="basicInformation.environmentId"
            @select-environment-id="
              (environmentId) => {
                basicInformation.environmentId = environmentId;
              }
            "
          />
        </div>

        <div class="sm:col-span-3 sm:col-start-1">
          <template v-if="basicInformation.engine !== 'SPANNER'">
            <label for="host" class="textlabel block">
              <template v-if="basicInformation.engine == 'SNOWFLAKE'">
                {{ $t("instance.account-name") }}
                <span class="text-red-600 mr-2">*</span>
              </template>
              <template v-else>
                {{ $t("instance.host-or-socket") }}
                <span class="text-red-600 mr-2">*</span>
              </template>
            </label>
            <input
              id="host"
              required
              type="text"
              name="host"
              :placeholder="
                basicInformation.engine == 'SNOWFLAKE'
                  ? $t('instance.your-snowflake-account-name')
                  : $t('instance.sentence.host.snowflake')
              "
              class="textfield mt-1 w-full"
              :disabled="!allowEdit"
              :value="adminDataSource.host"
              @input="handleInstanceHostInput"
            />
            <div
              v-if="basicInformation.engine == 'SNOWFLAKE'"
              class="mt-2 textinfolabel"
            >
              {{ $t("instance.sentence.proxy.snowflake") }}
            </div>
          </template>
          <SpannerHostInput
            v-else
            :host="adminDataSource.host"
            :allow-edit="allowEdit"
            @update:host="handleUpdateSpannerHost"
          />
        </div>

        <template v-if="basicInformation.engine !== 'SPANNER'">
          <div class="sm:col-span-1">
            <label for="port" class="textlabel block">{{
              $t("instance.port")
            }}</label>
            <input
              id="port"
              type="number"
              name="port"
              class="textfield mt-1 w-full"
              :placeholder="defaultPort"
              :disabled="!allowEdit || !allowEditPort"
              :value="adminDataSource.port"
              @wheel="handleInstancePortWheelScroll"
              @input="handleInstancePortInput"
            />
          </div>
        </template>

        <div
          v-if="basicInformation.engine === 'MONGODB'"
          class="sm:col-span-4 sm:col-start-1"
        >
          <label
            for="connectionStringSchema"
            class="textlabel flex flex-row items-center"
          >
            {{ $t("data-source.connection-string-schema") }}
          </label>
          <label
            v-for="type in mongodbConnectionStringSchemaList"
            :key="type"
            class="radio h-7"
          >
            <input
              type="radio"
              class="btn"
              name="connectionStringSchema"
              :value="type"
              :checked="type === currentMongoDBConnectionSchema"
              @change="handleMongodbConnectionStringSchemaChange"
            />
            <span class="label">
              {{ type }}
            </span>
          </label>
        </div>

        <!--Do not show external link on create to reduce cognitive load-->
        <div v-if="!isCreating" class="sm:col-span-3 sm:col-start-1">
          <label for="externallink" class="textlabel inline-flex">
            <span class>
              {{
                basicInformation.engine == "SNOWFLAKE"
                  ? $t("instance.snowflake-web-console")
                  : $t("instance.external-link")
              }}
            </span>
            <button
              class="ml-1 btn-icon"
              :disabled="instanceLink.trim().length == 0"
              @click.prevent="window.open(urlfy(instanceLink), '_blank')"
            >
              <heroicons-outline:external-link class="w-4 h-4" />
            </button>
          </label>
          <template v-if="basicInformation.engine == 'SNOWFLAKE'">
            <input
              id="externallink"
              required
              name="externallink"
              type="text"
              class="textfield mt-1 w-full"
              disabled="true"
              :value="instanceLink"
            />
          </template>
          <template v-else>
            <div class="mt-1 textinfolabel">
              {{ $t("instance.sentence.console.snowflake") }}
            </div>
            <input
              id="externallink"
              required
              name="externallink"
              type="text"
              :disabled="!allowEdit"
              :value="basicInformation.externalLink"
              class="textfield mt-1 w-full"
              :placeholder="snowflakeExtraLinkPlaceHolder"
              @input="handleInstanceExternalLinkInput"
            />
          </template>
        </div>
      </div>

      <!-- Connection Info -->
      <p class="mt-6 pt-4 w-full text-lg leading-6 font-medium text-gray-900">
        {{ $t("instance.connection-info") }}
      </p>
      <div
        v-if="!isCreating && !hasReadOnlyDataSource"
        class="mt-2 flex flex-row justify-start items-center bg-yellow-50 border-none rounded-lg p-2 px-3"
      >
        <heroicons-outline:exclamation
          class="h-6 w-6 text-yellow-400 flex-shrink-0 mr-1"
        />
        <span class="text-yellow-800 text-sm">{{
          $t("instance.no-read-only-data-source-warn")
        }}</span>
        <button
          type="button"
          class="btn-normal ml-4 text-sm"
          @click.prevent="handleCreateRODataSource"
        >
          {{ $t("common.create") }}
        </button>
      </div>

      <div
        class="mt-2 grid grid-cols-1 gap-y-2 gap-x-4 border-none sm:grid-cols-3"
      >
        <NTabs
          v-if="!isCreating"
          class="sm:col-span-3"
          type="line"
          :value="state.currentDataSourceType"
          @update:value="handleDataSourceTypeChange"
        >
          <NTab name="ADMIN">Admin</NTab>
          <NTab name="RO" class="relative" :disabled="!hasReadOnlyDataSource">
            <span>Read only</span>
            <BBButtonConfirm
              v-if="hasReadOnlyDataSource"
              :style="'DELETE'"
              class="absolute left-full ml-1"
              :require-confirm="readonlyDataSource?.id !== UNKNOWN_ID"
              :ok-text="$t('common.delete')"
              :confirm-title="
                $t('data-source.delete-read-only-data-source') + '?'
              "
              @confirm="handleDeleteRODataSource"
            />
          </NTab>
        </NTabs>

        <template v-if="basicInformation.engine !== 'SPANNER'">
          <CreateDataSourceExample
            class-name="sm:col-span-3 border-none mt-2"
            :create-instance-flag="isCreating"
            :engine-type="basicInformation.engine"
            :data-source-type="state.currentDataSourceType"
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
              name="username"
              type="text"
              class="textfield mt-1 w-full"
              :disabled="!allowEdit"
              :placeholder="
                basicInformation.engine == 'CLICKHOUSE'
                  ? $t('common.default')
                  : ''
              "
              :value="currentDataSource.username"
              @input="handleCurrentDataSourceNameInput"
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
                :value="currentDataSource.useEmptyPassword"
                @toggle="handleToggleUseEmptyPassword"
              />
            </div>
            <input
              id="password"
              name="password"
              type="text"
              class="textfield mt-1 w-full"
              autocomplete="off"
              :placeholder="
                currentDataSource.useEmptyPassword
                  ? $t('instance.no-password')
                  : $t('instance.password-write-only')
              "
              :disabled="!allowEdit || currentDataSource.useEmptyPassword"
              :value="
                currentDataSource.useEmptyPassword
                  ? ''
                  : currentDataSource.updatedPassword
              "
              @input="handleCurrentDataSourcePasswordInput"
            />
          </div>
        </template>
        <SpannerCredentialInput
          v-else
          :value="currentDataSource.updatedPassword"
          :write-only="!isCreating"
          class="mt-2 sm:col-span-3 sm:col-start-1"
          @update:value="handleUpdateSpannerCredential"
        />

        <template v-if="basicInformation.engine === 'ORACLE'">
          <OracleSIDAndServiceNameInput
            v-model:sid="currentDataSource.options.sid"
            v-model:service-name="currentDataSource.options.serviceName"
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
              :value="currentDataSource.options.authenticationDatabase"
              @input="handleInstanceAuthenticationDatabaseInput"
            />
          </div>
        </template>

        <template
          v-if="
            state.currentDataSourceType === 'RO' &&
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
              <span class="textinfolabel">({{ $t("common.optional") }})</span>
            </div>
            <input
              id="host"
              name="host"
              type="text"
              class="textfield mt-1 w-full"
              autocomplete="off"
              :value="currentDataSource.host"
              @input="handleCurrentDataSourceHostInput"
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
              <span class="textinfolabel">({{ $t("common.optional") }})</span>
            </div>
            <input
              id="port"
              name="port"
              type="text"
              class="textfield mt-1 w-full"
              autocomplete="off"
              :value="currentDataSource.port"
              @input="handleCurrentDataSourcePortInput"
            />
          </div>
        </template>

        <div v-if="showDatabase" class="mt-2 sm:col-span-1 sm:col-start-1">
          <label for="database" class="textlabel block">
            {{ $t("common.database") }}
          </label>
          <input
            id="database"
            v-model="currentDataSource.database"
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
          <template v-if="currentDataSource.id === UNKNOWN_ID">
            <SslCertificateForm
              :value="currentDataSource"
              @change="handleCurrentDataSourceSslChange"
            />
          </template>
          <template v-else>
            <template v-if="currentDataSource.updateSsl">
              <SslCertificateForm
                :value="currentDataSource"
                @change="handleCurrentDataSourceSslChange"
              />
            </template>
            <template v-else>
              <button class="btn-normal mt-2" @click.prevent="handleEditSsl">
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
              class="text-accent"
            />
          </div>
          <SshConnectionForm
            :value="currentDataSource.options"
            @change="handleCurrentDataSourceSshChange"
          />
        </div>
      </div>

      <BBAttention
        v-if="outboundIpList && actuatorStore.isSaaSMode"
        class="my-5 border-none"
        :style="'INFO'"
        :title="$t('instance.sentence.outbound-ip-list')"
        :description="outboundIpList"
      />

      <div class="mt-6 pt-0 border-none">
        <div class="flex flex-row space-x-2">
          <button
            type="button"
            class="btn-normal whitespace-nowrap items-center"
            :disabled="!allowCreate || state.isRequesting"
            @click.prevent="testConnection"
          >
            <BBSpin v-if="state.isTestingConnection" />
            {{ $t("instance.test-connection") }}
          </button>
        </div>
      </div>
    </div>

    <!-- Action Button Group -->
    <div class="pt-4">
      <div class="w-full flex justify-between items-center">
        <div class="w-full flex justify-end items-center">
          <div>
            <BBSpin v-if="state.isRequesting" />
          </div>
          <div class="ml-2">
            <template v-if="isCreating">
              <button
                type="button"
                class="btn-normal py-2 px-4"
                :disabled="state.isRequesting"
                @click.prevent="cancel"
              >
                {{ $t("common.cancel") }}
              </button>
              <button
                type="button"
                class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
                :disabled="!allowCreate || state.isRequesting"
                @click.prevent="tryCreate"
              >
                {{ $t("common.create") }}
              </button>
            </template>
            <template v-else>
              <button
                v-if="allowEdit"
                type="button"
                :disabled="!allowUpdate || state.isRequesting"
                :class="
                  !allowUpdate || state.isRequesting
                    ? 'btn-normal'
                    : 'btn-primary'
                "
                @click.prevent="doUpdate"
              >
                {{ $t("common.update") }}
              </button>
            </template>
          </div>
        </div>
      </div>
    </div>
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.read-replica-connection"
    @cancel="state.showFeatureModal = false"
  />

  <BBAlert
    v-if="state.showCreateInstanceWarningModal"
    :style="'WARN'"
    :ok-text="t('instance.ignore-and-create')"
    :title="$t('instance.connection-info-seems-to-be-incorrect')"
    :description="state.createInstanceWarning"
    :progress-text="$t('common.creating')"
    @ok="handleWarningModalOkClick"
    @cancel="state.showCreateInstanceWarningModal = false"
  ></BBAlert>
</template>

<script lang="ts" setup>
import { computed, reactive, PropType, ref, watch, onMounted } from "vue";
import { cloneDeep, isEqual, isEmpty, omit } from "lodash-es";
import { useI18n } from "vue-i18n";
import { Status } from "nice-grpc-common";
import { useRouter } from "vue-router";

import {
  hasWorkspacePermission,
  instanceHasSSL,
  instanceHasSSH,
  instanceSlug,
  isDev,
  isValidSpannerHost,
  supportedEngineList,
} from "../utils";
import {
  InstancePatch,
  DataSourceType,
  Instance,
  ConnectionInfo,
  DataSource,
  UNKNOWN_ID,
  DataSourceCreate,
  DataSourcePatch,
  EngineType,
  engineName,
  InstanceId,
  ResourceId,
  RowStatus,
  InstanceCreate,
  unknown,
  ValidatedMessage,
  DataSourceOptions,
} from "../types";
import {
  hasFeature,
  pushNotification,
  useCurrentUser,
  useDatabaseStore,
  useDataSourceStore,
  useInstanceStore,
  useActuatorStore,
  useSQLStore,
} from "@/store";
import { getErrorCode } from "@/utils/grpcweb";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";
import FeatureBadge from "./FeatureBadge.vue";
import {
  SpannerHostInput,
  SpannerCredentialInput,
  SslCertificateForm,
  SshConnectionForm,
  OracleSIDAndServiceNameInput,
} from "./InstanceForm";
import { useInstanceV1Store } from "@/store/modules/v1/instance";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { useSettingV1Store } from "@/store/modules/v1/setting";

const props = defineProps({
  instance: {
    type: Object as PropType<Instance>,
    default: undefined,
  },
});

const emit = defineEmits(["dismiss"]);

const cancel = () => {
  emit("dismiss");
};

interface EditDataSource extends DataSource {
  updatedPassword: string;
  useEmptyPassword: boolean;
}

interface BasicInformation {
  id: InstanceId;
  resourceId: ResourceId;
  rowStatus: RowStatus;
  name: string;
  engine: EngineType;
  externalLink?: string;
  environmentId: number;
}

interface LocalState {
  currentDataSourceType: DataSourceType;
  showFeatureModal: boolean;
  isTestingConnection: boolean;
  isRequesting: boolean;
  showCreateInstanceWarningModal: boolean;
  createInstanceWarning: string;
}

const { t } = useI18n();
const router = useRouter();
const instanceStore = useInstanceStore();
const instanceV1Store = useInstanceV1Store();
const dataSourceStore = useDataSourceStore();
const currentUser = useCurrentUser();
const sqlStore = useSQLStore();
const settingV1Store = useSettingV1Store();
const actuatorStore = useActuatorStore();

const state = reactive<LocalState>({
  currentDataSourceType: "ADMIN",
  showFeatureModal: false,
  isTestingConnection: false,
  isRequesting: false,
  showCreateInstanceWarningModal: false,
  createInstanceWarning: "",
});

const basicInformation = ref<BasicInformation>({
  id: props.instance?.id || UNKNOWN_ID,
  resourceId: props.instance?.resourceId || "",
  rowStatus: props.instance?.rowStatus || "NORMAL",
  name: props.instance?.name || t("instance.new-instance"),
  engine: props.instance?.engine || "MYSQL",
  environmentId: (props.instance?.environment.id || UNKNOWN_ID) as number,
});

const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

const getDataSourceWithType = (type: DataSourceType) =>
  props.instance?.dataSourceList.find((ds) => ds.type === type);

// We only support one admin data source and one read-only data source.
const adminDataSource = ref<EditDataSource>({
  ...cloneDeep(getDataSourceWithType("ADMIN") || unknown("DATA_SOURCE")),
  updatedPassword: "",
  useEmptyPassword: false,
});

const readonlyDataSource = ref<EditDataSource | undefined>(
  getDataSourceWithType("RO")
    ? ({
        ...cloneDeep(getDataSourceWithType("RO")),
        updatedPassword: "",
        useEmptyPassword: false,
      } as EditDataSource)
    : undefined
);

const getDefaultPort = (engine: EngineType) => {
  if (engine == "CLICKHOUSE") {
    return "9000";
  } else if (engine == "POSTGRES") {
    return "5432";
  } else if (engine == "SNOWFLAKE") {
    return "443";
  } else if (engine == "TIDB") {
    return "4000";
  } else if (engine == "MONGODB") {
    return "27017";
  } else if (engine === "REDIS") {
    return "6379";
  } else if (engine === "ORACLE") {
    return "1521";
  } else if (engine === "MSSQL") {
    return "1433";
  } else if (engine === "REDSHIFT") {
    return "5439";
  } else if (engine === "OCEANBASE") {
    return "2883";
  }
  return "3306";
};

const isCreating = computed(() => props.instance === undefined);

onMounted(async () => {
  if (isCreating.value) {
    adminDataSource.value.host = isDev() ? "127.0.0.1" : "host.docker.internal";
    adminDataSource.value.options.srv = false;
    adminDataSource.value.options.authenticationDatabase = "";
  }
  await settingV1Store.fetchSettingList();
});

watch(
  () => basicInformation.value.engine,
  () => {
    if (isCreating.value) {
      adminDataSource.value.port = getDefaultPort(
        basicInformation.value.engine
      );
    }
  },
  {
    immediate: true,
  }
);

const engineList = computed(() => {
  return supportedEngineList();
});

const outboundIpList = computed(() => {
  if (!settingV1Store.workspaceProfileSetting) {
    return "";
  }
  return settingV1Store.workspaceProfileSetting.outboundIpList.join(",");
});

const EngineIconPath = {
  MYSQL: new URL("../assets/db-mysql.png", import.meta.url).href,
  POSTGRES: new URL("../assets/db-postgres.png", import.meta.url).href,
  TIDB: new URL("../assets/db-tidb.png", import.meta.url).href,
  SNOWFLAKE: new URL("../assets/db-snowflake.png", import.meta.url).href,
  CLICKHOUSE: new URL("../assets/db-clickhouse.png", import.meta.url).href,
  MONGODB: new URL("../assets/db-mongodb.png", import.meta.url).href,
  SPANNER: new URL("../assets/db-spanner.png", import.meta.url).href,
  REDIS: new URL("../assets/db-redis.png", import.meta.url).href,
  ORACLE: new URL("../assets/db-oracle.svg", import.meta.url).href,
  MSSQL: new URL("../assets/db-mssql.svg", import.meta.url).href,
  REDSHIFT: new URL("../assets/db-redshift.svg", import.meta.url).href,
  MARIADB: new URL("../assets/db-mariadb.png", import.meta.url).href,
  OCEANBASE: new URL("../assets/db-oceanbase.png", import.meta.url).href,
};

const mongodbConnectionStringSchemaList = ["mongodb://", "mongodb+srv://"];

const currentMongoDBConnectionSchema = computed(() => {
  return adminDataSource.value.options.srv === false
    ? mongodbConnectionStringSchemaList[0]
    : mongodbConnectionStringSchemaList[1];
});

const allowCreate = computed(() => {
  if (basicInformation.value.engine === "SPANNER") {
    return (
      basicInformation.value.name.trim() &&
      isValidSpannerHost(adminDataSource.value.host) &&
      adminDataSource.value.updatedPassword
    );
  }

  return (
    basicInformation.value.name.trim() &&
    resourceIdField.value?.resourceId &&
    resourceIdField.value?.isValidated &&
    adminDataSource.value.host
  );
});

const allowEdit = computed(() => {
  return (
    basicInformation.value.rowStatus == "NORMAL" &&
    hasWorkspacePermission(
      "bb.permission.workspace.manage-instance",
      currentUser.value.role
    )
  );
});

const allowEditPort = computed(() => {
  // MongoDB doesn't support specify port if using srv record.
  return !(
    basicInformation.value.engine === "MONGODB" &&
    currentDataSource.value.options.srv
  );
});

const allowUsingEmptyPassword = computed(() => {
  return basicInformation.value.engine !== "SPANNER";
});

const valueChanged = computed(() => {
  return !isEqual(
    {
      basicInformation: basicInformation.value,
      adminDataSource: adminDataSource.value,
      readonlyDataSource: readonlyDataSource.value,
    },
    getInstanceStateData()
  );
});

const connectionInfoChanged = computed(() => {
  if (!valueChanged.value) {
    return false;
  }

  return (
    !isEqual(adminDataSource.value, getInstanceStateData().adminDataSource) ||
    !isEqual(
      readonlyDataSource.value,
      getInstanceStateData().readonlyDataSource
    )
  );
});

const defaultPort = computed(() => {
  if (basicInformation.value.engine == "CLICKHOUSE") {
    return "9000";
  } else if (basicInformation.value.engine == "POSTGRES") {
    return "5432";
  } else if (basicInformation.value.engine == "SNOWFLAKE") {
    return "443";
  } else if (basicInformation.value.engine == "TIDB") {
    return "4000";
  } else if (basicInformation.value.engine == "MONGODB") {
    // MongoDB doesn't support specify port if using srv record.
    if (currentDataSource.value.options.srv) {
      return "";
    }
    return "27017";
  } else if (basicInformation.value.engine == "REDSHIFT") {
    return "5439";
  } else if (basicInformation.value.engine == "MARIADB") {
    return "3306";
  } else if (basicInformation.value.engine == "OCEANBASE") {
    return "2883";
  }
  return "3306";
});

const currentDataSource = computed(() => {
  if (state.currentDataSourceType === "ADMIN") {
    return adminDataSource.value;
  } else if (state.currentDataSourceType === "RO") {
    return readonlyDataSource.value as EditDataSource;
  } else {
    throw new Error("Unknown data source type");
  }
});

const hasReadOnlyDataSource = computed(() => {
  return readonlyDataSource.value !== undefined;
});

const snowflakeExtraLinkPlaceHolder =
  "https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=mysql-instance-foo;is-cluster=false";

const instanceLink = computed(() => {
  if (basicInformation.value.engine == "SNOWFLAKE") {
    if (adminDataSource.value.host) {
      return `https://${
        adminDataSource.value.host.split("@")[0]
      }.snowflakecomputing.com/console`;
    }
  }
  return basicInformation.value.externalLink || "";
});

const hasReadonlyReplicaHost = computed((): boolean => {
  return basicInformation.value.engine !== "SPANNER";
});

const hasReadonlyReplicaPort = computed((): boolean => {
  return basicInformation.value.engine !== "SPANNER";
});

const showDatabase = computed((): boolean => {
  return (
    (basicInformation.value.engine === "POSTGRES" ||
      basicInformation.value.engine === "REDSHIFT") &&
    state.currentDataSourceType === "ADMIN"
  );
});

const showSSL = computed((): boolean => {
  return instanceHasSSL(basicInformation.value.engine);
});

const showSSH = computed((): boolean => {
  return instanceHasSSH(basicInformation.value.engine);
});

const showAuthenticationDatabase = computed((): boolean => {
  return basicInformation.value.engine === "MONGODB";
});

const allowUpdate = computed((): boolean => {
  if (!valueChanged.value) {
    return false;
  }
  if (basicInformation.value.engine === "SPANNER") {
    if (!isValidSpannerHost(adminDataSource.value.host)) {
      return false;
    }
    if (
      readonlyDataSource.value &&
      !isValidSpannerHost(readonlyDataSource.value.host)
    ) {
      return false;
    }
  }
  return true;
});

const isEngineBeta = (engine: EngineType): boolean => {
  return ["ORACLE", "MSSQL", "REDSHIFT", "MARIADB", "OCEANBASE"].includes(
    engine
  );
};

// The default host name is 127.0.0.1 or host.docker.internal which is not applicable to Snowflake, so we change
// the host name between 127.0.0.1/host.docker.internal and "" if user hasn't changed default yet.
const changeInstanceEngine = (engine: EngineType) => {
  if (engine === "SNOWFLAKE" || engine === "SPANNER") {
    if (
      adminDataSource.value.host == "127.0.0.1" ||
      adminDataSource.value.host == "host.docker.internal"
    ) {
      adminDataSource.value.host = "";
    }
  } else {
    if (!adminDataSource.value.host) {
      adminDataSource.value.host = isDev()
        ? "127.0.0.1"
        : "host.docker.internal";
    }
  }
  basicInformation.value.engine = engine;
};

const handleInstanceNameInput = (event: Event) => {
  basicInformation.value.name = (event.target as HTMLInputElement).value;
};

const handleInstanceHostInput = (event: Event) => {
  adminDataSource.value.host = (event.target as HTMLInputElement).value;
};

const handleUpdateSpannerHost = (host: string) => {
  adminDataSource.value.host = host;
};

const handleInstancePortWheelScroll = (event: MouseEvent) => {
  (event.target as HTMLInputElement).blur();
};

const handleInstancePortInput = (event: Event) => {
  currentDataSource.value.port = (event.target as HTMLInputElement).value;
};

const handleInstanceExternalLinkInput = (event: Event) => {
  basicInformation.value.externalLink = (
    event.target as HTMLInputElement
  ).value.trim();
};

const handleDataSourceTypeChange = (value: string) => {
  state.currentDataSourceType = value as DataSourceType;
};

const handleCurrentDataSourceNameInput = (event: Event) => {
  const str = (event.target as HTMLInputElement).value.trim();
  currentDataSource.value.username = str;
};

const handleToggleUseEmptyPassword = (on: boolean) => {
  currentDataSource.value.useEmptyPassword = on;
};

const handleCurrentDataSourcePasswordInput = (event: Event) => {
  const password = (event.target as HTMLInputElement).value.trim();
  currentDataSource.value.updatedPassword = password;
};

const handleUpdateSpannerCredential = (credential: string) => {
  currentDataSource.value.updatedPassword = credential;
};

const handleInstanceAuthenticationDatabaseInput = (event: Event) => {
  const str = (event.target as HTMLInputElement).value.trim();
  currentDataSource.value.options.authenticationDatabase = str;
};

const handleMongodbConnectionStringSchemaChange = (event: Event) => {
  switch ((event.target as HTMLInputElement).value) {
    case mongodbConnectionStringSchemaList[0]:
      currentDataSource.value.options.srv = false;
      break;
    case mongodbConnectionStringSchemaList[1]:
      // MongoDB doesn't support specify port if using srv record.
      currentDataSource.value.port = "";
      currentDataSource.value.options.srv = true;
      break;
    default:
      currentDataSource.value.options.srv = false;
  }
};

const handleCurrentDataSourceHostInput = (event: Event) => {
  if (currentDataSource.value.type === "RO") {
    if (!hasFeature("bb.feature.read-replica-connection")) {
      if (currentDataSource.value.host || currentDataSource.value.port) {
        currentDataSource.value.host = "";
        currentDataSource.value.port = "";
        state.showFeatureModal = true;
        return;
      }
    }
  }

  const str = (event.target as HTMLInputElement).value.trim();
  currentDataSource.value.host = str;
};

const handleCurrentDataSourcePortInput = (event: Event) => {
  if (currentDataSource.value.type === "RO") {
    if (!hasFeature("bb.feature.read-replica-connection")) {
      if (currentDataSource.value.host || currentDataSource.value.port) {
        currentDataSource.value.host = "";
        currentDataSource.value.port = "";
        state.showFeatureModal = true;
        return;
      }
    }
  }

  const str = (event.target as HTMLInputElement).value.trim();
  currentDataSource.value.port = str;
};

const handleEditSsl = () => {
  const curr = currentDataSource.value;
  curr.sslCa = "";
  curr.sslCert = "";
  curr.sslKey = "";
  curr.updateSsl = true;
};

const handleCurrentDataSourceSslChange = (
  value: Pick<DataSource, "sslCa" | "sslCert" | "sslKey">
) => {
  Object.assign(currentDataSource.value, value);
  currentDataSource.value.updateSsl = true;
};

const handleCurrentDataSourceSshChange = (
  value: Partial<
    Pick<
      DataSourceOptions,
      "sshHost" | "sshPort" | "sshUser" | "sshPassword" | "sshPrivateKey"
    >
  >
) => {
  Object.assign(currentDataSource.value.options, value);
  currentDataSource.value.updateSsh = true;
};

const handleCreateRODataSource = () => {
  if (isCreating.value) {
    return;
  }

  const tempDataSource = {
    id: UNKNOWN_ID,
    instanceId: props.instance!.id,
    databaseId: adminDataSource.value.databaseId,
    name: `Read-only data source`,
    type: "RO",
    username: "",
    password: "",
    options: {
      authenticationDatabase: "",
      srv: false,
      sid: "",
      serviceName: "",
    },
  } as DataSource;
  if (basicInformation.value.engine === "SPANNER") {
    tempDataSource.host = adminDataSource.value.host;
  }
  readonlyDataSource.value = {
    ...tempDataSource,
    updatedPassword: "",
    useEmptyPassword: false,
  };
  state.currentDataSourceType = "RO";
};

const handleDeleteRODataSource = async () => {
  if (!readonlyDataSource.value) {
    return;
  }

  if (readonlyDataSource.value.id === UNKNOWN_ID) {
    readonlyDataSource.value = undefined;
  } else {
    await dataSourceStore.deleteDataSourceById({
      databaseId: readonlyDataSource.value.databaseId,
      dataSourceId: readonlyDataSource.value.id,
    });
    await updateInstanceState();
  }
  state.currentDataSourceType = "ADMIN";
};

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  try {
    const instance = await instanceV1Store.getOrFetchInstanceByName(
      instanceNamePrefix + resourceId
    );
    if (instance) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t("resource.instance"),
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

const updateInstanceState = async () => {
  if (!props.instance) {
    return;
  }

  const instance = await instanceStore.fetchInstanceById(props.instance.id);
  basicInformation.value = {
    id: instance.id,
    resourceId: instance.resourceId,
    rowStatus: instance.rowStatus,
    name: instance.name,
    engine: instance.engine,
    environmentId: instance.environment.id as number,
  };
  adminDataSource.value = {
    ...cloneDeep(instance.dataSourceList.find((ds) => ds.type === "ADMIN")!),
    updatedPassword: "",
    useEmptyPassword: false,
  } as EditDataSource;
  if (instance.dataSourceList.find((ds) => ds.type === "RO")) {
    readonlyDataSource.value = {
      ...(cloneDeep(
        instance.dataSourceList.find((ds) => ds.type === "RO")!
      ) as EditDataSource),
      updatedPassword: "",
      useEmptyPassword: false,
    };
  }
  useDatabaseStore().fetchDatabaseListByInstanceId(instance.id);
  instanceStore.fetchInstanceUserListById(instance.id);

  const reloadDatabaseAndUser = connectionInfoChanged.value;
  // Backend will sync the schema when connection info changed, so we need to fetch the synced schema here.
  if (reloadDatabaseAndUser) {
    await useDatabaseStore().fetchDatabaseListByInstanceId(instance.id);
    await instanceStore.fetchInstanceUserListById(instance.id);
  }

  return instance;
};

const handleWarningModalOkClick = async () => {
  state.showCreateInstanceWarningModal = false;
  doCreate();
};

const tryCreate = async () => {
  const connectionContext = getTestConnectionContext();
  state.isTestingConnection = true;
  try {
    const resultSet = await sqlStore.ping(connectionContext);
    state.isTestingConnection = false;
    if (isEmpty(resultSet.error)) {
      await doCreate();
    } else {
      state.createInstanceWarning = t("instance.unable-to-connect", [
        resultSet.error,
      ]);
      state.showCreateInstanceWarningModal = true;
    }
  } catch (error) {
    state.isTestingConnection = false;
  }
};

// We will also create the database * denoting all databases
// and its RW data source. The username, password is actually
// stored in that data source object instead of in the instance self.
// Conceptually, data source is the proper place to store connection info (thinking of DSN)
const doCreate = async () => {
  if (!isCreating.value) {
    return;
  }

  const instanceCreate: InstanceCreate = {
    resourceId: resourceIdField.value?.resourceId as string,
    name: basicInformation.value.name.trim(),
    engine: basicInformation.value.engine,
    externalLink: basicInformation.value.externalLink,
    environmentId: basicInformation.value.environmentId,
    host: adminDataSource.value.host,
    port: adminDataSource.value.port,
    database: adminDataSource.value.database,
    username: adminDataSource.value.username,
    password: adminDataSource.value.updatedPassword,
    sslCa: adminDataSource.value.sslCa,
    sslCert: adminDataSource.value.sslCert,
    sslKey: adminDataSource.value.sslKey,
    srv: adminDataSource.value.options.srv,
    authenticationDatabase:
      adminDataSource.value.options.authenticationDatabase,
    sid: "",
    serviceName: "",
    sshHost: "",
    sshPort: "",
    sshUser: "",
    sshPassword: "",
    sshPrivateKey: "",
  };

  if (
    instanceCreate.engine !== "POSTGRES" &&
    instanceCreate.engine !== "MONGODB" &&
    instanceCreate.engine !== "REDSHIFT"
  ) {
    // Clear the `database` field if not needed.
    instanceCreate.database = "";
  }

  if (instanceCreate.engine === "ORACLE") {
    instanceCreate.sid = adminDataSource.value.options.sid;
    instanceCreate.serviceName = adminDataSource.value.options.serviceName;
  }

  if (showSSH.value) {
    // Default to "NONE"
    instanceCreate.sshHost = adminDataSource.value.options.sshHost ?? "";
    instanceCreate.sshPort = adminDataSource.value.options.sshPort ?? "";
    instanceCreate.sshUser = adminDataSource.value.options.sshUser ?? "";
    instanceCreate.sshPassword =
      adminDataSource.value.options.sshPassword ?? "";
    instanceCreate.sshPrivateKey =
      adminDataSource.value.options.sshPrivateKey ?? "";
  }

  state.isRequesting = true;
  const createdInstance = await instanceStore.createInstance(instanceCreate);
  state.isRequesting = false;

  router.push(`/instance/${instanceSlug(createdInstance)}`);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("instance.successfully-created-instance-createdinstance-name", [
      createdInstance.name,
    ]),
  });
  emit("dismiss");
};

const doUpdate = async () => {
  if (!props.instance) {
    return;
  }
  const patchedInstance: InstancePatch = {};
  let instanceInfoChanged = false;
  let dataSourceListChanged = false;

  if (basicInformation.value.name.trim() != props.instance.name) {
    patchedInstance.name = basicInformation.value.name.trim();
    instanceInfoChanged = true;
  }
  if (basicInformation.value.externalLink != props.instance.externalLink) {
    patchedInstance.externalLink = basicInformation.value.externalLink;
    instanceInfoChanged = true;
  }

  const instance = await instanceStore.getOrFetchInstanceById(
    props.instance.id
  );
  const originAdminDataSource = instance.dataSourceList.find(
    (ds) => ds.type === "ADMIN"
  ) as DataSource;
  const originReadonlyDataSource = instance.dataSourceList.find(
    (ds) => ds.type === "RO"
  );
  if (
    !isEqual(
      originAdminDataSource,
      convertEditDataSource(adminDataSource.value)
    ) ||
    !isEqual(
      originReadonlyDataSource,
      readonlyDataSource.value
        ? convertEditDataSource(readonlyDataSource.value)
        : undefined
    )
  ) {
    dataSourceListChanged = true;
  }

  if (instanceInfoChanged || dataSourceListChanged) {
    state.isRequesting = true;
    const requests: Promise<any>[] = [];

    if (dataSourceListChanged) {
      if (
        !isEqual(
          originAdminDataSource,
          convertEditDataSource(adminDataSource.value)
        )
      ) {
        const dataSource = convertEditDataSource(adminDataSource.value);
        const dataSourcePatch: DataSourcePatch = {
          ...dataSource,
        };
        requests.push(
          dataSourceStore.patchDataSource({
            databaseId: dataSource.databaseId,
            dataSourceId: dataSource.id,
            dataSourcePatch: dataSourcePatch,
          })
        );
      }

      if (
        !isEqual(
          originReadonlyDataSource,
          readonlyDataSource.value
            ? convertEditDataSource(readonlyDataSource.value)
            : undefined
        )
      ) {
        if (readonlyDataSource.value) {
          const dataSource = convertEditDataSource(readonlyDataSource.value);
          if (dataSource.id === UNKNOWN_ID) {
            const dataSourceCreate: DataSourceCreate = {
              ...readonlyDataSource.value,
              databaseId: adminDataSource.value.databaseId,
              instanceId: props.instance.id,
              name: dataSource.name,
              type: "RO",
              username: dataSource.username,
              password: dataSource.password,
              host: dataSource.host,
              port: dataSource.port,
              database: dataSource.database,
            };
            if (typeof dataSource.sslCa !== "undefined") {
              dataSourceCreate.sslCa = dataSource.sslCa;
            }
            if (typeof dataSource.sslKey !== "undefined") {
              dataSourceCreate.sslKey = dataSource.sslKey;
            }
            if (typeof dataSource.sslCert !== "undefined") {
              dataSourceCreate.sslCert = dataSource.sslCert;
            }
            requests.push(dataSourceStore.createDataSource(dataSourceCreate));
          } else {
            const dataSourcePatch: DataSourcePatch = {
              ...dataSource,
            };
            requests.push(
              dataSourceStore.patchDataSource({
                databaseId: dataSource.databaseId,
                dataSourceId: dataSource.id,
                dataSourcePatch: dataSourcePatch,
              })
            );
          }
        }
      }
    }

    if (instanceInfoChanged) {
      requests.push(
        instanceStore.patchInstance({
          instanceId: basicInformation.value.id,
          instancePatch: patchedInstance,
        })
      );
    }

    await Promise.all(requests);
    await updateInstanceState();
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-updated-instance-instance-name", [
        basicInformation.value.name.trim(),
      ]),
    });
    state.isRequesting = false;
  }
};

const getTestConnectionContext = () => {
  const dataSource = currentDataSource.value;
  let connectionHost = adminDataSource.value.host;
  let connectionPort = adminDataSource.value.port;
  if (dataSource.type === "RO") {
    if (dataSource.host) {
      connectionHost = dataSource.host;
    }
    if (dataSource.port) {
      connectionPort = dataSource.port;
    }
  }

  const connectionInfo: ConnectionInfo = {
    ...basicInformation.value,
    host: connectionHost,
    port: connectionPort,
    username: dataSource.username,
    password: dataSource.useEmptyPassword ? "" : dataSource.updatedPassword,
    useEmptyPassword: dataSource.useEmptyPassword,
    database: dataSource.database,
    srv: dataSource.options.srv,
    authenticationDatabase: dataSource.options.authenticationDatabase,
    sid: "",
    serviceName: "",
    sshHost: "",
    sshPort: "",
    sshUser: "",
    sshPassword: "",
    sshPrivateKey: "",
  };

  if (!isCreating.value) {
    connectionInfo.instanceId = basicInformation.value.id;
  }

  if (basicInformation.value.engine === "ORACLE") {
    connectionInfo.sid = dataSource.options.sid;
    connectionInfo.serviceName = dataSource.options.serviceName;
  }

  if (showSSL.value) {
    // Default to "NONE"
    connectionInfo.sslCa = adminDataSource.value.sslCa ?? "";
    connectionInfo.sslKey = adminDataSource.value.sslKey ?? "";
    connectionInfo.sslCert = adminDataSource.value.sslCert ?? "";

    if (typeof dataSource.sslCa !== "undefined") {
      connectionInfo.sslCa = dataSource.sslCa;
    }
    if (typeof dataSource.sslKey !== "undefined") {
      connectionInfo.sslKey = dataSource.sslKey;
    }
    if (typeof dataSource.sslCert !== "undefined") {
      connectionInfo.sslCert = dataSource.sslCert;
    }
  }

  if (showSSH.value) {
    // Default to "NONE"
    connectionInfo.sshHost = adminDataSource.value.options.sshHost ?? "";
    connectionInfo.sshPort = adminDataSource.value.options.sshPort ?? "";
    connectionInfo.sshUser = adminDataSource.value.options.sshUser ?? "";
    connectionInfo.sshPassword =
      adminDataSource.value.options.sshPassword ?? "";
    connectionInfo.sshPrivateKey =
      adminDataSource.value.options.sshPrivateKey ?? "";

    if (typeof dataSource.options.sshHost !== "undefined") {
      connectionInfo.sshHost = dataSource.options.sshHost;
    }
    if (typeof dataSource.options.sshPort !== "undefined") {
      connectionInfo.sshPort = dataSource.options.sshPort;
    }
    if (typeof dataSource.options.sshUser !== "undefined") {
      connectionInfo.sshUser = dataSource.options.sshUser;
    }
    if (typeof dataSource.options.sshPassword !== "undefined") {
      connectionInfo.sshPassword = dataSource.options.sshPassword;
    }
    if (typeof dataSource.options.sshPrivateKey !== "undefined") {
      connectionInfo.sshPrivateKey = dataSource.options.sshPrivateKey;
    }
  }
  return connectionInfo;
};

const testConnection = async () => {
  const connectionContext = getTestConnectionContext();
  state.isTestingConnection = true;
  const resultSet = await sqlStore.ping(connectionContext);
  if (isEmpty(resultSet.error)) {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("instance.successfully-connected-instance"),
    });
  } else {
    let title = t("instance.failed-to-connect-instance");
    if (
      connectionContext.host == "localhost" ||
      connectionContext.host == "127.0.0.1"
    ) {
      title = t("instance.failed-to-connect-instance-localhost");
    }
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: title,
      description: resultSet.error,
      // Manual hide, because user may need time to inspect the error
      manualHide: true,
    });
  }
  state.isTestingConnection = false;
};

// getInstanceStateData returns the origin instance data including
// basic information, admin data source and read-only data source.
const getInstanceStateData = () => {
  const instanceData: {
    basicInformation: BasicInformation;
    adminDataSource: EditDataSource;
    readonlyDataSource?: EditDataSource;
  } = {
    basicInformation: {
      id: props.instance?.id || UNKNOWN_ID,
      resourceId: props.instance?.resourceId || "",
      rowStatus: props.instance?.rowStatus || "NORMAL",
      name: props.instance?.name || t("instance.new-instance"),
      engine: props.instance?.engine || "MYSQL",
      environmentId: (props.instance?.environment.id || UNKNOWN_ID) as number,
    },
    adminDataSource: {
      ...(getDataSourceWithType("ADMIN") || unknown("DATA_SOURCE")),
      updatedPassword: "",
      useEmptyPassword: false,
    },
    readonlyDataSource: getDataSourceWithType("RO")
      ? ({
          ...getDataSourceWithType("RO"),
          updatedPassword: "",
          useEmptyPassword: false,
        } as EditDataSource)
      : undefined,
  };
  return instanceData;
};

const convertEditDataSource = (editDataSource: EditDataSource): DataSource => {
  const password = editDataSource.useEmptyPassword
    ? ""
    : editDataSource.updatedPassword || editDataSource.password;
  return {
    ...omit(editDataSource, ["updatedPassword", "useEmptyPassword"]),
    password: password,
  };
};
</script>

<style scoped>
/*  Removed the ticker in the number field  */
input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* Firefox */
input[type="number"] {
  -moz-appearance: textfield;
}
</style>
