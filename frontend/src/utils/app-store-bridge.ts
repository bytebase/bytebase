import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import type {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { Environment } from "@/types/v1/environment";

// Narrow, synchronous accessors into the React app store for shared `@/utils`
// code. The app store *registers* these at init (see `@/react/stores/app`)
// rather than `@/utils` importing the store directly — a static import would
// create an `@/utils` → app-store load-time cycle (the barrel-graph hazard
// that also gates the permission checkers in `./iam/permission`).
//
// Callers must supply their own fallback for the pre-registration window
// (`appStoreUtilBridge()` returns null until the app store module has loaded,
// which happens at boot, before any render-time call).
export interface AppStoreUtilBridge {
  currentUser: () => User | undefined;
  isLoggedIn: () => boolean;
  setUnauthenticatedOccurred: (value: boolean) => void;
  defaultProjectName: () => string;
  currentPlan: () => PlanType;
  hasFeature: (feature: PlanFeature) => boolean;
  environmentList: () => Environment[];
  getEnvironmentByName: (name: string, fallback?: boolean) => Environment;
  roleList: () => Role[];
  getRoleByName: (name: string) => Role | undefined;
  getGroupByIdentifier: (identifier: string) => Group | undefined;
  workspaceRoleMapToUsers: () => Map<string, Set<string>>;
  getProjectIamPolicy: (project: string) => IamPolicy;
}

let bridge: AppStoreUtilBridge | null = null;

export const registerAppStoreUtilBridge = (b: AppStoreUtilBridge): void => {
  bridge = b;
};

export const appStoreUtilBridge = (): AppStoreUtilBridge | null => bridge;
