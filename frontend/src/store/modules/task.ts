import axios from "axios";
import {
  UserId,
  TaskId,
  Task,
  TaskNew,
  TaskPatch,
  TaskState,
  ResourceObject,
  Principal,
  unknown,
  Project,
  ResourceIdentifier,
  ProjectId,
  Stage,
} from "../../types";

function convert(
  task: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Task {
  const creator = rootGetters["principal/principalById"](
    task.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    task.attributes.updaterId
  );

  let assignee = undefined;
  if (task.attributes.assigneeId) {
    assignee = rootGetters["principal/principalById"](
      task.attributes.assigneeId
    );
  }

  const stageList: Stage[] = (task.attributes.stageList as any[])!.map(
    (item) => {
      return {
        ...item,
        database: rootGetters["database/databaseById"](item.databaseId),
      };
    }
  );

  const subscriberList = (task.attributes.subscriberIdList as Principal[]).map(
    (principalId) => {
      return rootGetters["principal/principalById"](principalId);
    }
  );

  const projectId = (task.relationships!.project.data as ResourceIdentifier).id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = projectId;

  for (const item of includedList || []) {
    if (item.type == "project" && item.id == projectId) {
      project = rootGetters["project/convert"](item);
    }
  }

  return {
    ...(task.attributes as Omit<
      Task,
      | "id"
      | "project"
      | "creator"
      | "updater"
      | "assignee"
      | "stageList"
      | "subscriberList"
    >),
    id: task.id,
    project,
    creator,
    updater,
    assignee,
    stageList,
    subscriberList,
  };
}

const state: () => TaskState = () => ({
  taskListByUser: new Map(),
  taskById: new Map(),
});

const getters = {
  taskListByUser: (state: TaskState) => (userId: UserId) => {
    return state.taskListByUser.get(userId) || [];
  },

  taskById: (state: TaskState) => (taskId: TaskId): Task => {
    return state.taskById.get(taskId) || (unknown("TASK") as Task);
  },
};

const actions = {
  async fetchTaskListForUser({ commit, rootGetters }: any, userId: UserId) {
    const data = (await axios.get(`/api/task?user=${userId}&include=project`))
      .data;
    const taskList = data.data.map((task: ResourceObject) => {
      return convert(task, data.included, rootGetters);
    });

    commit("setTaskListForUser", { userId, taskList });
    return taskList;
  },

  async fetchTaskListForProject({ rootGetters }: any, projectId: ProjectId) {
    const data = (
      await axios.get(`/api/task?project=${projectId}&include=project`)
    ).data;
    const taskList = data.data.map((task: ResourceObject) => {
      return convert(task, data.included, rootGetters);
    });

    // The caller consumes directly, so we don't store it.
    return taskList;
  },

  async fetchTaskById({ commit, rootGetters }: any, taskId: TaskId) {
    const data = (await axios.get(`/api/task/${taskId}?include=project`)).data;
    const task = convert(data.data, data.included, rootGetters);
    commit("setTaskById", {
      taskId,
      task,
    });
    return task;
  },

  async createTask({ commit, rootGetters }: any, newTask: TaskNew) {
    const data = (
      await axios.post(`/api/task?include=project`, {
        data: {
          type: "tasknew",
          attributes: newTask,
        },
      })
    ).data;
    const createdTask = convert(data.data, data.included, rootGetters);

    commit("setTaskById", {
      taskId: createdTask.id,
      task: createdTask,
    });

    return createdTask;
  },

  async patchTask(
    { commit, dispatch, rootGetters }: any,
    {
      taskId,
      taskPatch,
    }: {
      taskId: TaskId;
      taskPatch: TaskPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/task/${taskId}?include=project`, {
        data: {
          type: "taskpatch",
          attributes: taskPatch,
        },
      })
    ).data;
    const updatedTask = convert(data.data, data.included, rootGetters);

    commit("setTaskById", {
      taskId: taskId,
      task: updatedTask,
    });

    dispatch("activity/fetchActivityListForTask", taskId, { root: true });

    return updatedTask;
  },
};

const mutations = {
  setTaskListForUser(
    state: TaskState,
    {
      userId,
      taskList,
    }: {
      userId: UserId;
      taskList: Task[];
    }
  ) {
    state.taskListByUser.set(userId, taskList);
  },

  setTaskById(
    state: TaskState,
    {
      taskId,
      task,
    }: {
      taskId: TaskId;
      task: Task;
    }
  ) {
    state.taskById.set(taskId, task);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
