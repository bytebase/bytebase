import {
  ResourceIdentifier,
  ResourceObject,
  Pipeline,
  PipelineState,
  Stage,
} from "@/types";
import { defineStore } from "pinia";
import { getPrincipalFromIncludedList } from "./principal";
import { useStageStore } from "./stage";

function convert(
  pipeline: ResourceObject,
  includedList: ResourceObject[]
): Pipeline {
  const stageList: Stage[] = [];
  const stageIdList = pipeline.relationships!.stage
    .data as ResourceIdentifier[];
  const stageStore = useStageStore();
  // Needs to iterate through stageIdList to maintain the order
  for (const idItem of stageIdList) {
    for (const item of includedList || []) {
      if (item.type == "stage") {
        if (idItem.id == item.id) {
          const stage: Stage = stageStore.convertPartial(item, includedList);
          stageList.push(stage);
        }
      }
    }
  }

  const result: Pipeline = {
    ...(pipeline.attributes as Omit<
      Pipeline,
      "id" | "stageList" | "creator" | "updater"
    >),
    id: parseInt(pipeline.id),
    creator: getPrincipalFromIncludedList(
      pipeline.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      pipeline.relationships!.updater.data,
      includedList
    ),
    stageList,
  };

  // Now we have a complete issue, we assign it back to stage and task
  for (const stage of result.stageList) {
    stage.pipeline = result;
    for (const task of stage.taskList) {
      task.pipeline = result;
      task.stage = stage;
    }
  }

  return result;
}
export const usePipelineStore = defineStore("pipeline", {
  state: (): PipelineState => ({}),
  actions: {
    convert(
      pipeline: ResourceObject,
      includedList: ResourceObject[]
    ): Pipeline {
      return convert(pipeline, includedList);
    },
  },
});
