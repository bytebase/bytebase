import { computedAsync } from "@vueuse/core";
import { uniq } from "lodash-es";
import { computed, inject, provide } from "vue";
import type { InjectionKey, Ref } from "vue";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import { useSubscriptionV1Store, useInstanceResourceByName } from "@/store";
import { isValidDatabaseName } from "@/types";
import { extractDatabaseResourceName, wrapRefAsPromise } from "@/utils";

export type IssueIntanceContext = {
  existedDeactivatedInstance: Ref<boolean>;
};

const KEY = Symbol("bb.issue.instances") as InjectionKey<IssueIntanceContext>;

export const useIssueIntanceContext = () => {
  return inject(KEY)!;
};

export const provideIssueIntanceContext = () => {
  const { issue } = useIssueContext();
  const subscriptionStore = useSubscriptionV1Store();

  const distinctInstanceNameList = computed(() => {
    const names =
      issue.value.rolloutEntity?.stages.flatMap((stage) => {
        return stage.tasks
          .map((task) => databaseForTask(issue.value, task).name)
          .filter(isValidDatabaseName)
          .map((dbName) => extractDatabaseResourceName(dbName).instance);
      }) ?? [];
    return uniq(names);
  });

  const existedDeactivatedInstance = computedAsync(async () => {
    for (const instanceName of distinctInstanceNameList.value) {
      const { instance, ready } = useInstanceResourceByName(instanceName);
      await wrapRefAsPromise(ready, /* expected */ true);
      if (
        subscriptionStore.instanceMissingLicense(
          "bb.feature.custom-approval",
          instance.value
        )
      ) {
        return true;
      }
    }
    return false;
  }, false);

  const context: IssueIntanceContext = {
    existedDeactivatedInstance,
  };

  provide(KEY, context);
  return context;
};
