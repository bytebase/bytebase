import axios from "axios";
import {
  ResourceObject,
  StepState,
  Step,
  Task,
  unknown,
  IssueId,
  TaskId,
  PipelineId,
  Pipeline,
  StepStatusPatch,
  StepId,
} from "../../types";

const state: () => StepState = () => ({});

function convertPartial(
  step: ResourceObject,
  rootGetters: any
): Omit<Step, "pipeline" | "task"> {
  const creator = rootGetters["principal/principalById"](
    step.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    step.attributes.updaterId
  );

  return {
    ...(step.attributes as Omit<
      Step,
      "id" | "creator" | "updater" | "pipeline" | "task"
    >),
    id: step.id,
    creator,
    updater,
  };
}

const getters = {
  convertPartial: (
    state: StepState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (step: ResourceObject): Step => {
    // It's only called when pipeline/task tries to convert itself, so we don't have a issue yet.
    const pipelineId = step.attributes.pipelineId as PipelineId;
    let pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
    pipeline.id = pipelineId;

    const taskId = step.attributes.taskId as TaskId;
    let task: Task = unknown("TASK") as Task;
    task.id = taskId;

    return {
      ...convertPartial(step, rootGetters),
      pipeline,
      task,
    };
  },
};

const actions = {
  async updateStatus(
    { dispatch }: any,
    {
      issueId,
      pipelineId,
      stepId,
      stepStatusPatch,
    }: {
      issueId: IssueId;
      pipelineId: PipelineId;
      stepId: StepId;
      stepStatusPatch: StepStatusPatch;
    }
  ) {
    // TODO: Returns the updated pipeline and update the issue.
    const data = (
      await axios.patch(`/api/pipeline/${pipelineId}/step/${stepId}/status`, {
        data: {
          type: "stepstatuspatch",
          attributes: stepStatusPatch,
        },
      })
    ).data;

    dispatch("issue/fetchIssueById", issueId, { root: true });
  },
};

const mutations = {};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
