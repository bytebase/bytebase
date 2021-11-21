<template>
  <!-- Description list -->
  <form class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2">
    <dl class="sm:col-span-2">
      <dt class="flex items-center space-x-1">
        <div class="text-sm font-medium text-control-light">
          Connection string
        </div>
        <button
          class="btn-icon"
          @click.prevent="state.showPassword = !state.showPassword"
        >
          <svg
            v-if="state.showPassword"
            class="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
            ></path>
          </svg>
          <svg
            v-else
            class="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
            ></path>
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
            ></path>
          </svg>
        </button>
      </dt>
      <dd class="mt-2 text-sm text-main">
        <div class="space-y-4">
          <div
            class="flex"
            v-for="(connection, index) in connectionStringList"
            :key="index"
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
      </dd>
    </dl>

    <dl class="sm:col-span-1">
      <dt class="text-sm font-medium text-control-light">Username</dt>
      <dd class="mt-1 text-sm text-main">
        <input
          v-if="editing"
          required
          id="username"
          type="text"
          class="textfield"
          v-model="dataSource.username"
        />
        <div v-else class="mt-2.5 mb-3">
          {{ dataSource.username }}
        </div>
      </dd>
    </dl>

    <dl class="sm:col-span-1">
      <dt class="text-sm font-medium text-control-light">Password</dt>
      <dd class="mt-1 text-sm text-main">
        <input
          v-if="editing"
          required
          autocomplete="off"
          id="password"
          :type="state.showPassword ? 'text' : 'password'"
          class="textfield"
          v-model="dataSource.password"
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
      <dt class="text-sm font-medium text-control-light">Updated</dt>
      <dd class="mt-1 text-sm text-main">
        {{ humanizeTs(dataSource.updatedTs) }}
      </dd>
    </dl>

    <dl class="sm:col-span-1">
      <dt class="text-sm font-medium text-control-light">Created</dt>
      <dd class="mt-1 text-sm text-main">
        {{ humanizeTs(dataSource.createdTs) }}
      </dd>
    </dl>
  </form>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import { DataSource } from "../types";

type Connection = {
  name: string;
  value: string;
};

interface LocalState {
  showPassword: boolean;
}

export default {
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
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      showPassword: false,
    });

    const database = computed(() => {
      return store.getters["database/databaseByID"](
        props.dataSource.database.id,
        props.dataSource.instance.id
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

      let jdbcString = `JDBC can't connect to socket ${database.value.instance.host} `;
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
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: `${connection.name} string copied to clipboard.`,
        });
      });
    };

    return {
      state,
      connectionStringList,
      copyText,
    };
  },
};
</script>
