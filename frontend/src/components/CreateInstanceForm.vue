<template>
  <form class="space-y-6 divide-y divide-block-border">
    <div class="divide-y divide-block-border px-1">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-6">
        <template v-for="engine in engineList" :key="engine">
          <div
            class="flex justify-center px-2 py-4 border border-control-border hover:bg-control-bg-hover cursor-pointer"
            @click.capture="changeInstanceEngine(engine)"
          >
            <div class="flex flex-col items-center">
              <img class="h-8 w-auto" :src="EngineIconPath[engine]" />
              <p class="mt-1 text-center textlabel">
                {{ engineName(engine) }}
              </p>
              <div class="mt-3 radio text-sm">
                <input
                  type="radio"
                  class="btn"
                  :checked="state.instance.engine == engine"
                />
              </div>
            </div>
          </div>
        </template>
      </div>
      <!-- Instance Name -->
      <div class="mt-6 grid grid-cols-1 gap-y-6 gap-x-4 pt-4 sm:grid-cols-4">
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="name" class="textlabel flex flex-row items-center">
            {{ $t("instance.instance-name") }}
            &nbsp;
            <span style="color: red">*</span>
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
            <span style="color: red">*</span>
          </label>
          <!-- Disallow changing environment after creation. This is to take the conservative approach to limit capability -->
          <!-- eslint-disable vue/attribute-hyphenation -->
          <EnvironmentSelect
            id="environment"
            class="mt-1 w-full"
            name="environment"
            :selectedId="state.instance.environmentId"
            @select-environment-id="
              (environmentId) => {
                state.instance.environmentId = environmentId;
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
      </div>

      <p class="mt-6 pt-4 w-full text-lg leading-6 font-medium text-gray-900">
        {{ $t("instance.connection-info") }}
      </p>

      <div
        class="mt-2 grid grid-cols-1 gap-y-2 gap-x-4 border-none sm:grid-cols-3"
      >
        <CreateDataSourceExample
          className="sm:col-span-3"
          :createInstanceFlag="true"
          :engineType="state.instance.engine"
          :dataSourceType="'ADMIN'"
        />
        <div class="sm:col-span-1 sm:col-start-1">
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
            :placeholder="
              state.instance.engine == 'CLICKHOUSE' ? $t('common.default') : ''
            "
            :value="state.instance.username"
            @input="handleInstanceUsernameInput"
          />
        </div>

        <div class="sm:col-span-1 sm:col-start-1">
          <div class="flex flex-row items-center space-x-2">
            <label for="password" class="textlabel block">
              {{ $t("common.password") }}
            </label>
          </div>
          <input
            id="password"
            name="password"
            type="text"
            class="textfield mt-1 w-full"
            autocomplete="off"
            :placeholder="$t('instance.password-write-only')"
            :value="state.instance.password"
            @input="handleInstancePasswordInput"
          />
        </div>

        <div v-if="showSSL" class="sm:col-span-3 sm:col-start-1">
          <div class="flex flex-row items-center space-x-2">
            <label class="textlabel block">{{
              $t("datasource.ssl-connection")
            }}</label>
          </div>
          <SslCertificateForm
            :value="state.instance"
            @change="Object.assign(state.instance, $event)"
          />
        </div>
      </div>

      <div class="mt-6 border-none">
        <div class="flex flex-row space-x-2">
          <button
            type="button"
            class="btn-normal whitespace-nowrap items-center"
            :disabled="!state.instance.host"
            @click.prevent="testConnection"
          >
            {{ $t("instance.test-connection") }}
          </button>
        </div>
      </div>
    </div>

    <!-- Action Button Group -->
    <div class="pt-4 px-2">
      <!-- Create button group -->
      <div class="flex justify-between items-center">
        <BBCheckbox
          :title="$t('instance.sync-schema-now')"
          :value="state.instance.syncSchema"
          @toggle="state.instance.syncSchema = !state.instance.syncSchema"
        />
        <div class="flex justify-end items-center">
          <BBSpin
            v-if="state.isCreatingInstance"
            :title="$t('common.creating')"
          />
          <div class="ml-2">
            <button
              type="button"
              class="btn-normal py-2 px-4"
              :disabled="state.isCreatingInstance"
              @click.prevent="cancel"
            >
              {{ $t("common.cancel") }}
            </button>
            <button
              type="button"
              class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
              :disabled="!allowCreate || state.isCreatingInstance"
              @click.prevent="tryCreate"
            >
              {{ $t("common.create") }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </form>
  <BBAlert
    v-if="state.showCreateInstanceWarningModal"
    :style="'WARN'"
    :ok-text="$t('instance.ignore-and-create')"
    :title="$t('instance.connection-info-seems-to-be-incorrect')"
    :description="state.createInstanceWarning"
    @ok="
      () => {
        state.showCreateInstanceWarningModal = false;
        doCreate();
      }
    "
    @cancel="state.showCreateInstanceWarningModal = false"
  ></BBAlert>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useRouter } from "vue-router";
import EnvironmentSelect from "./EnvironmentSelect.vue";
import CreateDataSourceExample from "./CreateDataSourceExample.vue";
import { SslCertificateForm } from "./InstanceForm";
import { instanceSlug, isDev } from "../utils";
import {
  InstanceCreate,
  UNKNOWN_ID,
  ConnectionInfo,
  SQLResultSet,
  EngineType,
} from "../types";
import isEmpty from "lodash-es/isEmpty";
import { useI18n } from "vue-i18n";
import { pushNotification, useInstanceStore, useSQLStore } from "@/store";

interface LocalState {
  instance: InstanceCreate;
  isCreatingInstance: boolean;
  showCreateInstanceWarningModal: boolean;
  createInstanceWarning: string;
}

const emit = defineEmits(["dismiss"]);

const router = useRouter();
const { t } = useI18n();
const sqlStore = useSQLStore();

const engineList: EngineType[] = [
  "MYSQL",
  "POSTGRES",
  "TIDB",
  "SNOWFLAKE",
  "CLICKHOUSE",
];

const EngineIconPath = {
  MYSQL: new URL("../assets/db-mysql.png", import.meta.url).href,
  POSTGRES: new URL("../assets/db-postgres.png", import.meta.url).href,
  TIDB: new URL("../assets/db-tidb.png", import.meta.url).href,
  SNOWFLAKE: new URL("../assets/db-snowflake.png", import.meta.url).href,
  CLICKHOUSE: new URL("../assets/db-clickhouse.png", import.meta.url).href,
};

const state = reactive<LocalState>({
  instance: {
    environmentId: UNKNOWN_ID,
    name: t("instance.new-instance"),
    engine: "MYSQL",
    // In dev mode, Bytebase is likely run in naked style and access the local network via 127.0.0.1.
    // In release mode, Bytebase is likely run inside docker and access the local network via host.docker.internal.
    host: isDev() ? "127.0.0.1" : "host.docker.internal",
    username: "",
    syncSchema: true,
  },
  showCreateInstanceWarningModal: false,
  createInstanceWarning: "",
  isCreatingInstance: false,
});

const allowCreate = computed(() => {
  return state.instance.name && state.instance.host;
});

const allowEdit = computed(() => {
  return true;
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

const showSSL = computed((): boolean => {
  return state.instance.engine === "CLICKHOUSE";
});

watch(showSSL, (ssl) => {
  // Clean up SSL options when they are not needed.
  if (!ssl) {
    state.instance.sslCa = "";
    state.instance.sslKey = "";
    state.instance.sslCert = "";
  }
});

const engineName = (type: EngineType): string => {
  switch (type) {
    case "CLICKHOUSE":
      return "ClickHouse";
    case "MYSQL":
      return "MySQL";
    case "POSTGRES":
      return "PostgreSQL";
    case "SNOWFLAKE":
      return "Snowflake";
    case "TIDB":
      return "TiDB";
  }
};

// The default host name is 127.0.0.1 or host.docker.internal which is not applicable to Snowflake, so we change
// the host name between 127.0.0.1/host.docker.internal and "" if user hasn't changed default yet.
const changeInstanceEngine = (engine: EngineType) => {
  if (engine == "SNOWFLAKE") {
    if (
      state.instance.host == "127.0.0.1" ||
      state.instance.host == "host.docker.internal"
    ) {
      state.instance.host = "";
    }
  } else {
    if (!state.instance.host) {
      state.instance.host = isDev() ? "127.0.0.1" : "host.docker.internal";
    }
  }
  state.instance.engine = engine;
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

const handleInstanceUsernameInput = (event: Event) => {
  updateInstance("username", (event.target as HTMLInputElement).value);
};

const handleInstancePasswordInput = (event: Event) => {
  updateInstance("password", (event.target as HTMLInputElement).value);
};

const updateInstance = (field: string, value: string) => {
  let str = value;
  if (
    field === "name" ||
    field === "host" ||
    field === "port" ||
    field === "externalLink" ||
    field === "username" ||
    field === "password"
  ) {
    str = value.trim();
  }
  (state.instance as any)[field] = str;
};

const cancel = () => {
  emit("dismiss");
};

const tryCreate = () => {
  const { instance } = state;
  const connectionInfo: ConnectionInfo = {
    engine: instance.engine,
    username: instance.username,
    password: instance.password,
    // When creating instance, the password is always needed.
    useEmptyPassword: false,
    host: instance.host,
    port: instance.port,
  };

  if (typeof instance.sslCa !== "undefined") {
    connectionInfo.sslCa = instance.sslCa;
  }
  if (typeof instance.sslKey !== "undefined") {
    connectionInfo.sslKey = instance.sslKey;
  }
  if (typeof instance.sslCert !== "undefined") {
    connectionInfo.sslCert = instance.sslCert;
  }

  sqlStore.ping(connectionInfo).then((resultSet: SQLResultSet) => {
    if (isEmpty(resultSet.error)) {
      doCreate();
    } else {
      state.createInstanceWarning = t("instance.unable-to-connect", [
        resultSet.error,
      ]);
      state.showCreateInstanceWarningModal = true;
    }
  });
};

// We will also create the database * denoting all databases
// and its RW data source. The username, password is actually
// stored in that data source object instead of in the instance self.
// Conceptually, data source is the proper place to store connection info (thinking of DSN)
const doCreate = () => {
  state.isCreatingInstance = true;
  useInstanceStore()
    .createInstance(state.instance)
    .then((createdInstance) => {
      state.isCreatingInstance = false;
      emit("dismiss");

      router.push(`/instance/${instanceSlug(createdInstance)}`);

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "instance.successfully-created-instance-createdinstance-name",
          [createdInstance.name]
        ),
      });
    });
};

const testConnection = () => {
  const { instance } = state;

  const connectionInfo: ConnectionInfo = {
    host: instance.host,
    port: instance.port,
    engine: instance.engine,
    username: instance.username,
    password: instance.password,
    useEmptyPassword: false,
    instanceId: undefined,
  };

  if (typeof instance.sslCa !== "undefined") {
    connectionInfo.sslCa = instance.sslCa;
  }
  if (typeof instance.sslKey !== "undefined") {
    connectionInfo.sslKey = instance.sslKey;
  }
  if (typeof instance.sslCert !== "undefined") {
    connectionInfo.sslCert = instance.sslCert;
  }

  sqlStore.ping(connectionInfo).then((resultSet: SQLResultSet) => {
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
