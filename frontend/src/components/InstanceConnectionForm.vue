<template>
  <div class="pt-4">
    <div class="flex justify-between">
      <div>
        <h3 class="text-lg leading-6 font-medium text-gray-900">{{ $t("instance.connection-info") }}</h3>
        <p class="mt-1 text-sm text-gray-500" :class="create ? 'max-w-xl' : ''">
          {{ $t("instance.sentence.create-user") }}
          <span
            v-if="!create"
            class="normal-link"
            @click="toggleCreateUserExample"
          >{{ $t("instance.show-how-to-create") }}</span>
        </p>
        <!-- Specify the fixed width so the create instance dialog width won't shift when switching engine types-->
        <div v-if="state.showCreateUserExample" class="mt-2 text-sm text-main w-208">
          <template
            v-if="
              instance.engine == 'MYSQL' ||
              instance.engine == 'TIDB'
            "
          >
            <i18n-t tag="p" keypath="instance.sentence.create-user-example.mysql.template">
              <template #user>{{ $t("instance.sentence.create-user-example.mysql.user") }}</template>
              <template #password>
                <span class="text-red-600">
                  {{
                    $t(
                      "instance.sentence.create-user-example.mysql.password"
                    )
                  }}
                </span>
              </template>
            </i18n-t>
          </template>
          <template v-else-if="instance.engine == 'CLICKHOUSE'">
            <i18n-t tag="p" keypath="instance.sentence.create-user-example.clickhouse.template">
              <template #password>
                <span class="text-red-600">YOUR_DB_PWD</span>
              </template>
              <template #link>
                <a
                  class="normal-link"
                  href="https://clickhouse.com/docs/en/operations/access-rights/#access-control-usage"
                  target="__blank"
                >
                  {{
                    $t(
                      "instance.sentence.create-user-example.clickhouse.sql-driven-workflow"
                    )
                  }}
                </a>
              </template>
            </i18n-t>
          </template>
          <template v-else-if="instance.engine == 'POSTGRES'">
            <BBAttention
              class="mb-1"
              :style="'WARN'"
              :title="
                $t('instance.sentence.create-user-example.postgres.warn')
              "
            />
            <i18n-t tag="p" keypath="instance.sentence.create-user-example.postgres.template">
              <template #password>
                <span class="text-red-600">YOUR_DB_PWD</span>
              </template>
            </i18n-t>
          </template>
          <template v-else-if="instance.engine == 'SNOWFLAKE'">
            <i18n-t tag="p" keypath="instance.sentence.create-user-example.snowflake.template">
              <template #password>
                <span class="text-red-600">YOUR_DB_PWD</span>
              </template>
              <template #warehouse>
                <span class="text-red-600">YOUR_COMPUTE_WAREHOUSE</span>
              </template>
            </i18n-t>
          </template>
          <div class="mt-2 flex flex-row">
            <span
              class="flex-1 min-w-0 w-full inline-flex items-center px-3 py-2 border border-r border-control-border bg-gray-50 sm:text-sm whitespace-pre"
            >{{ grantStatement(instance.engine) }}</span>
            <button
              tabindex="-1"
              class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light disabled:text-gray-300 bg-gray-50 hover:bg-gray-100 disabled:bg-gray-50 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1 disabled:cursor-not-allowed"
              @click.prevent="copyGrantStatement"
            >
              <heroicons-outline:clipboard class="w-6 h-6" />
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="pt-4 grid grid-cols-1 gap-y-4 gap-x-4 sm:grid-cols-3">
      <div class="sm:col-span-1 sm:col-start-1">
        <label for="username" class="textlabel block">{{ $t("common.username") }}</label>
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
            instance.engine == 'CLICKHOUSE'
              ? $t('common.default')
              : ''
          "
          :value="instance.username"
          @input="$emit('update-username', $event.target.value)"
        />
      </div>

      <div class="sm:col-span-1 sm:col-start-1">
        <div class="flex flex-row items-center space-x-2">
          <label for="password" class="textlabel block">
            {{
              $t("common.password")
            }}
          </label>
          <!-- In create mode, user can leave the password field empty and create the instance,
          so there is no need to show the checkbox.-->
          <BBCheckbox
            v-if="!create"
            :title="$t('common.empty')"
            :value="state.useEmptyPassword"
            @toggle="
              (on) => {
                state.useEmptyPassword = on;
                $emit('toggle-empty-password', on);
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
              ? $t('instance.no-password')
              : $t('instance.password-write-only')
          "
          :disabled="!allowEdit || state.useEmptyPassword"
          :value="
            create
              ? state.useEmptyPassword
                ? ''
                : instance.password
              : state.useEmptyPassword
                ? ''
                : state.updatedPassword
          "
          @input="(e) => {
            state.updatedPassword = e.target.value
            $emit('update-password', e.target.value)
          }"
        />
      </div>
    </div>
    <div v-if="showTestConnection" class="pt-8 space-y-2">
      <div class="flex flex-row space-x-2">
        <button
          type="button"
          class="btn-normal whitespace-nowrap items-center"
          :disabled="!instance.host"
          @click.prevent="testConnection"
        >{{ $t("instance.test-connection") }}</button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive, watch, PropType, ComputedRef } from "vue";
import { useStore } from "vuex";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { isDBAOrOwner } from "../utils";
import {
  Instance,
  Principal,
  ConnectionInfo,
  SqlResultSet,
  EngineType,
} from "../types";
import isEmpty from "lodash-es/isEmpty";
import { useI18n } from "vue-i18n";

interface LocalState {
  // Only used in non-create case.
  updatedPassword: string;
  useEmptyPassword: boolean;
  showCreateUserExample: boolean;
}

export default {
  name: "InstanceConnectionForm",
  components: {},
  props: {
    create: {
      default: false,
      type: Boolean,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
    instance: {
      // Can be false when create is true
      required: false,
      type: Object as PropType<Instance>,
    },
  },
  emits: ["update-username", "update-password", "toggle-empty-password"],
  setup(props, { }) {
    const store = useStore();
    const { t } = useI18n();

    const currentUser: ComputedRef<Principal> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    const state = reactive<LocalState>({
      updatedPassword: "",
      useEmptyPassword: false,
      showCreateUserExample: props.create,
    });

    watch(
      () => props.instance,
      () => {
        state.updatedPassword = "";
        state.useEmptyPassword = false;
      }
    );

    const showTestConnection = computed(() => {
      return (
        props.create ||
        (props.instance.rowStatus == "NORMAL" &&
          isDBAOrOwner(currentUser.value.role))
      );
    });

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

    const toggleCreateUserExample = () => {
      state.showCreateUserExample = !state.showCreateUserExample;
    };

    const copyGrantStatement = () => {
      toClipboard(grantStatement(props.instance.engine)).then(() => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: t("instance.copy-grant-statement"),
        });
      });
    };

    const testConnection = () => {
      const connectionInfo: ConnectionInfo = {
        engine: props.instance.engine,
        username: props.instance.username,
        password: props.create
          ? state.useEmptyPassword
            ? ""
            : props.instance.password
          : state.useEmptyPassword
            ? ""
            : state.updatedPassword,
        useEmptyPassword: state.useEmptyPassword,
        host: props.instance.host,
        port: props.instance.port,
        instanceId: props.create ? undefined : (props.instance as Instance).id,
      };
      store
        .dispatch("sql/ping", connectionInfo)
        .then((resultSet: SqlResultSet) => {
          if (isEmpty(resultSet.error)) {
            store.dispatch("notification/pushNotification", {
              module: "bytebase",
              style: "SUCCESS",
              title: t("instance.successfully-connected-instance"),
            });
          } else {
            store.dispatch("notification/pushNotification", {
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

    return {
      state,
      showTestConnection,
      grantStatement,
      toggleCreateUserExample,
      copyGrantStatement,
      testConnection,
    };
  },
};
</script>