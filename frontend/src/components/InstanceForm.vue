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
            class="flex justify-center px-2 py-4 border border-control-border hover:bg-control-bg-hover cursor-pointer"
            @click.capture="changeInstanceEngine(engine)"
          >
            <div class="flex flex-col items-center">
              <img class="h-8 w-auto" :src="EngineIconPath[engine]" alt />
              <p class="mt-1 text-center textlabel">{{ engineName(engine) }}</p>
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
            {{ $t("instance.instance-name") }}
            &nbsp;
            <span style="color: red">*</span>
            <template v-if="!create">
              <InstanceEngineIcon class="ml-1" :instance="state.instance" />
              <span class="ml-1">{{ state.instance.engineVersion }}</span>
            </template>
          </label>
          <input
            id="name"
            required
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
            {{ $t("common.environment") }}
            <span v-if="create" style="color: red">*</span>
          </label>
          <!-- Disallow changing environment after creation. This is to take the conservative approach to limit capability -->
          <!-- eslint-disable vue/attribute-hyphenation -->
          <EnvironmentSelect
            id="environment"
            class="mt-1 w-full"
            name="environment"
            :disabled="!create"
            :selectedId="
              create
                ? state.instance.environmentId
                : state.instance.environment.id
            "
            @select-environment-id="
              (environmentId) => {
                if (create) {
                  state.instance.environmentId = environmentId;
                } else {
                  updateInstance('environmentId', environmentId);
                }
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
            @input="updateInstance('host', $event.target.value)"
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
            @input="updateInstance('port', $event.target.value)"
          />
        </div>

        <!--Do not show external link on create to reduce cognitive load-->
        <div v-if="!create" class="sm:col-span-3 sm:col-start-1">
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
      <InstanceConnectionForm
        :create="create"
        :allowEdit="allowEdit"
        :instance="state.instance"
        @update-username="updateUsername"
        @update-password="updatePassword"
        @toggle-empty-password="toggleEmptyPassword"
      />
    </div>
    <!-- Action Button Group -->
    <div class="pt-4">
      <!-- Create button group -->
      <div v-if="create" class="flex justify-end items-center">
        <div>
          <BBSpin
            v-if="state.creatingOrUpdating"
            :title="$t('common.creating')"
          />
        </div>
        <div class="ml-2">
          <button
            type="button"
            class="btn-normal py-2 px-4"
            :disabled="state.creatingOrUpdating"
            @click.prevent="cancel"
          >
            {{ $t("common.cancel") }}
          </button>
          <button
            type="button"
            class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
            :disabled="!allowCreate || state.creatingOrUpdating"
            @click.prevent="tryCreate"
          >
            {{ $t("common.create") }}
          </button>
        </div>
      </div>
      <!-- Update button group -->
      <div v-else class="flex justify-end items-center">
        <div>
          <BBSpin
            v-if="state.creatingOrUpdating"
            :title="$t('common.updating')"
          />
        </div>
        <button
          v-if="allowEdit"
          type="button"
          class="btn-normal ml-2 inline-flex justify-center py-2 px-4"
          :disabled="!valueChanged || state.creatingOrUpdating"
          @click.prevent="doUpdate"
        >
          {{ $t("common.update") }}
        </button>
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

<script lang="ts">
import { computed, reactive, PropType, ComputedRef } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import InstanceConnectionForm from "../components/InstanceConnectionForm.vue";
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
import { useI18n } from "vue-i18n";

interface LocalState {
  originalInstance?: Instance;
  instance: Instance | InstanceCreate;
  // Only used in non-create case.
  updatedPassword: string;
  useEmptyPassword: boolean;
  showCreateInstanceWarningModal: boolean;
  createInstanceWarning: string;
  creatingOrUpdating: boolean;
}

export default {
  name: "InstanceForm",
  components: { EnvironmentSelect, InstanceConnectionForm, InstanceEngineIcon },
  props: {
    create: {
      default: false,
      type: Boolean,
    },
    instance: {
      // Can be false when create is true
      required: false,
      type: Object as PropType<Instance>,
    },
  },
  emits: ["dismiss"],
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();
    const { t } = useI18n();

    const currentUser: ComputedRef<Principal> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    const EngineIconPath = {
      MYSQL: new URL("../assets/db-mysql.png", import.meta.url).href,
      POSTGRES: new URL("../assets/db-postgres.png", import.meta.url).href,
      TIDB: new URL("../assets/db-tidb.png", import.meta.url).href,
      SNOWFLAKE: new URL("../assets/db-snowflake.png", import.meta.url).href,
      CLICKHOUSE: new URL("../assets/db-clickhouse.png", import.meta.url).href,
    };

    const state = reactive<LocalState>({
      originalInstance: props.instance,
      // Make hard copy since we are going to make equal comparison to determine the update button enable state.
      instance: props.instance
        ? cloneDeep(props.instance)
        : {
            environmentId: UNKNOWN_ID,
            name: t("instance.new-instance"),
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

    const updateUsername = (username: string) => {
      state.instance.username = username;
    };

    const updatePassword = (password: string) => {
      if (props.create) {
        state.instance.password = password;
      } else {
        state.updatedPassword = password;
      }
    };

    const toggleEmptyPassword = (useEmptyPassword: boolean) => {
      state.useEmptyPassword = useEmptyPassword;
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
            title: t(
              "instance.successfully-created-instance-createdinstance-name",
              [createdInstance.name]
            ),
          });

          // After creating the instance, we will check if migration schema exists on the instance.
          // setTimeout(() => {}, 1000);
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
          instanceId: (state.instance as Instance).id,
          instancePatch: patchedInstance,
        })
        .then((instance) => {
          state.creatingOrUpdating = false;
          state.originalInstance = instance;
          // Make hard copy since we are going to make equal comparison to determine the update button enable state.
          state.instance = cloneDeep(state.originalInstance!);
          state.updatedPassword = "";
          state.useEmptyPassword = false;

          store.dispatch("notification/pushNotification", {
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
              instance.id
            );
          }
        });
    };

    return {
      state,
      allowCreate,
      allowEdit,
      valueChanged,
      EngineIconPath,
      defaultPort,
      engineName,
      instanceLink,
      changeInstanceEngine,
      updateUsername,
      updatePassword,
      toggleEmptyPassword,
      updateInstance,
      cancel,
      tryCreate,
      doCreate,
      doUpdate,
    };
  },
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
