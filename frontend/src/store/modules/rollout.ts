import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import dayjs from "dayjs";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch } from "vue";
import { rolloutServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { MaybeRef } from "@/types";
import { isValidRolloutName, unknownRollout } from "@/types";
import {
  GetRolloutRequestSchema,
  ListRolloutsRequestSchema,
  type Rollout,
  Task_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  getTsRangeFromSearchParams,
  getValueFromSearchParams,
  type SearchParams,
} from "@/utils";

export interface RolloutFind {
  project: string;
  taskType?: Task_Type[];
  query?: string;
  updatedTsAfter?: number;
  updatedTsBefore?: number;
}

export const buildRolloutFilter = (find: RolloutFind): string => {
  const filter: string[] = [];
  if (find.taskType && find.taskType.length > 0) {
    const types = find.taskType.map((t) => `"${Task_Type[t]}"`).join(", ");
    filter.push(`task_type in [${types}]`);
  }
  if (find.updatedTsAfter) {
    filter.push(
      `update_time >= "${dayjs(find.updatedTsAfter).utc().toISOString()}"`
    );
  }
  if (find.updatedTsBefore) {
    filter.push(
      `update_time <= "${dayjs(find.updatedTsBefore).utc().toISOString()}"`
    );
  }
  return filter.join(" && ");
};

export const buildRolloutFindBySearchParams = (
  params: SearchParams,
  defaultFind?: Partial<RolloutFind>
) => {
  const projectScope = getValueFromSearchParams(params, "project");
  const updatedTsRange = getTsRangeFromSearchParams(params, "updated");

  const filter: RolloutFind = {
    ...defaultFind,
    project: `projects/${projectScope || "-"}`,
    query: params.query,
    updatedTsAfter: updatedTsRange?.[0],
    updatedTsBefore: updatedTsRange?.[1],
  };
  return filter;
};

export type ListRolloutParams = {
  find: RolloutFind;
  pageSize?: number;
  pageToken?: string;
};

export const useRolloutStore = defineStore("rollout", () => {
  const rolloutMapByName = reactive(new Map<string, Rollout>());

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
      pageSize: pageSize,
      pageToken: pageToken || "",
      filter: buildRolloutFilter(find),
    });
    const resp = await rolloutServiceClientConnect.listRollouts(request);
    const rollouts = resp.rollouts;
    rollouts.forEach((rollout) => {
      rolloutMapByName.set(rollout.name, rollout);
    });
    return {
      rollouts: rollouts,
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
    rolloutMapByName.set(rollout.name, rollout);
    return rollout;
  };

  const getRolloutByName = (name: string) => {
    return rolloutMapByName.get(name) ?? unknownRollout();
  };

  return {
    rolloutList,
    listRollouts,
    fetchRolloutByName,
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
        try {
          await store.fetchRolloutByName(name);
        } catch {
          // ignore
        }
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
