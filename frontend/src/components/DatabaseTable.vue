<template>
  <BBTable
    :column-list="columnList"
    :data-source="databaseList"
    :show-header="true"
    :left-bordered="bordered"
    :right-bordered="bordered"
    :top-bordered="bordered"
    :bottom-bordered="bordered"
    @click-row="clickDatabase"
  >
    <template #body="{ rowData: database }">
      <BBTableCell :left-padding="4" class="w-16">
        <div class="flex flex-row items-center space-x-1 tooltip-wrapper">
          <span>
            {{ database.name }}
          </span>
          <div v-if="!showMiscColumn && database.syncStatus != 'OK'">
            <span class="tooltip"
              >Last sync status {{ database.syncStatus }} at
              {{ humanizeTs(database.lastSuccessfulSyncTs) }}</span
            >
            <svg
              class="w-5 h-5 text-error"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
          </div>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showProjectColumn" class="w-16">
        <div class="flex flex-row space-x-2 items-center">
          <div>{{ projectName(database.project) }}</div>
          <div class="tooltip-wrapper">
            <svg
              v-if="database.project.workflowType == 'UI'"
              class="w-4 h-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            ></svg>
            <template v-else-if="database.project.workflowType == 'VCS'">
              <span class="tooltip">Version control enabled</span>
              <svg
                class="w-4 h-4 text-control hover:text-control-hover"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
                ></path>
              </svg>
            </template>
          </div>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showEnvironmentColumn" class="w-16">
        {{ environmentName(database.instance.environment) }}
      </BBTableCell>
      <BBTableCell v-if="showInstanceColumn" class="w-32">
        <div class="flex flex-row items-center space-x-1">
          <InstanceEngineIcon :instance="database.instance" />
          <span>{{ instanceName(database.instance) }}</span>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showMiscColumn" class="w-8">
        {{ database.syncStatus }}
      </BBTableCell>
      <BBTableCell v-if="showMiscColumn" class="w-16">
        {{ humanizeTs(database.lastSuccessfulSyncTs) }}
      </BBTableCell>
      <BBTableCell v-if="showConsoleLink" class="w-4">
        <button
          class="btn-icon"
          @click.stop="
            window.open(databaseConsoleLink(database.name), '_blank')
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
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { consoleLink, databaseSlug } from "../utils";
import { Database } from "../types";
import { BBTableColumn } from "../bbkit/types";
import InstanceEngineIcon from "./InstanceEngineIcon.vue";
import { cloneDeep, isEmpty } from "lodash";

type Mode = "ALL" | "ALL_SHORT" | "INSTANCE" | "PROJECT" | "PROJECT_SHORT";

const columnListMap: Map<Mode, BBTableColumn[]> = new Map([
  [
    "ALL",
    [
      {
        title: "Name",
      },
      {
        title: "Project",
      },
      {
        title: "Environment",
      },
      {
        title: "Instance",
      },
      {
        title: "Sync status",
      },
      {
        title: "Last successful sync",
      },
    ],
  ],
  [
    "ALL_SHORT",
    [
      {
        title: "Name",
      },
      {
        title: "Project",
      },
      {
        title: "Environment",
      },
      {
        title: "Instance",
      },
    ],
  ],
  [
    "INSTANCE",
    [
      {
        title: "Name",
      },
      {
        title: "Project",
      },
      {
        title: "Sync status",
      },
      {
        title: "Last successful sync",
      },
    ],
  ],
  [
    "PROJECT",
    [
      {
        title: "Name",
      },
      {
        title: "Environment",
      },
      {
        title: "Instance",
      },
      {
        title: "Sync status",
      },
      {
        title: "Last successful sync",
      },
    ],
  ],
  [
    "PROJECT_SHORT",
    [
      {
        title: "Name",
      },
      {
        title: "Environment",
      },
      {
        title: "Instance",
      },
    ],
  ],
]);

export default {
  name: "DatabaseTable",
  components: { InstanceEngineIcon },
  props: {
    bordered: {
      default: true,
      type: Boolean,
    },
    mode: {
      default: "ALL",
      type: String as PropType<Mode>,
    },
    singleInstance: {
      default: true,
      type: Boolean,
    },
    customClick: {
      default: false,
      type: Boolean,
    },
    databaseList: {
      required: true,
      type: Object as PropType<Database[]>,
    },
  },
  emits: ["select-database"],
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    // const currentUser = computed(() => store.getters["auth/currentUser"]());

    const showInstanceColumn = computed(() => {
      return props.mode != "INSTANCE";
    });

    const showProjectColumn = computed(() => {
      return props.mode != "PROJECT" && props.mode != "PROJECT_SHORT";
    });

    const showEnvironmentColumn = computed(() => {
      return props.mode != "INSTANCE";
    });

    const showMiscColumn = computed(() => {
      return props.mode != "ALL_SHORT" && props.mode != "PROJECT_SHORT";
    });

    const columnList = computed(() => {
      var list: BBTableColumn[] = columnListMap.get(props.mode)!;
      if (showConsoleLink.value) {
        // Use cloneDeep, otherwise it will alter the one in columnListMap
        list = cloneDeep(list);
        list.push({ title: "SQL console" });
      }
      return list;
    });

    const showConsoleLink = computed(() => {
      if (props.mode == "ALL_SHORT" || props.mode == "PROJECT_SHORT") {
        return false;
      }

      const consoleURL =
        store.getters["setting/settingByName"]("bb.console.url").value;
      return !isEmpty(consoleURL);
    });

    const databaseConsoleLink = (databaseName: string) => {
      const consoleURL =
        store.getters["setting/settingByName"]("bb.console.url").value;
      if (!isEmpty(consoleURL)) {
        return consoleLink(consoleURL, databaseName);
      }
      return "";
    };

    const clickDatabase = function (section: number, row: number) {
      const database = props.databaseList[row];
      if (props.customClick) {
        emit("select-database", database);
      } else {
        router.push(`/db/${databaseSlug(database)}`);
      }
    };

    return {
      showInstanceColumn,
      showProjectColumn,
      showEnvironmentColumn,
      showMiscColumn,
      columnList,
      showConsoleLink,
      databaseConsoleLink,
      clickDatabase,
    };
  },
};
</script>
