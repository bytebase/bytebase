import axios from "axios";
import {
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageId,
  StageState,
  Task,
  Issue,
  IssueId,
  unknown,
  PipelineId,
  Pipeline,
  Database,
  empty,
  Environment,
  Principal,
} from "../../types";

const state: () => StageState = () => ({});

function convertPartial(
  stage: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Stage, "pipeline"> {
  const creatorId = (stage.relationships!.creator.data as ResourceIdentifier)
    .id;
  let creator: Principal = unknown("PRINCIPAL") as Principal;
  creator.id = creatorId;

  const updaterId = (stage.relationships!.updater.data as ResourceIdentifier)
    .id;
  let updater: Principal = unknown("PRINCIPAL") as Principal;
  updater.id = updaterId;

  // We should always have a valid environment
  const environment: Environment = rootGetters["environment/environmentById"](
    (stage.relationships!.environment.data as ResourceIdentifier).id
  );

  const taskList: Task[] = [];
  for (const item of includedList || []) {
    if (
      item.type == "principal" &&
      (stage.relationships!.creator.data as ResourceIdentifier).id == item.id
    ) {
      creator = rootGetters["principal/convert"](item);
    }

    if (
      item.type == "principal" &&
      (stage.relationships!.updater.data as ResourceIdentifier).id == item.id
    ) {
      updater = rootGetters["principal/convert"](item);
    }

    if (item.type == "task") {
      const taskIdList = stage.relationships!.task.data as ResourceIdentifier[];
      for (const idItem of taskIdList) {
        if (idItem.id == item.id) {
          const task: Task = rootGetters["task/convertPartial"](
            item,
            includedList
          );
          taskList.push(task);
        }
      }
    }
  }

  const result: Omit<Stage, "pipeline"> = {
    ...(stage.attributes as Omit<
      Stage,
      "id" | "creator" | "updater" | "database" | "taskList"
    >),
    id: stage.id,
    creator,
    updater,
    environment,
    taskList,
  };

  return result;
}

const getters = {
  convertPartial:
    (state: StageState, getters: any, rootState: any, rootGetters: any) =>
    (stage: ResourceObject, includedList: ResourceObject[]): Stage => {
      // It's only called when pipeline tries to convert itself, so we don't have a issue yet.
      const pipelineId = stage.attributes.pipelineId as PipelineId;
      let pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
      pipeline.id = pipelineId;

      const result: Stage = {
        ...convertPartial(stage, includedList, rootGetters),
        pipeline,
      };

      for (const task of result.taskList) {
        task.stage = result;
        task.pipeline = pipeline;
      }

      return result;
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
