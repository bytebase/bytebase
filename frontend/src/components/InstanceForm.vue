<template>
  <div class="space-y-6 divide-y divide-block-border">
    <div class="divide-y divide-block-border px-1">
      <div
        v-if="isCreating"
        class="w-full mt-4 mb-6 grid grid-cols-1 gap-4 sm:grid-cols-7"
      >
        <template v-for="engine in engineList" :key="engine">
          <div
            class="flex relative justify-center px-2 py-4 border border-control-border hover:bg-control-bg-hover cursor-pointer"
            @click.capture="changeInstanceEngine(engine)"
          >
            <div class="flex flex-col items-center">
              <img class="h-8 w-auto" :src="EngineIconPath[engine]" />
              <p class="mt-2 text-center textlabel">
                {{ engineName(engine) }}
              </p>
              <template v-if="isEngineBeta(engine)">
                <BBBetaBadge class="absolute right-0.5 top-1" />
              </template>
              <div class="mt-4 radio text-sm">
                <input
                  type="radio"
                  class="btn"
                  :checked="basicInformation.engine == engine"
                />
              </div>
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
              :disabled="!allowEdit"
              :value="adminDataSource.port"
              @wheel="handleInstancePortWheelScroll"
              @input="handleInstancePortInput"
            />
          </div>
        </template>

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
          @click.prevent="handleCreateDataSource('RO')"
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
              :require-confirm="currentDataSource.id !== UNKNOWN_ID"
              :ok-text="$t('common.delete')"
              :confirm-title="
                $t('data-source.delete-read-only-data-source') + '?'
              "
              @confirm="handleDeleteReadOnlyDataSource"
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
              <span class="text-red-600">*</span>
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
                <span class="text-red-600">*</span>
              </label>
              <BBCheckbox
                v-if="allowUsingEmptyPassword"
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
          :write-only="true"
          class="mt-2 sm:col-span-3 sm:col-start-1"
          @update:value="handleUpdateSpannerCredential"
        />

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
            <label for="password" class="textlabel block">
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
              <button
                class="btn-normal mt-2"
                @click.prevent="handleEditSsl(true)"
              >
                {{ $t("common.edit") }} - {{ $t("common.write-only") }}
              </button>
            </template>
          </template>
        </div>
      </div>
      <div class="mt-6 pt-0 border-none">
        <div class="flex flex-row space-x-2">
          <button
            type="button"
            class="btn-normal whitespace-nowrap items-center"
            @click.prevent="testConnection"
          >
            {{ $t("instance.test-connection") }}
          </button>
        </div>
      </div>
    </div>

    <!-- Action Button Group -->
    <div class="pt-4 px-2">
      <div class="w-full flex justify-between items-center">
        <div class="w-full flex justify-end items-center">
          <div>
            <BBSpin v-if="state.isRequesting" :title="$t('common.updating')" />
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
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual, omit } from "lodash-es";
import { computed, reactive, PropType, ref, watch, onMounted } from "vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";
import {
  SpannerHostInput,
  SpannerCredentialInput,
  SslCertificateForm,
} from "./InstanceForm";
import {
  hasWorkspacePermission,
  instanceSlug,
  isDev,
  isValidSpannerHost,
} from "../utils";
import {
  InstancePatch,
  DataSourceType,
  Instance,
  SQLResultSet,
  ConnectionInfo,
  DataSource,
  UNKNOWN_ID,
  DataSourceCreate,
  DataSourcePatch,
  EngineType,
  engineName,
  empty,
  InstanceId,
  ResourceId,
  RowStatus,
  InstanceCreate,
  unknown,
} from "../types";
import isEmpty from "lodash-es/isEmpty";
import { useI18n } from "vue-i18n";
import {
  hasFeature,
  pushNotification,
  useCurrentUser,
  useDatabaseStore,
  useDataSourceStore,
  useInstanceStore,
  useSQLStore,
} from "@/store";
import { useRouter } from "vue-router";

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
  isRequesting: boolean;
  isPingingInstance: boolean;
  showCreateInstanceWarningModal: boolean;
  createInstanceWarning: string;
}

const { t } = useI18n();
const router = useRouter();
const instanceStore = useInstanceStore();
const dataSourceStore = useDataSourceStore();
const currentUser = useCurrentUser();
const sqlStore = useSQLStore();

const state = reactive<LocalState>({
  currentDataSourceType: "ADMIN",
  showFeatureModal: false,
  isRequesting: false,
  isPingingInstance: false,
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

const getDataSourceWithType = (type: DataSourceType) =>
  props.instance?.dataSourceList.find((ds) => ds.type === type);

// We only support one admin data source and one read-only data source.
const adminDataSource = ref<EditDataSource>({
  ...(getDataSourceWithType("ADMIN") || unknown("DATA_SOURCE")),
  updatedPassword: "",
  useEmptyPassword: false,
});

const readonlyDataSource = ref<EditDataSource | undefined>(
  getDataSourceWithType("RO")
    ? ({
        ...getDataSourceWithType("RO"),
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
  }
  return "3306";
};

const isCreating = computed(() => props.instance === undefined);

onMounted(() => {
  if (isCreating.value) {
    (adminDataSource.value.host = isDev()
      ? "127.0.0.1"
      : "host.docker.internal"),
      (adminDataSource.value.options.srv = false);
    adminDataSource.value.options.authenticationDatabase = "";
  }
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
  const engines: EngineType[] = [
    "MYSQL",
    "POSTGRES",
    "TIDB",
    "SNOWFLAKE",
    "CLICKHOUSE",
    "MONGODB",
    "SPANNER",
  ];
  return engines;
});

const EngineIconPath = {
  MYSQL: new URL("../assets/db-mysql.png", import.meta.url).href,
  POSTGRES: new URL("../assets/db-postgres.png", import.meta.url).href,
  TIDB: new URL("../assets/db-tidb.png", import.meta.url).href,
  SNOWFLAKE: new URL("../assets/db-snowflake.png", import.meta.url).href,
  CLICKHOUSE: new URL("../assets/db-clickhouse.png", import.meta.url).href,
  MONGODB: new URL("../assets/db-mongodb.png", import.meta.url).href,
  SPANNER: new URL("../assets/db-spanner.png", import.meta.url).href,
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
      basicInformation.value.name &&
      isValidSpannerHost(adminDataSource.value.host) &&
      adminDataSource.value.password
    );
  }
  return basicInformation.value.name && adminDataSource.value.host;
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

const allowUsingEmptyPassword = computed(() => {
  return basicInformation.value.engine !== "SPANNER";
});

const valueChanged = computed(() => {
  return true;
});

const connectionInfoChanged = computed(() => {
  if (!valueChanged.value) {
    return false;
  }

  return true;
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
    return "27017";
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
    if (currentDataSource.value.host) {
      return `https://${
        currentDataSource.value.host.split("@")[0]
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
    basicInformation.value.engine === "POSTGRES" &&
    state.currentDataSourceType === "ADMIN"
  );
});

const showSSL = computed((): boolean => {
  return (
    basicInformation.value.engine === "CLICKHOUSE" ||
    basicInformation.value.engine === "MYSQL" ||
    basicInformation.value.engine === "TIDB" ||
    basicInformation.value.engine === "POSTGRES"
  );
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
  return engine === "MONGODB" || engine === "SPANNER";
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
  basicInformation.value.name = (event.target as HTMLInputElement).value.trim();
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

const handleDeleteReadOnlyDataSource = async () => {
  if (!readonlyDataSource.value || readonlyDataSource.value.id === UNKNOWN_ID) {
    return;
  }

  await dataSourceStore.deleteDataSourceById({
    databaseId: readonlyDataSource.value.databaseId,
    dataSourceId: readonlyDataSource.value.id,
  });
  state.currentDataSourceType = "ADMIN";
  await updateInstanceState();
};

const handleEditSsl = (edit: boolean) => {
  const curr = currentDataSource.value;
  if (!edit) {
    delete curr.sslCa;
    delete curr.sslCert;
    delete curr.sslKey;
    delete curr.updateSsl;
  } else {
    curr.sslCa = "";
    curr.sslCert = "";
    curr.sslKey = "";
    curr.updateSsl = true;
  }
};

const handleCurrentDataSourceSslChange = (
  value: Pick<DataSource, "sslCa" | "sslCert" | "sslKey">
) => {
  Object.assign(currentDataSource.value, value);
  currentDataSource.value.updateSsl = true;
};

const handleCreateDataSource = (type: DataSourceType) => {
  if (isCreating.value) {
    return;
  }

  const tempDataSource = {
    id: UNKNOWN_ID,
    instanceId: props.instance!.id,
    databaseId: adminDataSource.value.databaseId,
    name: `${type} data source`,
    type: type,
    username: "",
    password: "",
    options: {
      authenticationDatabase: "",
      srv: false,
    },
  } as DataSource;
  if (basicInformation.value.engine === "SPANNER") {
    tempDataSource.host = adminDataSource.value.host;
  }
  state.currentDataSourceType = type;
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
  // Backend will sync the schema upon connection info change, so here we try to fetch the synced schema.
  if (reloadDatabaseAndUser) {
    await useDatabaseStore().fetchDatabaseListByInstanceId(instance.id);
    await instanceStore.fetchInstanceUserListById(instance.id);
  }

  return instance;
};

const tryCreate = () => {
  const connectionInfo: ConnectionInfo = {
    engine: basicInformation.value.engine,
    username: adminDataSource.value.username,
    password: adminDataSource.value.password,
    // When creating instance, the password is always needed.
    useEmptyPassword: false,
    host: adminDataSource.value.host,
    port: adminDataSource.value.port,
    srv: adminDataSource.value.options.srv,
    authenticationDatabase:
      adminDataSource.value.options.authenticationDatabase,
  };

  if (showSSL.value) {
    // Default to "NONE"
    connectionInfo.sslCa = adminDataSource.value.sslCa ?? "";
    connectionInfo.sslKey = adminDataSource.value.sslKey ?? "";
    connectionInfo.sslCert = adminDataSource.value.sslCert ?? "";
  }

  // MongoDB can use auth database.
  // https://www.mongodb.com/docs/manual/tutorial/authenticate-a-user/#std-label-authentication-auth-as-user
  if (basicInformation.value.engine === "MONGODB") {
    connectionInfo.database = adminDataSource.value.database;
  }

  state.isPingingInstance = true;
  sqlStore
    .ping(connectionInfo)
    .then((resultSet: SQLResultSet) => {
      state.isPingingInstance = false;
      if (isEmpty(resultSet.error)) {
        doCreate();
      } else {
        state.createInstanceWarning = t("instance.unable-to-connect", [
          resultSet.error,
        ]);
        state.showCreateInstanceWarningModal = true;
      }
    })
    .catch(() => {
      state.isPingingInstance = false;
    });
};

// We will also create the database * denoting all databases
// and its RW data source. The username, password is actually
// stored in that data source object instead of in the instance self.
// Conceptually, data source is the proper place to store connection info (thinking of DSN)
const doCreate = async () => {
  state.isRequesting = true;

  const instanceCreate: InstanceCreate = {
    name: basicInformation.value.name,
    engine: basicInformation.value.engine,
    externalLink: basicInformation.value.externalLink,
    environmentId: basicInformation.value.environmentId,
    host: adminDataSource.value.host,
    port: adminDataSource.value.port,
    database: adminDataSource.value.database,
    username: adminDataSource.value.username,
    password: adminDataSource.value.password,
    sslCa: adminDataSource.value.sslCa,
    sslCert: adminDataSource.value.sslCert,
    sslKey: adminDataSource.value.sslKey,
    srv: adminDataSource.value.options.srv,
    authenticationDatabase:
      adminDataSource.value.options.authenticationDatabase,
  };

  if (
    instanceCreate.engine !== "POSTGRES" &&
    instanceCreate.engine !== "MONGODB"
  ) {
    // Clear the `database` field if not needed.
    instanceCreate.database = "";
  }
  const createdInstance = await instanceStore.createInstance(instanceCreate);
  state.isRequesting = false;
  emit("dismiss");
  router.push(`/instance/${instanceSlug(createdInstance)}`);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("instance.successfully-created-instance-createdinstance-name", [
      createdInstance.name,
    ]),
  });
};

const doUpdate = async () => {
  if (!props.instance) {
    return;
  }
  const patchedInstance: InstancePatch = {};
  let instanceInfoChanged = false;
  let dataSourceListChanged = false;

  if (basicInformation.value.name != props.instance.name) {
    patchedInstance.name = basicInformation.value.name;
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
        basicInformation.value.name,
      ]),
    });
    state.isRequesting = false;
  }
};

const testConnection = () => {
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
    srv: false,
    authenticationDatabase: "",
  };
  if (!isCreating.value) {
    connectionInfo.password = adminDataSource.value.password;
    connectionInfo.srv = dataSource.options.srv;
    connectionInfo.authenticationDatabase =
      dataSource.options.authenticationDatabase;
  }

  if (typeof dataSource.sslCa !== "undefined") {
    connectionInfo.sslCa = dataSource.sslCa;
  }
  if (typeof dataSource.sslKey !== "undefined") {
    connectionInfo.sslKey = dataSource.sslKey;
  }
  if (typeof dataSource.sslCert !== "undefined") {
    connectionInfo.sslCert = dataSource.sslCert;
  }

  sqlStore.ping(connectionInfo).then((resultSet: SQLResultSet) => {
    if (isEmpty(resultSet.error)) {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("instance.successfully-connected-instance"),
      });
    } else {
      let title = t("instance.failed-to-connect-instance");
      if (
        connectionInfo.host == "localhost" ||
        connectionInfo.host == "127.0.0.1"
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
  });
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
