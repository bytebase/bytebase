import { useCurrentUserV1 } from "@/store";
import { QuickActionType, RoleType } from "@/types";
import { isDBA, isDeveloper, isOwner } from "@/utils";

export const getQuickActionList = (
  map: Map<RoleType, QuickActionType[]>
): QuickActionType[] => {
  const currentUserV1 = useCurrentUserV1();
  const role = currentUserV1.value.userRole;

  const list: QuickActionType[] = [];

  // We write this way because for free version, the user wears the three role hat,
  // and we want to display all quick actions relevant to those three roles without duplication.
  if (isOwner(role)) {
    for (const item of map.get("OWNER") || []) {
      list.push(item);
    }
  }

  if (isDBA(role)) {
    for (const item of map.get("DBA") || []) {
      if (
        !list.find((myItem: QuickActionType) => {
          return item == myItem;
        })
      ) {
        list.push(item);
      }
    }
  }

  if (isDeveloper(role)) {
    for (const item of map.get("DEVELOPER") || []) {
      if (
        !list.find((myItem: QuickActionType) => {
          return item == myItem;
        })
      ) {
        list.push(item);
      }
    }
  }
  return list;
};
