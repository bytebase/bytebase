<template>
  <form class="space-y-6 divide-y divide-block-border">
    <div class="divide-y divide-block-border px-1">
      <!-- Instance Name -->
      <div class="mt-6 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="name" class="textlabel flex flex-row items-center">
            {{ $t("instance.instance-name") }}
            &nbsp;
            <span style="color: red">*</span>
            <InstanceEngineIcon class="ml-1" :instance="state.instance" />
            <span class="ml-1">{{ state.instance.engineVersion }}</span>
          </label>
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
              <span style="color: red">*</span>
            </template>
            <template v-else>
              {{ $t("instance.host-or-socket") }}
              <span style="color: red">*</span>
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
        v-if="!hasReadonlyDataSource"
        class="mt-4 flex flex-row justify-start items-center bg-yellow-50 border-none rounded-lg p-2 px-3 mt-0"
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
          <NTab name="RO" :disabled="!hasReadonlyDataSource">Read only</NTab>
        </NTabs>
        <CreateDataSourceExample
          className="sm:col-span-3 border-none mt-2"
          :createInstanceFlag="false"
          :engineType="state.instance.engine"
          :dataSourceType="state.currentDataSourceType"
        />
        <div class="mt-2 sm:col-span-1 sm:col-start-1">
          <label for="username" class="textlabel block">{{
            $t("common.username")
          }}</label>
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
    <div class="pt-4">
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
  </form>
</template>

<script lang="ts" setup>
import { computed, reactive, PropType, ComputedRef } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";
import { isDBAOrOwner } from "../utils";
import {
  Principal,
  InstancePatch,
  DataSourceType,
  Instance,
  SqlResultSet,
  ConnectionInfo,
  DataSource,
  UNKNOWN_ID,
} from "../types";
import isEmpty from "lodash-es/isEmpty";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

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
}

const props = defineProps({
  instance: {
    required: true,
    type: Object as PropType<Instance>,
  },
});

const store = useStore();
const { t } = useI18n();

const currentUser: ComputedRef<Principal> = computed(() =>
  store.getters["auth/currentUser"]()
);

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
});

const allowEdit = computed(() => {
  return (
    state.instance.rowStatus == "NORMAL" && isDBAOrOwner(currentUser.value.role)
  );
});

const valueChanged = computed(() => {
  return !isEqual(state.instance, state.originalInstance);
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

const hasReadonlyDataSource = computed(() => {
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

const handleInstanceNameInput = (event: Event) => {
  updateInstance("name", (event.target as HTMLInputElement).value);
};

const handleInstanceHostInput = (event: Event) => {
  updateInstance("host", (event.target as HTMLInputElement).value);
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

const updateInstanceDataSource = () => {
  const index = state.dataSourceList.findIndex(
    (ds) => ds === currentDataSource.value
  );
  state.instance.dataSourceList[index] = {
    ...state.instance.dataSourceList[index],
    username: currentDataSource.value.username,
    password: currentDataSource.value.useEmptyPassword
      ? ""
      : currentDataSource.value.updatedPassword,
  };
};

const handleCreateDataSource = (type: DataSourceType) => {
  // Don't allow creating RO in UI for SNOWFLAKE/POSTGRES till we figure out the grant.
  if (
    state.instance.engine === "SNOWFLAKE" ||
    state.instance.engine === "POSTGRES"
  ) {
    pushNotification({
      module: "bytebase",
      style: "WARN",
      title: t("instance.no-read-only-data-source-support", {
        database: state.instance.engine,
      }),
    });
    return;
  }

  const tempDataSource = {
    id: UNKNOWN_ID,
    instanceId: state.instance.id,
    databaseId: adminDataSource.value.databaseId,
    name: `${type} data source`,
    type: type,
    username: "",
    password: "",
  } as DataSource;
  state.instance.dataSourceList.push(tempDataSource);
  state.dataSourceList.push({
    ...tempDataSource,
    updatedPassword: "",
    useEmptyPassword: false,
  });
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

const doUpdate = () => {
  const patchedInstance: InstancePatch = {};
  let instanceInfoChanged = false;
  let dataSourceListChanged = false;
  let connectionInfoChanged = false;

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
    connectionInfoChanged = true;
    instanceInfoChanged = true;
  }
  if (state.instance.port != state.originalInstance.port) {
    patchedInstance.port = state.instance.port;
    instanceInfoChanged = true;
    connectionInfoChanged = true;
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
    const requests = [];

    if (dataSourceListChanged) {
      for (let i = 0; i < state.instance.dataSourceList.length; i++) {
        const dataSource = state.instance.dataSourceList[i];
        if (dataSource.id === UNKNOWN_ID) {
          // Only used to create ReadOnly data source right now.
          if (dataSource.type === "RO") {
            requests.push(
              store.dispatch("dataSource/createDataSource", {
                databaseId: dataSource.databaseId,
                instanceId: state.instance.id,
                name: dataSource.name,
                type: dataSource.type,
                username: dataSource.username,
                password: dataSource.password,
              })
            );
          }
        } else if (
          !isEqual(dataSource, state.originalInstance.dataSourceList[i])
        ) {
          requests.push(
            store.dispatch("dataSource/patchDataSource", {
              databaseId: dataSource.databaseId,
              dataSourceId: dataSource.id,
              dataSource: dataSource,
            })
          );
        }
      }
    }

    if (instanceInfoChanged) {
      requests.push(
        store.dispatch("instance/patchInstance", {
          instanceId: state.instance.id,
          instancePatch: patchedInstance,
        })
      );
    }

    Promise.all(requests).then(() => {
      store
        .dispatch("instance/fetchInstanceById", state.instance.id)
        .then((instance) => {
          state.isUpdating = false;
          state.originalInstance = instance;
          state.instance = cloneDeep(state.originalInstance);
          state.dataSourceList = state.instance.dataSourceList.map(
            (dataSource) => {
              return {
                ...cloneDeep(dataSource),
                updatedPassword: "",
                useEmptyPassword: false,
              } as EditDataSource;
            }
          );
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("instance.successfully-updated-instance-instance-name", [
              instance.name,
            ]),
          });

          // Backend will sync the schema upon connection info change, so here we try to fetch the synced schema.
          if (connectionInfoChanged) {
            store.dispatch(
              "database/fetchDatabaseListByInstanceId",
              state.instance.id
            );
          }
        })
        .finally(() => {
          state.isUpdating = false;
        });
    });
  }
};

const testConnection = () => {
  const connectionInfo: ConnectionInfo = {
    engine: state.instance.engine,
    username: currentDataSource.value.username,
    password: currentDataSource.value.useEmptyPassword
      ? ""
      : currentDataSource.value.updatedPassword,
    useEmptyPassword: currentDataSource.value.useEmptyPassword,
    host: state.instance.host,
    port: state.instance.port,
    instanceId: state.instance.id,
  };
  store.dispatch("sql/ping", connectionInfo).then((resultSet: SqlResultSet) => {
    if (isEmpty(resultSet.error)) {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("instance.successfully-connected-instance"),
      });
    } else {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("instance.failed-to-connect-instance"),
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
