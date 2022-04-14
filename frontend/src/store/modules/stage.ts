import { defineStore } from "pinia";
import {
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageState,
  Task,
  unknown,
  PipelineId,
  Pipeline,
  Environment,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";
import { useEnvironmentStore } from "./environment";
import { useTaskStore } from "./task";

function convertPartial(
  stage: ResourceObject,
  includedList: ResourceObject[]
): Omit<Stage, "pipeline"> {
  let environment = unknown("ENVIRONMENT") as Environment;
  if (stage.relationships?.environment.data) {
    const environmentId = (
      stage.relationships.environment.data as ResourceIdentifier
    ).id;
    environment.id = parseInt(environmentId, 10);
  }

  const taskList: Task[] = [];
  const taskIdList = stage.relationships!.task.data as ResourceIdentifier[];
  const taskStore = useTaskStore();
  // Needs to iterate through taskIdList to maintain the order
  for (const idItem of taskIdList) {
    for (const item of includedList || []) {
      if (item.type == "task") {
        if (idItem.id == item.id) {
          const task = taskStore.convertPartial(item, includedList);
          taskList.push(task);
        }
      }
    }
  }

  const environmentStore = useEnvironmentStore();
  for (const item of includedList || []) {
    if (
      item.type == "environment" &&
      (stage.relationships!.environment.data as ResourceIdentifier).id ==
        item.id
    ) {
      environment = environmentStore.convert(item, includedList);
    }
  }

  const result: Omit<Stage, "pipeline"> = {
    ...(stage.attributes as Omit<
      Stage,
      "id" | "database" | "taskList" | "creator" | "updater"
    >),
    id: parseInt(stage.id),
    creator: getPrincipalFromIncludedList(
      stage.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      stage.relationships!.updater.data,
      includedList
    ),
    environment,
    taskList,
  };

  return result;
}

export const useStageStore = defineStore("stage", {
  state: (): StageState => ({}),
  actions: {
    convertPartial(
      stage: ResourceObject,
      includedList: ResourceObject[]
    ): Stage {
      // It's only called when pipeline tries to convert itself, so we don't have a issue yet.
      const pipelineId = stage.attributes.pipelineId as PipelineId;
      const pipeline: Pipeline = unknown("PIPELINE") as Pipeline;
      pipeline.id = pipelineId;

      const result: Stage = {
        ...convertPartial(stage, includedList),
        pipeline,
      };

      for (const task of result.taskList) {
        task.stage = result;
        task.pipeline = pipeline;
      }

      return result;
    },
  },
});
