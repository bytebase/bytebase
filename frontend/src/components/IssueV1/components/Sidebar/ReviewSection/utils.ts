import { uniq } from "lodash-es";
import { computed, inject, provide } from "vue";
import type { InjectionKey, Ref } from "vue";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import { useSubscriptionV1Store, useInstanceResourceByName } from "@/store";
import { isValidDatabaseName } from "@/types";

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

  const distinctInstanceList = computed(() => {
    const instances =
      issue.value.rolloutEntity?.stages.flatMap((stage) => {
        return stage.tasks
          .map((task) => databaseForTask(issue.value, task))
          .filter((db) => isValidDatabaseName(db.name))
          .map((db) => db.instance);
      }) ?? [];

    const resp = [];
    for (const instanceName of uniq(instances)) {
      const { instance } = useInstanceResourceByName(instanceName);
      resp.push(instance);
    }
    return resp;
  });

  const existedDeactivatedInstance = computed(() => {
    return distinctInstanceList.value.some((ins) =>
      subscriptionStore.instanceMissingLicense(
        "bb.feature.custom-approval",
        ins.value
      )
    );
  });

  const context: IssueIntanceContext = {
    existedDeactivatedInstance,
  };

  provide(KEY, context);
  return context;
};
