import { watchEffect } from "vue";
import { defineStore, storeToRefs } from "pinia";
import axios from "axios";
import type {
  AuditLogState,
  ResourceObject,
  ResourceIdentifier,
} from "@/types";
import { AuditActivityType, empty, isPagedResponse } from "@/types";
import type { AuditLog } from "@/types/auditLog";
import { getPrincipalFromIncludedList } from "./principal";
import { convertEntityList } from "./utils";

const convert = (
  auditLog: ResourceObject,
  includedList: ResourceObject[]
): AuditLog => {
  return {
    ...(auditLog.attributes as Omit<AuditLog, "creator">),
    creator: getPrincipalFromIncludedList(
      auditLog.relationships!.creator.data,
      includedList
    )?.email as string,
  };
};

function getAuditLogFromIncludedList(
  data:
    | ResourceIdentifier<ResourceObject>
    | ResourceIdentifier<ResourceObject>[]
    | undefined,
  includedList: ResourceObject[]
): AuditLog {
  if (data == null) {
    return empty("AUDIT_LOG");
  }
  for (const item of includedList || []) {
    if (item.type !== "activity") {
      continue;
    }
    if (item.id == (data as ResourceIdentifier).id) {
      return convert(item, includedList);
    }
  }
  return empty("AUDIT_LOG");
}

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
    async fetchPagedAuditLogList() {
      const url = `/api/activity?${typePrefixQuery}`;
      const responseData = (await axios.get(url)).data;
      const auditLogList = convertEntityList(
        responseData,
        "activityList",
        convert,
        getAuditLogFromIncludedList
      );
      const nextToken = isPagedResponse(responseData, "activityList")
        ? responseData.data.attributes.nextToken
        : "";
      return {
        nextToken,
        auditLogList,
      };
    },
    async fetchAuditLogList() {
      const res = await this.fetchPagedAuditLogList();
      this.setAuditLogList(res.auditLogList);
      return res.auditLogList;
    },
  },
});

export const useAuditLogList = () => {
  const store = useAuditLogStore();
  watchEffect(async () => await store.fetchAuditLogList());

  return storeToRefs(store).auditLogList;
};
