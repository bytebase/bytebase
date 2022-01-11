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
    <template
      #body="{ rowData: database }: { rowData: (typeof databaseList)[number] }"
    >
      <BBTableCell :left-padding="4" class="w-16">
        <div class="flex flex-row items-center space-x-1 tooltip-wrapper">
          <span>{{ database.name }}</span>
          <div v-if="!showMiscColumn && database.syncStatus != 'OK'">
            <span class="tooltip">
              {{
                $t("database.last-sync-status-long", [
                  database.syncStatus,
                  humanizeTs(database.lastSuccessfulSyncTs),
                ])
              }}
            </span>
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
              <span class="tooltip whitespace-nowrap">
                {{ $t("database.version-control-enabled") }}
              </span>
              <heroicons-outline:collection
                class="w-4 h-4 text-control hover:text-control-hover"
              />
            </template>
          </div>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showEnvironmentColumn" class="w-16">{{
        environmentName(database.instance.environment)
      }}</BBTableCell>
      <BBTableCell v-if="showInstanceColumn" class="w-32">
        <div class="flex flex-row items-center space-x-1">
          <InstanceEngineIcon :instance="database.instance" />
          <span>{{ instanceName(database.instance) }}</span>
        </div>
      </BBTableCell>
      <BBTableCell v-if="showMiscColumn" class="w-4">{{
        database.syncStatus
      }}</BBTableCell>
      <BBTableCell v-if="showMiscColumn" class="w-16">{{
        humanizeTs(database.lastSuccessfulSyncTs)
      }}</BBTableCell>
      <BBTableCell v-if="showSqlEditorLink" class="w-16">
        <button class="btn-icon" @click.stop="gotoSqlEditor(database)">
          <heroicons-outline:terminal class="w-4 h-4" />
        </button>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { connectionSlug, databaseSlug } from "../utils";
import { Database } from "../types";
import { BBTableColumn } from "../bbkit/types";
import InstanceEngineIcon from "./InstanceEngineIcon.vue";
import { cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";

type Mode = "ALL" | "ALL_SHORT" | "INSTANCE" | "PROJECT" | "PROJECT_SHORT";

export default defineComponent({
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
    rowClickable: {
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
    const { t } = useI18n();

    const columnListMap = computed(() => {
      return new Map([
        [
          "ALL",
          [
            {
              title: t("common.name"),
            },
            {
              title: t("common.project"),
            },
            {
              title: t("common.environment"),
            },
            {
              title: t("common.instance"),
            },
            {
              title: t("db.sync-status"),
            },
            {
              title: t("db.last-successful-sync"),
            },
          ],
        ],
        [
          "ALL_SHORT",
          [
            {
              title: t("common.name"),
            },
            {
              title: t("common.project"),
            },
            {
              title: t("common.environment"),
            },
            {
              title: t("common.instance"),
            },
          ],
        ],
        [
          "INSTANCE",
          [
            {
              title: t("common.name"),
            },
            {
              title: t("common.project"),
            },
            {
              title: t("db.sync-status"),
            },
            {
              title: t("db.last-successful-sync"),
            },
          ],
        ],
        [
          "PROJECT",
          [
            {
              title: t("common.name"),
            },
            {
              title: t("common.environment"),
            },
            {
              title: t("common.instance"),
            },
            {
              title: t("db.sync-status"),
            },
            {
              title: t("db.last-successful-sync"),
            },
          ],
        ],
        [
          "PROJECT_SHORT",
          [
            {
              title: t("common.name"),
            },
            {
              title: t("common.environment"),
            },
            {
              title: t("common.instance"),
            },
          ],
        ],
      ]);
    });

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
      var list: BBTableColumn[] = columnListMap.value.get(props.mode)!;
      if (showSqlEditorLink.value) {
        // Use cloneDeep, otherwise it will alter the one in columnListMap
        list = cloneDeep(list);
        list.push({ title: t("sql-editor.self") });
      }
      return list;
    });

    const showSqlEditorLink = computed(() => {
      if (props.mode == "ALL_SHORT" || props.mode == "PROJECT_SHORT") {
        return false;
      }
      return true;
    });

    const gotoSqlEditor = (database: Database) => {
      router.push({
        name: "sql-editor.detail",
        params: {
          connectionSlug: connectionSlug(database),
        },
      });
    };

    const clickDatabase = function (section: number, row: number) {
      if (!props.rowClickable) return;

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
      showSqlEditorLink,
      gotoSqlEditor,
      clickDatabase,
    };
  },
});
</script>
