import { head } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch } from "vue";
import { rolloutServiceClient } from "@/grpcweb";
import type { MaybeRef, Pagination, ComposedRollout } from "@/types";
import { isValidRolloutName, unknownRollout, unknownUser } from "@/types";
import type { Rollout } from "@/types/proto/v1/rollout_service";
import { extractUserResourceName } from "@/utils";
import { DEFAULT_PAGE_SIZE } from "./common";
import { useUserStore } from "./user";
import { useProjectV1Store, batchGetOrFetchProjects } from "./v1";
import { getProjectNameRolloutId, projectNamePrefix } from "./v1/common";

export const useRolloutStore = defineStore("rollout", () => {
  const rolloutMapByName = reactive(new Map<string, ComposedRollout>());

  const rolloutList = computed(() => {
    return Array.from(rolloutMapByName.values());
  });

  const fetchRolloutsByProject = async (
    project: string,
    pagination?: Pagination
  ) => {
    const resp = await rolloutServiceClient.listRollouts({
      parent: project,
      pageSize: pagination?.pageSize || DEFAULT_PAGE_SIZE,
      pageToken: pagination?.pageToken,
    });
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
    const rollout = await rolloutServiceClient.getRollout({ name }, { silent });
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
    fetchRolloutsByProject,
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
  const composedRolloutList = rolloutList.map((rollout) => {
    const composed = rollout as ComposedRollout;
    composed.project = `${projectNamePrefix}${head(getProjectNameRolloutId(rollout.name))}`;
    composed.creatorEntity =
      userStore.getUserByEmail(extractUserResourceName(composed.creator)) ??
      unknownUser();
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
