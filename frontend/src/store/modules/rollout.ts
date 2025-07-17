import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { head } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch } from "vue";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import type { MaybeRef, ComposedRollout } from "@/types";
import { isValidRolloutName, unknownRollout, unknownUser } from "@/types";
import {
  Task_Type,
  type Rollout,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  GetRolloutRequestSchema,
  ListRolloutsRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { DEFAULT_PAGE_SIZE } from "./common";
import { useUserStore } from "./user";
import { useProjectV1Store, batchGetOrFetchProjects } from "./v1";
import { getProjectNameRolloutId, projectNamePrefix } from "./v1/common";

export interface RolloutFind {
  project: string;
  taskType?: Task_Type | Task_Type[];
}

export const buildRolloutFilter = (find: RolloutFind): string => {
  const filter: string[] = [];
  if (find.taskType) {
    if (Array.isArray(find.taskType)) {
      const types = find.taskType.map((t) => `"${Task_Type[t]}"`).join(", ");
      filter.push(`task_type in [${types}]`);
    } else {
      filter.push(`task_type == "${Task_Type[find.taskType]}"`);
    }
  }
  return filter.join(" && ");
};

export type ListRolloutParams = {
  find: RolloutFind;
  pageSize?: number;
  pageToken?: string;
};

export const useRolloutStore = defineStore("rollout", () => {
  const rolloutMapByName = reactive(new Map<string, ComposedRollout>());

  const rolloutList = computed(() => {
    return Array.from(rolloutMapByName.values());
  });

  const listRollouts = async ({
    find,
    pageSize,
    pageToken,
  }: ListRolloutParams) => {
    const request = create(ListRolloutsRequestSchema, {
      parent: find.project,
      pageSize: pageSize || DEFAULT_PAGE_SIZE,
      pageToken: pageToken || "",
      filter: buildRolloutFilter(find),
    });
    const resp = await rolloutServiceClientConnect.listRollouts(request);
    const composedRolloutList = await batchComposeRollout(resp.rollouts);
    composedRolloutList.forEach((rollout) => {
      rolloutMapByName.set(rollout.name, rollout);
    });
    return {
      rollouts: composedRolloutList,
      nextPageToken: resp.nextPageToken,
    };
  };

  const fetchRolloutByName = async (name: string, silent = false) => {
    const request = create(GetRolloutRequestSchema, {
      name,
    });
    const rollout = await rolloutServiceClientConnect.getRollout(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    const [composedRollout] = await batchComposeRollout([rollout]);
    rolloutMapByName.set(composedRollout.name, composedRollout);
    return composedRollout;
  };

  const getRolloutsByProject = (project: string) => {
    return rolloutList.value.filter((rollout) => rollout.project === project);
  };

  const getRolloutByName = (name: string) => {
    return rolloutMapByName.get(name) ?? unknownRollout();
  };

  return {
    rolloutList,
    listRollouts,
    fetchRolloutByName,
    getRolloutsByProject,
    getRolloutByName,
  };
});

export const useRolloutByName = (name: MaybeRef<string>) => {
  const store = useRolloutStore();
  const ready = ref(true);
  watch(
    () => unref(name),
    async (name) => {
      if (!isValidRolloutName(name)) {
        return;
      }

      const cached = store.getRolloutByName(name);
      if (!isValidRolloutName(cached.name)) {
        ready.value = false;
        await store.fetchRolloutByName(name);
        ready.value = true;
      }
    },
    { immediate: true }
  );
  const rollout = computed(() => store.getRolloutByName(unref(name)));

  return {
    rollout,
    ready,
  };
};

export const batchComposeRollout = async (rolloutList: Rollout[]) => {
  const userStore = useUserStore();
  await userStore.batchGetUsers(rolloutList.map((rollout) => rollout.creator));

  const composedRolloutList = rolloutList.map((rollout) => {
    const composed = rollout as ComposedRollout;
    composed.project = `${projectNamePrefix}${head(getProjectNameRolloutId(rollout.name))}`;
    composed.creatorEntity =
      userStore.getUserByIdentifier(composed.creator) ?? unknownUser();
    return composed;
  });
  await batchGetOrFetchProjects(
    composedRolloutList.map((rollout) => rollout.project)
  );

  const projectV1Store = useProjectV1Store();
  return composedRolloutList.map((rollout) => {
    rollout.projectEntity = projectV1Store.getProjectByName(rollout.project);
    return rollout;
  });
};
