<template>
  <BBTable
    :columnList="columnList"
    :sectionDataSource="taskSectionList"
    :showHeader="true"
    :leftBordered="leftBordered"
    :rightBordered="rightBordered"
    :topBordered="topBordered"
    :bottomBordered="bottomBordered"
    @click-row="clickTask"
  >
    <template v-slot:header>
      <BBTableHeaderCell class="w-4 table-cell" :title="columnList[0].title" />
      <template v-if="mode == 'ALL'">
        <BBTableHeaderCell
          class="w-12 table-cell"
          :title="columnList[1].title"
        />
        <BBTableHeaderCell
          class="w-48 table-cell"
          :title="columnList[2].title"
        />
        <BBTableHeaderCell
          class="w-12 table-cell"
          :title="columnList[3].title"
        />
        <BBTableHeaderCell
          class="w-12 table-cell"
          :title="columnList[4].title"
        />
        <BBTableHeaderCell
          class="w-24 hidden sm:table-cell"
          :title="columnList[5].title"
        />
        <BBTableHeaderCell
          class="w-24 hidden md:table-cell"
          :title="columnList[6].title"
        />
        <BBTableHeaderCell
          class="w-36 hidden sm:table-cell"
          :title="columnList[7].title"
        />
      </template>
      <template v-else-if="mode == 'PROJECT'">
        <BBTableHeaderCell
          class="w-48 table-cell"
          :title="columnList[1].title"
        />
        <BBTableHeaderCell
          class="w-12 table-cell"
          :title="columnList[2].title"
        />
        <BBTableHeaderCell
          class="w-12 table-cell"
          :title="columnList[3].title"
        />
        <BBTableHeaderCell
          class="w-24 hidden sm:table-cell"
          :title="columnList[4].title"
        />
        <BBTableHeaderCell
          class="w-24 hidden md:table-cell"
          :title="columnList[5].title"
        />
        <BBTableHeaderCell
          class="w-36 hidden sm:table-cell"
          :title="columnList[6].title"
        />
      </template>
    </template>
    <template v-slot:body="{ rowData: task }">
      <BBTableCell :leftPadding="4" class="table-cell">
        <TaskStatusIcon
          :taskStatus="task.status"
          :stageStatus="activeStage(task).status"
        />
      </BBTableCell>
      <BBTableCell v-if="mode == 'ALL'" class="table-cell text-gray-500">
        <span class="">{{ task.project.key }}</span>
      </BBTableCell>
      <BBTableCell class="truncate">
        {{ task.name }}
      </BBTableCell>
      <BBTableCell class="table-cell">
        {{ activeEnvironmentName(task) }}
      </BBTableCell>
      <BBTableCell class="table-cell">
        {{ activeDatabaseName(task) }}
      </BBTableCell>
      <BBTableCell class="hidden sm:table-cell">
        <BBStepBar :stepList="stageList(task)" />
      </BBTableCell>
      <BBTableCell class="hidden md:table-cell">
        {{ humanizeTs(task.lastUpdatedTs) }}
      </BBTableCell>
      <BBTableCell class="hidden sm:table-cell">
        <div class="flex flex-row items-center">
          <BBAvatar
            :size="'small'"
            :username="task.assignee ? task.assignee.name : 'Unassigned'"
          />
          <span class="ml-2">{{
            task.assignee ? task.assignee.name : "Unassigned"
          }}</span>
        </div>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import {
  BBTableColumn,
  BBTableSectionDataSource,
  BBStep,
  BBStepStatus,
} from "../bbkit/types";
import TaskStatusIcon from "../components/TaskStatusIcon.vue";
import {
  taskSlug,
  activeEnvironmentId,
  activeDatabaseId,
  activeStage,
} from "../utils";
import { Task } from "../types";

type Mode = "ALL" | "PROJECT";

const columnListMap: Map<Mode, BBTableColumn[]> = new Map([
  [
    "ALL",
    [
      {
        title: "Status",
      },
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
        title: "Database",
      },
      {
        title: "Progress",
      },
      {
        title: "Updated",
      },
      {
        title: "Assignee",
      },
    ],
  ],
  [
    "PROJECT",
    [
      {
        title: "Status",
      },
      {
        title: "Name",
      },
      {
        title: "Environment",
      },
      {
        title: "Database",
      },
      {
        title: "Progress",
      },
      {
        title: "Updated",
      },
      {
        title: "Assignee",
      },
    ],
  ],
]);

interface LocalState {
  dataSource: Object[];
}

export default {
  name: "TaskTable",
  components: { TaskStatusIcon },
  props: {
    taskSectionList: {
      required: true,
      type: Object as PropType<BBTableSectionDataSource<Task>[]>,
    },
    mode: {
      default: "ALL",
      type: String as PropType<Mode>,
    },
    leftBordered: {
      default: true,
      type: Boolean,
    },
    rightBordered: {
      default: true,
      type: Boolean,
    },
    topBordered: {
      default: true,
      type: Boolean,
    },
    bottomBordered: {
      default: true,
      type: Boolean,
    },
  },
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      dataSource: [],
    });

    const activeEnvironmentName = function (task: Task) {
      const id = activeEnvironmentId(task);
      if (id) {
        return store.getters["environment/environmentById"](id).name;
      }
      return "";
    };

    const activeDatabaseName = function (task: Task) {
      const id = activeDatabaseId(task);
      if (id) {
        return store.getters["database/databaseById"](id).name;
      }
      return "";
    };

    const router = useRouter();

    const stageList = function (task: Task): BBStep[] {
      return task.stageList.map((stage) => {
        let stepStatus: BBStepStatus = "PENDING";
        switch (stage.status) {
          case "PENDING":
            if (activeStage(task).id === stage.id) {
              stepStatus = "PENDING_ACTIVE";
            } else {
              stepStatus = "PENDING";
            }
            break;
          case "RUNNING":
            stepStatus = "RUNNING";
            break;
          case "DONE":
            stepStatus = "DONE";
            break;
          case "FAILED":
            stepStatus = "FAILED";
            break;
          case "SKIPPED":
            stepStatus = "SKIPPED";
            break;
        }
        return {
          title: stage.name,
          status: stepStatus,
          link: (): string => {
            return `/task/${task.id}`;
          },
        };
      });
    };

    const clickTask = function (section: number, row: number) {
      const task = props.taskSectionList[section].list[row];
      router.push(`/task/${taskSlug(task.name, task.id)}`);
    };

    return {
      state,
      columnList: columnListMap.get(props.mode),
      activeEnvironmentName,
      activeDatabaseName,
      stageList,
      activeStage,
      clickTask,
    };
  },
};
</script>
