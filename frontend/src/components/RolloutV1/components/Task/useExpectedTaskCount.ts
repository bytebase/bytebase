import type { Ref } from "vue";
import { ref, watch } from "vue";
import { useDBGroupStore } from "@/store";
import { isValidDatabaseGroupName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";

export const useExpectedTaskCount = (plan: Ref<Plan>) => {
  const dbGroupStore = useDBGroupStore();
  const expectedTaskCount = ref(0);

  const countTargets = async (targets: string[]): Promise<number> => {
    let count = 0;
    for (const target of targets) {
      if (isValidDatabaseGroupName(target)) {
        try {
          const dbGroup = await dbGroupStore.getOrFetchDBGroupByName(target, {
            view: DatabaseGroupView.FULL,
            silent: true,
          });
          count += dbGroup.matchedDatabases?.length ?? 0;
        } catch {
          // Ignore errors
        }
      } else {
        count++;
      }
    }
    return count;
  };

  const update = async () => {
    let count = 0;
    for (const spec of plan.value.specs) {
      if (spec.config.case === "changeDatabaseConfig") {
        count += await countTargets(spec.config.value.targets ?? []);
      }
    }
    expectedTaskCount.value = count;
  };

  watch(() => plan.value.specs, update, { immediate: true });

  return { expectedTaskCount };
};
