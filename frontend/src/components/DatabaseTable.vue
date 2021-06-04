<template>
  <BBTable
    :columnList="columnList"
    :dataSource="databaseList"
    :showHeader="true"
    :leftBordered="bordered"
    :rightBordered="bordered"
    @click-row="clickDatabase"
  >
    <template v-slot:body="{ rowData: database }">
      <BBTableCell v-if="showProjectColumn" :leftPadding="4" class="w-16">
        {{ projectName(database.project) }}
      </BBTableCell>
      <BBTableCell
        :leftPadding="showProjectColumn ? undefined : 4"
        class="w-16"
      >
        {{ database.name }}
      </BBTableCell>
      <BBTableCell v-if="showEnvironmentColumn" class="w-12">
        {{ environmentName(database.instance.environment) }}
      </BBTableCell>
      <BBTableCell v-if="showInstanceColumn" class="w-24">
        {{ instanceName(database.instance) }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ database.characterSet }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ database.collation }}
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ database.syncStatus }}
      </BBTableCell>
      <BBTableCell class="w-16">
        {{ humanizeTs(database.lastSuccessfulSyncTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { databaseSlug, isDBAOrOwner } from "../utils";
import { Database } from "../types";
import { BBTableColumn } from "../bbkit/types";

type Mode = "ALL" | "INSTANCE" | "PROJECT";

const columnListMap: Map<Mode | "ALL_HIDE_INSTANCE", BBTableColumn[]> = new Map(
  [
    [
      "ALL",
      [
        {
          title: "Project",
        },
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
          title: "Character set",
        },
        {
          title: "Collation",
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
      "ALL_HIDE_INSTANCE",
      [
        {
          title: "Project",
        },
        {
          title: "Name",
        },
        {
          title: "Environment",
        },
        {
          title: "Character set",
        },
        {
          title: "Collation",
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
      "INSTANCE",
      [
        {
          title: "Project",
        },
        {
          title: "Name",
        },
        {
          title: "Character set",
        },
        {
          title: "Collation",
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
          title: "Character set",
        },
        {
          title: "Collation",
        },
        {
          title: "Sync status",
        },
        {
          title: "Last successful sync",
        },
      ],
    ],
  ]
);

export default {
  name: "DatabaseTable",
  components: {},
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
    databaseList: {
      required: true,
      type: Object as PropType<Database[]>,
    },
  },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const showProjectColumn = computed(() => {
      return props.mode != "PROJECT";
    });

    const showEnvironmentColumn = computed(() => {
      return props.mode != "INSTANCE";
    });

    const showInstanceColumn = computed(() => {
      return props.mode == "ALL" && isDBAOrOwner(currentUser.value.role);
    });

    const columnList = computed(() => {
      if (props.mode == "ALL" && !showInstanceColumn.value) {
        return columnListMap.get("ALL_HIDE_INSTANCE");
      }
      return columnListMap.get(props.mode);
    });

    const clickDatabase = function (section: number, row: number) {
      const database = props.databaseList[row];
      router.push(`/db/${databaseSlug(database)}`);
    };

    return {
      showProjectColumn,
      showEnvironmentColumn,
      showInstanceColumn,
      columnList,
      clickDatabase,
    };
  },
};
</script>
