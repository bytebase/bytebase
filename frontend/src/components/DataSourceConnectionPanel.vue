<template>
  <!-- Description list -->
  <form class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2">
    <dl class="sm:col-span-2">
      <dt class="flex items-center space-x-1">
        <div class="text-sm font-medium text-control-light">
          {{ $t("common.connection-string") }}
        </div>
        <button
          class="btn-icon"
          @click.prevent="state.showPassword = !state.showPassword"
        >
          <heroicons-outline:eye-off
            v-if="state.showPassword"
            class="w-5 h-5"
          />
          <heroicons-outline:eye v-else class="w-5 h-5" />
        </button>
      </dt>
      <dd class="mt-2 text-sm text-main">
        <div class="space-y-4">
          <div
            v-for="(connection, index) in connectionStringList"
            :key="index"
            class="flex"
          >
            <span
              class="whitespace-nowrap inline-flex items-center px-3 rounded-l-md border border-l border-r-0 border-control-border bg-gray-50 text-control-light sm:text-sm"
            >
              {{ connection.name }}
            </span>
            <span
              class="flex-1 min-w-0 w-full inline-flex items-center px-3 py-2 border border-r border-control-border bg-gray-50 sm:text-sm"
            >
              {{ connection.value }}
            </span>
            <button
              class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light bg-gray-50 hover:bg-gray-100 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
              @click.prevent="copyText(connection)"
            >
              <heroicons-outline:clipboard class="w-6 h-6" />
            </button>
          </div>
        </div>
      </dd>
    </dl>

    <dl class="sm:col-span-1">
      <dt class="text-sm font-medium text-control-light">
        {{ $t("common.username") }}
      </dt>
      <dd class="mt-1 text-sm text-main">
        <input
          v-if="editing"
          id="username"
          v-model="dataSource.username"
          required
          type="text"
          class="textfield"
        />
        <div v-else class="mt-2.5 mb-3">
          {{ dataSource.username }}
        </div>
      </dd>
    </dl>

    <dl class="sm:col-span-1">
      <dt class="text-sm font-medium text-control-light">
        {{ $t("common.password") }}
      </dt>
      <dd class="mt-1 text-sm text-main">
        <input
          v-if="editing"
          id="password"
          v-model="dataSource.password"
          required
          autocomplete="off"
          :type="state.showPassword ? 'text' : 'password'"
          class="textfield"
        />
        <div v-else class="mt-2.5 mb-3">
          <template v-if="state.showPassword">
            {{ dataSource.password }}
          </template>
          <template v-else> ****** </template>
        </div>
      </dd>
    </dl>

    <dl class="sm:col-span-1">
      <dt class="text-sm font-medium text-control-light">
        {{ $t("common.updated-at") }}
      </dt>
      <dd class="mt-1 text-sm text-main">
        {{ humanizeTs(dataSource.updatedTs) }}
      </dd>
    </dl>

    <dl class="sm:col-span-1">
      <dt class="text-sm font-medium text-control-light">
        {{ $t("common.created-at") }}
      </dt>
      <dd class="mt-1 text-sm text-main">
        {{ humanizeTs(dataSource.createdTs) }}
      </dd>
    </dl>
  </form>
</template>

<script lang="ts">
import { computed, reactive, PropType, defineComponent } from "vue";
import { useStore } from "vuex";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { DataSource } from "../types";
import { useI18n } from "vue-i18n";
import { useNotificationStore } from "@/store";

type Connection = {
  name: string;
  value: string;
};

interface LocalState {
  showPassword: boolean;
}

export default defineComponent({
  name: "DataSourceConnectionPanel",
  components: {},
  props: {
    editing: {
      default: false,
      type: Boolean,
    },
    dataSource: {
      required: true,
      type: Object as PropType<DataSource>,
    },
  },
  setup(props) {
    const store = useStore();
    const notificationStore = useNotificationStore();

    const state = reactive<LocalState>({
      showPassword: false,
    });

    const { t } = useI18n();

    const database = computed(() => {
      return store.getters["database/databaseById"](
        props.dataSource.databaseId,
        props.dataSource.instanceId
      );
    });

    const connectionStringList = computed<Connection[]>(() => {
      // If host starts with "/", we assume it's a local socket.
      const isSocket = database.value.instance.host.startsWith("/");
      const cliOptionList = isSocket
        ? [`mysql -S ${database.value.instance.host}`]
        : [`mysql -h ${database.value.instance.host}`];
      if (database.value.instance.port) {
        cliOptionList.push(`-P ${database.value.instance.port}`);
      }
      if (database.value.name) {
        cliOptionList.push(`-D ${database.value.name}`);
      }
      if (props.dataSource.username) {
        cliOptionList.push(`-u ${props.dataSource.username}`);
      }
      if (props.dataSource.password) {
        if (state.showPassword) {
          cliOptionList.push(`-p${props.dataSource.password}`);
        } else {
          cliOptionList.push(`-p`);
        }
      }

      let jdbcString = t(
        "datasource.jdbc-cant-connect-to-socket-database-value-instance-host",
        [database.value.instance.host]
      );
      if (!isSocket) {
        jdbcString = `jdbc:mysql://${database.value.instance.host}`;
        if (database.value.instance.port) {
          jdbcString += `:${database.value.instance.port}`;
        }
        if (database.value.name) {
          jdbcString += `/${database.value.name}`;
        }
        const optionList = [];
        if (props.dataSource.username) {
          optionList.push(`user=${props.dataSource.username}`);
        }
        if (props.dataSource.password) {
          if (state.showPassword) {
            optionList.push(`password=${props.dataSource.password}`);
          } else {
            optionList.push(`password=******`);
          }
        }
        if (optionList.length > 0) {
          jdbcString += `&${optionList.join("&")}`;
        }
      }

      return [
        { name: "CLI", value: cliOptionList.join(" ") },
        { name: "JDBC", value: jdbcString },
      ];
    });

    const copyText = (connection: Connection) => {
      toClipboard(connection.value).then(() => {
        notificationStore.pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("datasource.connection-name-string-copied-to-clipboard", [
            connection.name,
          ]),
        });
      });
    };

    return {
      state,
      connectionStringList,
      copyText,
    };
  },
});
</script>
