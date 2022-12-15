import { watchEffect } from "vue";
import { defineStore, storeToRefs } from "pinia";
import axios from "axios";
import type { AuditLogState, ResponseWithData, ResourceObjects } from "@/types";
import { AuditActivityType } from "@/types";
import type { AuditLog } from "@/types/auditLog";

const convertAuditLogList = (res: ResponseWithData): AuditLog[] => {
  const activityList = res.data as ResourceObjects;
  const auditLogList: AuditLog[] = activityList.map(
    (activity) =>
      ({
        createdTs: activity.attributes.createdTs as number,
        creator: res.included!.find(
          (item) =>
            item.id ===
            (
              activity.relationships!.creator.data as {
                id: string;
                type: string;
              }
            ).id
        )?.attributes.email as string,
        type: activity.attributes.type as string,
        level: activity.attributes.level as string,
        comment: activity.attributes.comment as string,
        payload: activity.attributes.payload as string,
      } as AuditLog)
  );

  return auditLogList;
};

const typePrefixQuery = `${(
  Object.keys(AuditActivityType) as Array<keyof typeof AuditActivityType>
)
  .map((key) => `typePrefix=${AuditActivityType[key]}`)
  .join("&")}`;

export const useAuditLogStore = defineStore("auditLog", {
  state: (): AuditLogState => ({
    auditLogList: [],
  }),
  actions: {
    setAuditLogList(auditLogList: AuditLog[]) {
      this.auditLogList = auditLogList;
    },
    async fetchAuditLogList() {
      const res = (await axios.get(`/api/activity?${typePrefixQuery}`)).data;
      const list = convertAuditLogList(res);
      this.setAuditLogList(list);
      return list;
    },
  },
});

export const useAuditLogList = () => {
  const store = useAuditLogStore();
  watchEffect(async () => await store.fetchAuditLogList());

  return storeToRefs(store).auditLogList;
};
