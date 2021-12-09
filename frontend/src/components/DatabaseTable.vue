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
            <heroicons-outline:exclamation-circle class="w-5 h-5 text-error" />
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
              <heroicons-outline:collection
                class="w-4 h-4 text-control hover:text-control-hover"
              />
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
          <heroicons-outline:external-link class="w-4 h-4" />
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

      const consoleUrl =
        store.getters["setting/settingByName"]("bb.console.url").value;
      return !isEmpty(consoleUrl);
    });

    const databaseConsoleLink = (databaseName: string) => {
      const consoleUrl =
        store.getters["setting/settingByName"]("bb.console.url").value;
      if (!isEmpty(consoleUrl)) {
        return consoleLink(consoleUrl, databaseName);
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
