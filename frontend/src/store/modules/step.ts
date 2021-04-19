import {
  ResourceIdentifier,
  ResourceObject,
  StepState,
  Step,
  Issue,
  Task,
  unknown,
  IssueId,
  TaskId,
} from "../../types";

const state: () => StepState = () => ({});

function convertPartial(
  step: ResourceObject,
  rootGetters: any
): Omit<Step, "issue" | "task"> {
  const creator = rootGetters["principal/principalById"](
    step.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    step.attributes.updaterId
  );

  return {
    ...(step.attributes as Omit<
      Step,
      "id" | "creator" | "updater" | "issue" | "task"
    >),
    id: step.id,
    creator,
    updater,
  };
}

function convert(step: ResourceObject, rootGetters: any): Step {
  const issue: Issue = rootGetters["issue/issueById"](step.attributes.issueId);
  const task: Task = rootGetters["task/taskById"](step.attributes.taskId);
  return {
    ...convertPartial(step, rootGetters),
    issue,
    task,
  };
}

const getters = {
  convertPartial: (
    state: StepState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (step: ResourceObject): Step => {
    // It's only called when issue/task tries to convert themselves, so we don't have a issue/task yet.
    const issueId = step.attributes.issueId as IssueId;
    let issue: Issue = unknown("ISSUE") as Issue;
    issue.id = issueId;

    const taskId = step.attributes.taskId as TaskId;
    let task: Task = unknown("TASK") as Task;
    task.id = taskId;

    return {
      ...convertPartial(step, rootGetters),
      issue,
      task,
    };
  },
};

const actions = {};

const mutations = {};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
