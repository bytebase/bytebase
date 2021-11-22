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

<template>
  <form class="space-y-6 divide-y divide-block-border">
    <div class="space-y-6 divide-y divide-block-border px-1">
      <div v-if="create" class="grid grid-cols-1 gap-4 sm:grid-cols-6">
        <template
          v-for="(engine, index) in [
            'MYSQL',
            'POSTGRES',
            'TIDB',
            'SNOWFLAKE',
            'CLICKHOUSE',
          ]"
          :key="index"
        >
          <div
            class="
              flex
              justify-center
              px-2
              py-4
              border border-control-border
              hover:bg-control-bg-hover
              cursor-pointer
            "
            @click.capture="changeInstanceEngine(engine)"
          >
            <div class="flex flex-col items-center">
              <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
              <template v-if="engine == 'MYSQL'">
                <img class="h-8 w-auto" src="../assets/db-mysql.png" alt="" />
              </template>
              <template v-else-if="engine == 'POSTGRES'">
                <img
                  class="h-8 w-auto"
                  src="../assets/db-postgres.png"
                  alt=""
                />
              </template>
              <template v-else-if="engine == 'TIDB'">
                <img class="h-8 w-auto" src="../assets/db-tidb.png" />
              </template>
              <template v-else-if="engine == 'SNOWFLAKE'">
                <img
                  class="h-8 w-auto"
                  src="../assets/db-snowflake.png"
                  alt=""
                />
              </template>
              <template v-else-if="engine == 'CLICKHOUSE'">
                <img
                  class="h-8 w-auto"
                  src="../assets/db-clickhouse.png"
                  alt=""
                />
              </template>
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
      <div
        class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4"
        :class="create ? 'pt-4' : ''"
      >
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="name" class="textlabel flex flex-row items-center">
            Instance Name&nbsp;<span style="color: red">*</span>
            <template v-if="!create">
              <InstanceEngineIcon class="ml-1" :instance="state.instance" />
              <span class="ml-1">{{ state.instance.engineVersion }}</span>
            </template>
          </label>
          <input
            required
            id="name"
            name="name"
            type="text"
            class="textfield mt-1 w-full"
            :disabled="!allowEdit"
            :value="state.instance.name"
            @input="updateInstance('name', $event.target.value)"
          />
        </div>

        <div class="sm:col-span-2 sm:col-start-1">
          <label for="environment" class="textlabel">
            Environment <span v-if="create" style="color: red">*</span>
          </label>
          <!-- Disallow changing environment after creation. This is to take the conservative approach to limit capability -->
          <EnvironmentSelect
            class="mt-1 w-full"
            id="environment"
            name="environment"
            :disabled="!create"
            :selectedID="
              create
                ? state.instance.environmentID
                : state.instance.environment.id
            "
            @select-environment-id="
              (environmentID) => {
                if (create) {
                  state.instance.environmentID = environmentID;
                } else {
                  updateInstance('environmentID', environmentID);
                }
              }
            "
          />
        </div>

        <div class="sm:col-span-3 sm:col-start-1">
          <label for="host" class="textlabel block">
            <template v-if="state.instance.engine == 'SNOWFLAKE'">
              Account name <span style="color: red">*</span>
            </template>
            <template v-else>
              Host or Socket <span style="color: red">*</span>
            </template>
          </label>
          <input
            required
            type="text"
            id="host"
            name="host"
            :placeholder="
              state.instance.engine == 'SNOWFLAKE'
                ? 'your Snowflake account name'
                : 'e.g. host.docker.internal | <<ip>> | <<local socket>>'
            "
            class="textfield mt-1 w-full"
            :disabled="!allowEdit"
            :value="state.instance.host"
            @input="updateInstance('host', $event.target.value)"
          />
          <div
            v-if="state.instance.engine == 'SNOWFLAKE'"
            class="mt-2 textinfolabel"
          >
            For proxy server, append @PROXY_HOST and specify PROXY_PORT in the
            port
          </div>
        </div>

        <div class="sm:col-span-1">
          <label for="port" class="textlabel block"> Port </label>
          <input
            type="number"
            id="port"
            name="port"
            class="textfield mt-1 w-full"
            :placeholder="defaultPort"
            :disabled="!allowEdit"
            :value="state.instance.port"
            @input="updateInstance('port', $event.target.value)"
          />
        </div>

        <!--Do not show external link on create to reduce cognitive load-->
        <div v-if="!create" class="sm:col-span-3 sm:col-start-1">
          <label for="externallink" class="textlabel inline-flex">
            <span class="">
              {{
                state.instance.engine == "SNOWFLAKE"
                  ? "Snowflake Web Console"
                  : "External Link"
              }}</span
            >
            <button
              class="ml-1 btn-icon"
              :disabled="instanceLink(state.instance)?.trim().length == 0"
              @click.prevent="
                window.open(urlfy(instanceLink(state.instance)), '_blank')
              "
            >
              <svg
                class="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                ></path>
              </svg>
            </button>
          </label>
          <template v-if="state.instance.engine == 'SNOWFLAKE'">
            <input
              required
              id="externallink"
              name="externallink"
              type="text"
              class="textfield mt-1 w-full"
              disabled="true"
              :value="instanceLink(state.instance)"
            />
          </template>
          <template v-else>
            <div class="mt-1 textinfolabel">
              The external console URL managing this instance (e.g. AWS RDS
              console, your in-house DB instance console)
            </div>
            <input
              required
              id="externallink"
              name="externallink"
              type="text"
              class="textfield mt-1 w-full"
              placeholder="https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=mysql-instance-foo;is-cluster=false"
              :disabled="!allowEdit"
              :value="state.instance.externalLink"
              @input="updateInstance('externalLink', $event.target.value)"
            />
          </template>
        </div>
      </div>
      <!-- Read/Write Connection Info -->
      <div class="pt-4">
        <div class="flex justify-between">
          <div>
            <h3 class="text-lg leading-6 font-medium text-gray-900">
              Connection info
            </h3>
            <p
              class="mt-1 text-sm text-gray-500"
              :class="create ? 'max-w-xl' : ''"
            >
              This is the connection user used by Bytebase to perform DDL and
              DML operations.
              <span
                v-if="!create"
                class="normal-link"
                @click="toggleCreateUserExample"
                >Show how to create</span
              >
            </p>
            <!-- Specify the fixed width so the create instance dialog width won't shift when switching engine types-->
            <div
              v-if="state.showCreateUserExample"
              class="mt-2 text-sm text-main w-208"
            >
              <template
                v-if="
                  state.instance.engine == 'MYSQL' ||
                  state.instance.engine == 'TIDB'
                "
              >
                Below is an example to create user 'bytebase@%' with password
                <span class="text-red-600">YOUR_DB_PWD</span> and grant the user
                with the needed privileges.
              </template>
              <template v-else-if="state.instance.engine == 'CLICKHOUSE'">
                Below is an example to create user 'bytebase' with password
                <span class="text-red-600">YOUR_DB_PWD</span> and grant the user
                with the needed privileges. First you need to enable
                <a
                  class="normal-link"
                  href="https://clickhouse.com/docs/en/operations/access-rights/#access-control-usage"
                  target="__blank"
                >
                  ClickHouse SQL-driven workflow</a
                >
                and then run the following query to create the user.
              </template>
              <template v-else-if="state.instance.engine == 'POSTGRES'">
                <BBAttention
                  class="mb-1"
                  :style="'WARN'"
                  :title="'If the connecting instance is managed by the cloud provider, then SUPERUSER is not available and you should create the user via that provider\'s admin console. The created user will have provider specific semi-SUPERUSER privileges.'"
                />
                Below is an example to create user 'bytebase' with password
                <span class="text-red-600">YOUR_DB_PWD</span> and grant the user
                with the needed privileges. If the connecting instance is
                self-hosted, then you can grant SUPERUSER.
              </template>
              <template v-else-if="state.instance.engine == 'SNOWFLAKE'">
                Below is an example to create user 'bytebase' with password
                <span class="text-red-600">YOUR_DB_PWD</span> for
                <span class="text-red-600">YOUR_COMPUTE_WAREHOUSE</span> and
                grant the user with the needed privileges.
              </template>
              <div class="mt-2 flex flex-row">
                <span
                  class="
                    flex-1
                    min-w-0
                    w-full
                    inline-flex
                    items-center
                    px-3
                    py-2
                    border border-r border-control-border
                    bg-gray-50
                    sm:text-sm
                    whitespace-pre
                  "
                >
                  {{ grantStatement(state.instance.engine) }}
                </span>
                <button
                  tabindex="-1"
                  class="
                    -ml-px
                    px-2
                    py-2
                    border border-gray-300
                    text-sm
                    font-medium
                    text-control-light
                    disabled:text-gray-300
                    bg-gray-50
                    hover:bg-gray-100
                    disabled:bg-gray-50
                    focus:ring-control focus:outline-none
                    focus-visible:ring-2
                    focus:ring-offset-1
                    disabled:cursor-not-allowed
                  "
                  @click.prevent="copyGrantStatement"
                >
                  <svg
                    class="w-6 h-6"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                    ></path>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="pt-4 grid grid-cols-1 gap-y-4 gap-x-4 sm:grid-cols-3">
          <div class="sm:col-span-1 sm:col-start-1">
            <label for="username" class="textlabel block"> Username </label>
            <!-- For mysql, username can be empty indicating anonymous user. 
            But it's a very bad practice to use anonymous user for admin operation,
            thus we make it REQUIRED here. -->
            <input
              id="username"
              name="username"
              type="text"
              class="textfield mt-1 w-full"
              :disabled="!allowEdit"
              :placeholder="
                state.instance.engine == 'CLICKHOUSE' ? 'default' : ''
              "
              :value="state.instance.username"
              @input="state.instance.username = $event.target.value"
            />
          </div>

          <div class="sm:col-span-1 sm:col-start-1">
            <div class="flex flex-row items-center space-x-2">
              <label for="password" class="textlabel block">Password</label>
              <!-- In create mode, user can leave the password field empty and create the instance,
              so there is no need to show the checkbox. -->
              <BBCheckbox
                v-if="!create"
                :title="'Empty'"
                :value="state.useEmptyPassword"
                @toggle="
                  (on) => {
                    state.useEmptyPassword = on;
                  }
                "
              />
            </div>
            <input
              id="password"
              name="password"
              type="text"
              class="textfield mt-1 w-full"
              autocomplete="off"
              :placeholder="
                state.useEmptyPassword
                  ? 'No password'
                  : 'YOUR_DB_PWD - write only'
              "
              :disabled="!allowEdit || state.useEmptyPassword"
              :value="
                create
                  ? state.useEmptyPassword
                    ? ''
                    : state.instance.password
                  : state.useEmptyPassword
                  ? ''
                  : state.updatedPassword
              "
              @input="
                create
                  ? (state.instance.password = $event.target.value)
                  : (state.updatedPassword = $event.target.value)
              "
            />
          </div>
        </div>
        <div v-if="showTestConnection" class="pt-8 space-y-2">
          <div class="flex flex-row space-x-2">
            <button
              @click.prevent="testConnection"
              type="button"
              class="btn-normal whitespace-nowrap items-center"
              :disabled="!state.instance.host"
            >
              Test Connection
            </button>
          </div>
        </div>
      </div>
    </div>
    <!-- Action Button Group -->
    <div class="pt-4">
      <!-- Create button group -->
      <div v-if="create" class="flex justify-end items-center">
        <div>
          <BBSpin v-if="state.creatingOrUpdating" :title="'Creating...'" />
        </div>
        <div class="ml-2">
          <button
            type="button"
            class="btn-normal py-2 px-4"
            :disabled="state.creatingOrUpdating"
            @click.prevent="cancel"
          >
            Cancel
          </button>
          <button
            type="button"
            class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
            :disabled="!allowCreate || state.creatingOrUpdating"
            @click.prevent="tryCreate"
          >
            Create
          </button>
        </div>
      </div>
      <!-- Update button group -->
      <div v-else class="flex justify-end items-center">
        <div>
          <BBSpin v-if="state.creatingOrUpdating" :title="'Updating...'" />
        </div>
        <button
          v-if="allowEdit"
          type="button"
          class="btn-normal ml-2 inline-flex justify-center py-2 px-4"
          :disabled="!valueChanged || state.creatingOrUpdating"
          @click.prevent="doUpdate"
        >
          Update
        </button>
      </div>
    </div>
  </form>
  <BBAlert
    v-if="state.showCreateInstanceWarningModal"
    :style="'WARN'"
    :okText="'Ignore and create'"
    :title="'Connection info seems to be incorrect'"
    :description="state.createInstanceWarning"
    @ok="
      () => {
        state.showCreateInstanceWarningModal = false;
        doCreate();
      }
    "
    @cancel="state.showCreateInstanceWarningModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { computed, reactive, PropType, ComputedRef } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import InstanceEngineIcon from "../components/InstanceEngineIcon.vue";
import { instanceSlug, isDBAOrOwner, isDev } from "../utils";
import {
  Instance,
  InstanceCreate,
  UNKNOWN_ID,
  Principal,
  InstancePatch,
  ConnectionInfo,
  SqlResultSet,
  EngineType,
} from "../types";
import isEmpty from "lodash-es/isEmpty";

interface LocalState {
  originalInstance?: Instance;
  instance: Instance | InstanceCreate;
  // Only used in non-create case.
  updatedPassword: string;
  useEmptyPassword: boolean;
  showCreateInstanceWarningModal: boolean;
  createInstanceWarning: string;
  showCreateUserExample: boolean;
  creatingOrUpdating: boolean;
}

export default {
  name: "DataSourceCreateForm",
  emits: ["dismiss"],
  props: {
    create: {
      default: "false",
      type: Boolean,
    },
    instance: {
      // Can be false when create is true
      required: false,
      type: Object as PropType<Instance>,
    },
  },
  components: { EnvironmentSelect, InstanceEngineIcon },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const currentUser: ComputedRef<Principal> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    const state = reactive<LocalState>({
      originalInstance: props.instance,
      // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
      instance: props.instance
        ? cloneDeep(props.instance)
        : {
            environmentID: UNKNOWN_ID,
            name: "New Instance",
            engine: "MYSQL",
            // In dev mode, Bytebase is likely run in naked style and access the local network via 127.0.0.1.
            // In release mode, Bytebase is likely run inside docker and access the local network via host.docker.internal.
            host: isDev() ? "127.0.0.1" : "host.docker.internal",
            username: "",
          },
      updatedPassword: "",
      useEmptyPassword: false,
      showCreateInstanceWarningModal: false,
      createInstanceWarning: "",
      showCreateUserExample: props.create,
      creatingOrUpdating: false,
    });

    const allowCreate = computed(() => {
      return state.instance.name && state.instance.host;
    });

    const allowEdit = computed(() => {
      return (
        props.create ||
        ((state.instance as Instance).rowStatus == "NORMAL" &&
          isDBAOrOwner(currentUser.value.role))
      );
    });

    const showTestConnection = computed(() => {
      return (
        props.create ||
        ((state.instance as Instance).rowStatus == "NORMAL" &&
          isDBAOrOwner(currentUser.value.role))
      );
    });

    const valueChanged = computed(() => {
      return (
        !isEqual(state.instance, state.originalInstance) ||
        !isEmpty(state.updatedPassword) ||
        state.useEmptyPassword
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

    const grantStatement = (type: EngineType): string => {
      switch (type) {
        case "CLICKHOUSE":
          return "CREATE USER bytebase IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALL ON *.* TO bytebase WITH GRANT OPTION;";
        case "SNOWFLAKE":
          return "CREATE OR REPLACE USER bytebase PASSWORD = 'YOUR_DB_PWD'\nDEFAULT_ROLE = \"ACCOUNTADMIN\"\nDEFAULT_WAREHOUSE = 'YOUR_COMPUTE_WAREHOUSE';\n\nGRANT ROLE \"ACCOUNTADMIN\" TO USER bytebase;";
        case "MYSQL":
        case "TIDB":
          return "CREATE USER bytebase@'%' IDENTIFIED BY 'YOUR_DB_PWD';\n\nGRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, \nDELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, \nSELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE \nON *.* to bytebase@'%';";
        case "POSTGRES":
          return "CREATE USER bytebase WITH ENCRYPTED PASSWORD 'YOUR_DB_PWD';\n\nALTER USER bytebase WITH SUPERUSER;";
      }
    };

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

    const toggleCreateUserExample = () => {
      state.showCreateUserExample = !state.showCreateUserExample;
    };

    const updateInstance = (field: string, value: string) => {
      (state.instance as any)[field] = value;
    };

    const cancel = () => {
      emit("dismiss");
    };

    const tryCreate = () => {
      const connectionInfo: ConnectionInfo = {
        engine: state.instance.engine,
        username: state.instance.username,
        password: state.useEmptyPassword ? "" : state.instance.password,
        useEmptyPassword: state.useEmptyPassword,
        host: state.instance.host,
        port: state.instance.port,
      };
      store
        .dispatch("sql/ping", connectionInfo)
        .then((resultSet: SqlResultSet) => {
          if (isEmpty(resultSet.error)) {
            doCreate();
          } else {
            state.createInstanceWarning = `Bytebase is unable to connect the instance. We recommend you to review the connection info again. But it's OK to ignore this warning for now. You can still fix the connection info from the instance detail page after creation.\n\nError detail: ${resultSet.error}`;
            state.showCreateInstanceWarningModal = true;
          }
        });
    };

    // We will also create the database * denoting all databases
    // and its RW data source. The username, password is actually
    // stored in that data source object instead of in the instance self.
    // Conceptually, data source is the proper place to store connnection info (thinking of DSN)
    const doCreate = () => {
      state.creatingOrUpdating = true;
      if (state.useEmptyPassword) {
        state.instance.password = "";
      }
      store
        .dispatch("instance/createInstance", state.instance)
        .then((createdInstance) => {
          state.creatingOrUpdating = false;
          state.useEmptyPassword = false;
          emit("dismiss");

          router.push(`/instance/${instanceSlug(createdInstance)}`);

          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully created instance '${createdInstance.name}'.`,
          });

          // After creating the instance, we will check if migration schema exists on the instance.
          setTimeout(() => {}, 1000);
        });
    };

    const doUpdate = () => {
      const patchedInstance: InstancePatch = { useEmptyPassword: false };
      let connectionInfoChanged = false;
      if (state.instance.name != state.originalInstance!.name) {
        patchedInstance.name = state.instance.name;
      }
      if (state.instance.externalLink != state.originalInstance!.externalLink) {
        patchedInstance.externalLink = state.instance.externalLink;
      }
      if (state.instance.host != state.originalInstance!.host) {
        patchedInstance.host = state.instance.host;
        connectionInfoChanged = true;
      }
      if (state.instance.port != state.originalInstance!.port) {
        patchedInstance.port = state.instance.port;
        connectionInfoChanged = true;
      }
      if (state.instance.username != state.originalInstance!.username) {
        patchedInstance.username = state.instance.username;
        connectionInfoChanged = true;
      }
      if (state.useEmptyPassword) {
        patchedInstance.useEmptyPassword = true;
        connectionInfoChanged = true;
      } else if (!isEmpty(state.updatedPassword)) {
        patchedInstance.password = state.updatedPassword;
        connectionInfoChanged = true;
      }

      state.creatingOrUpdating = true;
      store
        .dispatch("instance/patchInstance", {
          instanceID: (state.instance as Instance).id,
          instancePatch: patchedInstance,
        })
        .then((instance) => {
          state.creatingOrUpdating = false;
          state.originalInstance = instance;
          // Make hard copy since we are going to make equal comparsion to determine the update button enable state.
          state.instance = cloneDeep(state.originalInstance!);
          state.updatedPassword = "";
          state.useEmptyPassword = false;

          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully updated instance '${instance.name}'.`,
          });

          // Backend will sync the schema upon connection info change, so here we try to fetch the synced schema.
          if (connectionInfoChanged) {
            store.dispatch(
              "database/fetchDatabaseListByInstanceID",
              instance.id
            );
          }
        });
    };

    const copyGrantStatement = () => {
      toClipboard(grantStatement(state.instance.engine)).then(() => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: `CREATE USER and GRANT statements copied to clipboard. Paste to your mysql client to apply.`,
        });
      });
    };

    const testConnection = () => {
      const connectionInfo: ConnectionInfo = {
        engine: state.instance.engine,
        username: state.instance.username,
        password: props.create
          ? state.useEmptyPassword
            ? ""
            : state.instance.password
          : state.useEmptyPassword
          ? ""
          : state.updatedPassword,
        useEmptyPassword: state.useEmptyPassword,
        host: state.instance.host,
        port: state.instance.port,
        instanceID: props.create ? undefined : (state.instance as Instance).id,
      };
      store
        .dispatch("sql/ping", connectionInfo)
        .then((resultSet: SqlResultSet) => {
          if (isEmpty(resultSet.error)) {
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "SUCCESS",
              title: `Successfully connected instance.`,
            });
          } else {
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "CRITICAL",
              title: `Failed to connect instance.`,
              description: resultSet.error,
              // Manual hide, because user may need time to inspect the error
              manualHide: true,
            });
          }
        });
    };

    return {
      state,
      allowCreate,
      allowEdit,
      showTestConnection,
      valueChanged,
      defaultPort,
      engineName,
      grantStatement,
      instanceLink,
      changeInstanceEngine,
      toggleCreateUserExample,
      updateInstance,
      cancel,
      tryCreate,
      doCreate,
      doUpdate,
      copyGrantStatement,
      testConnection,
    };
  },
};
</script>
