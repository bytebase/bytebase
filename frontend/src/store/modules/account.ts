import { getUserTypeByEmail } from "@/types";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import { useServiceAccountStore } from "./serviceAccount";
import { useUserStore } from "./user";
import { useWorkloadIdentityStore } from "./workloadIdentity";

export const batchGetOrFetchAccounts = async (nameList: string[]) => {
  const userStore = useUserStore();
  const serviceAccountStore = useServiceAccountStore();
  const workloadIdentityStore = useWorkloadIdentityStore();

  const endUsers: string[] = [];
  const serviceAccounts: string[] = [];
  const workloadIdentities: string[] = [];

  for (const name of nameList) {
    const userType = getUserTypeByEmail(name);
    switch (userType) {
      case UserType.SERVICE_ACCOUNT:
        serviceAccounts.push(name);
        break;
      case UserType.WORKLOAD_IDENTITY:
        workloadIdentities.push(name);
        break;
      default:
        endUsers.push(name);
        break;
    }
  }

  await Promise.all([
    userStore.batchGetOrFetchUsers(endUsers),
    serviceAccountStore.batchGetOrFetchServiceAccounts(serviceAccounts),
    workloadIdentityStore.batchGetOrFetchWorkloadIdentities(workloadIdentities),
  ]);
};
