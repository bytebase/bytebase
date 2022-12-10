<template>
  <div class="space-y-6 divide-y divide-block-border">
    <div class="divide-y divide-block-border px-1">
      <!-- Instance Name -->
      <div class="mt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="name" class="textlabel flex flex-row items-center">
            {{ $t("instance.instance-name") }}
            <span class="text-red-600 mr-2">*</span>
            <InstanceEngineIcon :instance="state.instance" />
            <span class="ml-1">{{ state.instance.engineVersion }}</span>
          </label>
          <p class="text-sm text-gray-500 mt-1 mb-2">
            {{ $t("instance.instance-name-unique") }}
          </p>
          <input
            id="name"
            required
            name="name"
            type="text"
            class="textfield mt-1 w-full"
            :disabled="!allowEdit"
            :value="state.instance.name"
            @input="handleInstanceNameInput"
          />
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="environment" class="textlabel">
            {{ $t("common.environment") }}
          </label>
          <!-- Disallow changing environment after creation. This is to take the conservative approach to limit capability -->
          <!-- eslint-disable vue/attribute-hyphenation -->
          <EnvironmentSelect
            id="environment"
            class="mt-1 w-full"
            name="environment"
            :disabled="true"
            :selectedId="state.instance.environment.id"
            @select-environment-id="
              (environmentId) => {
                updateInstance('environmentId', environmentId);
              }
            "
          />
        </div>

        <div class="sm:col-span-3 sm:col-start-1">
          <label for="host" class="textlabel block">
            <template v-if="state.instance.engine == 'SNOWFLAKE'">
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
              state.instance.engine == 'SNOWFLAKE'
                ? $t('instance.your-snowflake-account-name')
                : $t('instance.sentence.host.snowflake')
            "
            class="textfield mt-1 w-full"
            :disabled="!allowEdit"
            :value="state.instance.host"
            @input="handleInstanceHostInput"
          />
          <div
            v-if="state.instance.engine == 'SNOWFLAKE'"
            class="mt-2 textinfolabel"
          >
            {{ $t("instance.sentence.proxy.snowflake") }}
          </div>
        </div>

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
            :value="state.instance.port"
            @wheel="handleInstancePortWheelScroll"
            @input="handleInstancePortInput"
          />
        </div>

        <!--Do not show external link on create to reduce cognitive load-->
        <div class="sm:col-span-3 sm:col-start-1">
          <label for="externallink" class="textlabel inline-flex">
            <span class>
              {{
                state.instance.engine == "SNOWFLAKE"
                  ? $t("instance.snowflake-web-console")
                  : $t("instance.external-link")
              }}
            </span>
            <button
              class="ml-1 btn-icon"
              :disabled="instanceLink(state.instance)?.trim().length == 0"
              @click.prevent="
                window.open(urlfy(instanceLink(state.instance)), '_blank')
              "
            >
              <heroicons-outline:external-link class="w-4 h-4" />
            </button>
          </label>
          <template v-if="state.instance.engine == 'SNOWFLAKE'">
            <input
              id="externallink"
              required
              name="externallink"
              type="text"
              class="textfield mt-1 w-full"
              disabled="true"
              :value="instanceLink(state.instance)"
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
              :value="state.instance.externalLink"
              class="textfield mt-1 w-full"
              :placeholder="snowflakeExtraLinkPlaceHolder"
              @input="handleInstanceExternalLinkInput"
            />
          </template>
        </div>
      </div>

      <p class="mt-6 pt-4 w-full text-lg leading-6 font-medium text-gray-900">
        {{ $t("instance.connection-info") }}
      </p>

      <div
        v-if="!hasReadOnlyDataSource"
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
        <CreateDataSourceExample
          className="sm:col-span-3 border-none mt-2"
          :createInstanceFlag="false"
          :engineType="state.instance.engine"
          :dataSourceType="state.currentDataSourceType"
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
              instance.engine == 'CLICKHOUSE' ? $t('common.default') : ''
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

        <template v-if="state.currentDataSourceType === 'RO'">
          <div class="mt-2 sm:col-span-1 sm:col-start-1">
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
              :value="currentDataSource.hostOverride"
              @input="handleCurrentDataSourceHostOverrideInput"
            />
          </div>

          <div class="mt-2 sm:col-span-1 sm:col-start-1">
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
              :value="currentDataSource.portOverride"
              @input="handleCurrentDataSourcePortOverrideInput"
            />
          </div>
        </template>

        <div v-if="showDatabase" class="mt-2 sm:col-span-1 sm:col-start-1">
          <label for="database" class="textlabel block">
            {{ $t("common.database") }}
          </label>
          <input
            id="database"
            v-model="state.instance.database"
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
              <button
                class="btn-normal mt-2"
                @click.prevent="handleEditSsl(false)"
              >
                {{ $t("common.revert") }}
              </button>
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
            :disabled="!instance.host"
            @click.prevent="testConnection"
          >
            {{ $t("instance.test-connection") }}
          </button>
        </div>
      </div>
    </div>

    <!-- Action Button Group -->
    <div class="pt-4 px-2">
      <div class="flex justify-between items-center">
        <div class="flex justify-end items-center">
          <div>
            <BBSpin v-if="state.isUpdating" :title="$t('common.updating')" />
          </div>
          <button
            v-if="allowEdit"
            type="button"
            :disabled="!valueChanged || state.isUpdating"
            :class="
              !valueChanged || state.isUpdating ? 'btn-normal' : 'btn-primary'
            "
            @click.prevent="doUpdate"
          >
            {{ $t("common.update") }}
          </button>
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
import { cloneDeep, isEqual, pick } from "lodash-es";
import { computed, reactive, PropType } from "vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";
import { SslCertificateForm } from "./InstanceForm";
import { hasWorkspacePermission } from "../utils";
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
import { isNullOrUndefined } from "@/plugins/demo/utils";

interface EditDataSource extends DataSource {
  updatedPassword: string;
  useEmptyPassword: boolean;
}

interface State {
  originalInstance: Instance;
  instance: Instance;
  isUpdating: boolean;
  dataSourceList: EditDataSource[];
  currentDataSourceType: DataSourceType;
  showFeatureModal: boolean;
  useDNSSRVRecord: boolean;
}

const props = defineProps({
  instance: {
    required: true,
    type: Object as PropType<Instance>,
  },
});

const instanceStore = useInstanceStore();
const { t } = useI18n();
const dataSourceStore = useDataSourceStore();

const currentUser = useCurrentUser();
const sqlStore = useSQLStore();

const dataSourceList = props.instance.dataSourceList.map((dataSource) => {
  return {
    ...cloneDeep(dataSource),
    updatedPassword: "",
    useEmptyPassword: false,
  } as EditDataSource;
});

const state = reactive<State>({
  originalInstance: props.instance,
  // Make hard copy since we are going to make equal comparison to determine the update button enable state.
  instance: cloneDeep(props.instance),
  isUpdating: false,
  dataSourceList: dataSourceList,
  currentDataSourceType: "ADMIN",
  showFeatureModal: false,
  useDNSSRVRecord: false,
});

const allowEdit = computed(() => {
  return (
    state.instance.rowStatus == "NORMAL" &&
    hasWorkspacePermission(
      "bb.permission.workspace.manage-instance",
      currentUser.value.role
    )
  );
});

const valueChanged = computed(() => {
  return !isEqual(state.instance, state.originalInstance);
});

const connectionInfoChanged = computed(() => {
  if (!valueChanged.value) {
    return false;
  }

  return (
    state.instance.host !== state.originalInstance.host ||
    state.instance.port !== state.originalInstance.port ||
    state.instance.database !== state.originalInstance.database ||
    !isEqual(
      state.originalInstance.dataSourceList,
      state.instance.dataSourceList
    )
  );
});

const defaultPort = computed(() => {
  if (state.instance.engine == "CLICKHOUSE") {
    return "9000";
  } else if (state.instance.engine == "POSTGRES") {
    return "5432";
  } else if (state.instance.engine == "SNOWFLAKE") {
    return "443";
  } else if (state.instance.engine == "TIDB") {
    return "4000";
  }
  return "3306";
});

const adminDataSource = computed(() => {
  const temp = state.dataSourceList.find(
    (ds) => ds.type === "ADMIN"
  ) as DataSource;
  return temp;
});

const hasReadOnlyDataSource = computed(() => {
  for (const ds of state.dataSourceList) {
    if (ds.type === "RO") {
      return true;
    }
  }
  return false;
});

const currentDataSource = computed(() => {
  const temp = state.dataSourceList.find(
    (ds) => ds.type === state.currentDataSourceType
  ) as EditDataSource;
  return temp;
});

const snowflakeExtraLinkPlaceHolder =
  "https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=mysql-instance-foo;is-cluster=false";

const instanceLink = (instance: Instance): string => {
  if (instance.engine == "SNOWFLAKE") {
    if (instance.host) {
      return `https://${
        instance.host.split("@")[0]
      }.snowflakecomputing.com/console`;
    }
  }
  return instance.host;
};

const showDatabase = computed((): boolean => {
  return (
    state.instance.engine === "POSTGRES" &&
    currentDataSource.value.type === "ADMIN"
  );
});

const showSSL = computed((): boolean => {
  return state.instance.engine === "CLICKHOUSE";
});

const handleInstanceNameInput = (event: Event) => {
  updateInstance("name", (event.target as HTMLInputElement).value);
};

const handleInstanceHostInput = (event: Event) => {
  updateInstance("host", (event.target as HTMLInputElement).value);
};

const handleInstancePortWheelScroll = (event: MouseEvent) => {
  (event.target as HTMLInputElement).blur();
};

const handleInstancePortInput = (event: Event) => {
  updateInstance("port", (event.target as HTMLInputElement).value);
};

const handleInstanceExternalLinkInput = (event: Event) => {
  updateInstance("externalLink", (event.target as HTMLInputElement).value);
};

const handleDataSourceTypeChange = (value: string) => {
  state.currentDataSourceType = value as DataSourceType;
};

const handleCurrentDataSourceNameInput = (event: Event) => {
  const str = (event.target as HTMLInputElement).value.trim();
  currentDataSource.value.username = str;
  updateInstanceDataSource();
};

const handleToggleUseEmptyPassword = (on: boolean) => {
  currentDataSource.value.useEmptyPassword = on;
  updateInstanceDataSource();
};

const handleCurrentDataSourcePasswordInput = (event: Event) => {
  const str = (event.target as HTMLInputElement).value.trim();
  currentDataSource.value.updatedPassword = str;
  updateInstanceDataSource();
};

const handleCurrentDataSourceHostOverrideInput = (event: Event) => {
  const str = (event.target as HTMLInputElement).value.trim();
  currentDataSource.value.hostOverride = str;
  updateInstanceDataSource();
};

const handleCurrentDataSourcePortOverrideInput = (event: Event) => {
  const str = (event.target as HTMLInputElement).value.trim();
  currentDataSource.value.portOverride = str;
  updateInstanceDataSource();
};

const handleDeleteReadOnlyDataSource = async () => {
  const dataSource = state.dataSourceList.find(
    (item) => item.type === "RO"
  ) as DataSource;
  if (isNullOrUndefined(dataSource)) {
    return;
  }
  if (dataSource.type !== "RO") {
    return;
  }

  if (dataSource.id !== UNKNOWN_ID) {
    await dataSourceStore.deleteDataSourceById({
      databaseId: dataSource.databaseId,
      dataSourceId: dataSource.id,
    });
  }

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
  updateInstanceDataSource();
};

const handleCurrentDataSourceSslChange = (
  value: Pick<DataSource, "sslCa" | "sslCert" | "sslKey">
) => {
  Object.assign(currentDataSource.value, value);
  currentDataSource.value.updateSsl = true;
  updateInstanceDataSource();
};

const updateInstanceDataSource = () => {
  const curr = currentDataSource.value;
  const index = state.dataSourceList.findIndex((ds) => ds === curr);
  let newValue = {
    ...state.instance.dataSourceList[index],
    username: curr.username,
  };

  if (curr.type === "RO") {
    if (!hasFeature("bb.feature.read-replica-connection")) {
      if (curr.hostOverride || curr.portOverride) {
        state.dataSourceList[index].hostOverride = "";
        state.dataSourceList[index].portOverride = "";
        newValue.hostOverride = "";
        newValue.portOverride = "";
        state.showFeatureModal = true;
      }
    } else {
      newValue = {
        ...newValue,
        ...pick(curr, ["hostOverride", "portOverride"]),
      };
    }
  }

  if (curr.useEmptyPassword) {
    // When 'Password: Empty' is checked, we set the password to empty string.
    newValue.password = "";
  } else if (curr.updatedPassword) {
    // When the user has typed something in the password textbox, we use the typed value.
    newValue.password = curr.updatedPassword;
  } else {
    // When the user didn't touch the password textbox, or the user did typed something
    // but cleared the textbox again, we won't update the password.
    delete newValue.password;
  }

  if (curr.updateSsl) {
    newValue.sslCa = curr.sslCa;
    newValue.sslKey = curr.sslKey;
    newValue.sslCert = curr.sslCert;
  } else {
    delete newValue.sslCa;
    delete newValue.sslCert;
    delete newValue.sslKey;
  }

  state.instance.dataSourceList[index] = newValue;
};

const handleCreateDataSource = (type: DataSourceType) => {
  const tempDataSource = {
    id: UNKNOWN_ID,
    instanceId: state.instance.id,
    databaseId: adminDataSource.value.databaseId,
    name: `${type} data source`,
    type: type,
    username: "",
    password: "",
  } as DataSource;
  state.dataSourceList.push({
    ...tempDataSource,
    updatedPassword: "",
    useEmptyPassword: false,
  });
  state.instance.dataSourceList = state.dataSourceList;
  state.currentDataSourceType = type;
};

const updateInstance = (field: string, value: string) => {
  let str = value;
  if (
    field == "name" ||
    field == "host" ||
    field == "port" ||
    field == "externalLink"
  ) {
    str = value.trim();
  }
  (state.instance as any)[field] = str;
};

const updateInstanceState = async () => {
  const instance = await instanceStore.fetchInstanceById(state.instance.id);
  state.originalInstance = instance;
  state.instance = cloneDeep(state.originalInstance);
  state.dataSourceList = instance.dataSourceList.map((dataSource) => {
    return {
      ...cloneDeep(dataSource),
      updatedPassword: "",
      useEmptyPassword: false,
    } as EditDataSource;
  });
  useDatabaseStore().fetchDatabaseListByInstanceId(instance.id);
  useInstanceStore().fetchInstanceUserListById(instance.id);

  const reloadDatabaseAndUser = connectionInfoChanged.value;
  // Backend will sync the schema upon connection info change, so here we try to fetch the synced schema.
  if (reloadDatabaseAndUser) {
    await useDatabaseStore().fetchDatabaseListByInstanceId(instance.id);
    await useInstanceStore().fetchInstanceUserListById(instance.id);
  }

  return instance;
};

const doUpdate = () => {
  const patchedInstance: InstancePatch = {};
  let instanceInfoChanged = false;
  let dataSourceListChanged = false;

  if (state.instance.name != state.originalInstance.name) {
    patchedInstance.name = state.instance.name;
    instanceInfoChanged = true;
  }
  if (state.instance.externalLink != state.originalInstance.externalLink) {
    patchedInstance.externalLink = state.instance.externalLink;
    instanceInfoChanged = true;
  }
  if (state.instance.host != state.originalInstance.host) {
    patchedInstance.host = state.instance.host;
    instanceInfoChanged = true;
  }
  if (state.instance.port != state.originalInstance.port) {
    patchedInstance.port = state.instance.port;
    instanceInfoChanged = true;
  }
  if (state.instance.database !== state.originalInstance.database) {
    patchedInstance.database = state.instance.database;
    instanceInfoChanged = true;
  }

  if (
    !isEqual(
      state.originalInstance.dataSourceList,
      state.instance.dataSourceList
    )
  ) {
    dataSourceListChanged = true;
  }

  if (instanceInfoChanged || dataSourceListChanged) {
    state.isUpdating = true;
    const requests: Promise<any>[] = [];

    if (dataSourceListChanged) {
      for (let i = 0; i < state.instance.dataSourceList.length; i++) {
        const dataSource = state.instance.dataSourceList[i];
        if (dataSource.id === UNKNOWN_ID) {
          // Only used to create ReadOnly data source right now.
          if (dataSource.type === "RO") {
            const dataSourceCreate: DataSourceCreate = {
              databaseId: dataSource.databaseId,
              instanceId: state.instance.id,
              name: dataSource.name,
              type: dataSource.type,
              username: dataSource.username,
              password: dataSource.password,
              hostOverride: dataSource.hostOverride,
              portOverride: dataSource.portOverride,
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
          }
        } else if (
          !isEqual(dataSource, state.originalInstance.dataSourceList[i])
        ) {
          const dataSourcePatch: DataSourcePatch = {
            ...dataSource,
          };
          if (dataSource.type !== "RO") {
            dataSourcePatch.hostOverride = undefined;
            dataSourcePatch.portOverride = undefined;
          }

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

    if (instanceInfoChanged) {
      requests.push(
        instanceStore.patchInstance({
          instanceId: state.instance.id,
          instancePatch: patchedInstance,
        })
      );
    }

    Promise.all(requests).then(() => {
      updateInstanceState()
        .then((instance) => {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("instance.successfully-updated-instance-instance-name", [
              instance.name,
            ]),
          });
        })
        .finally(() => {
          state.isUpdating = false;
        });
    });
  }
};

const testConnection = () => {
  const instance = state.instance;
  const dataSource = currentDataSource.value;
  let connectionHost = instance.host;
  let connectionPort = instance.port;
  if (dataSource.type === "RO") {
    if (dataSource.hostOverride && dataSource.portOverride) {
      connectionHost = dataSource.hostOverride;
      connectionPort = dataSource.portOverride;
    }
  }

  const connectionInfo: ConnectionInfo = {
    engine: instance.engine,
    username: dataSource.username,
    password: dataSource.useEmptyPassword ? "" : dataSource.updatedPassword,
    useEmptyPassword: dataSource.useEmptyPassword,
    host: connectionHost,
    port: connectionPort,
    database: instance.database,
    instanceId: instance.id,
    srv: state.useDNSSRVRecord,
  };

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
