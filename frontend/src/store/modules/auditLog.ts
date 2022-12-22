import { defineStore } from "pinia";
import axios from "axios";
import { stringify } from "qs";
import type {
  AuditLogState,
  ResourceObject,
  ResourceIdentifier,
  ActivityFind,
} from "@/types";
import { empty, isPagedResponse } from "@/types";
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

export const useAuditLogStore = defineStore("auditLog", {
  state: (): AuditLogState => ({
    auditLogList: [],
  }),
  actions: {
    setAuditLogList(auditLogList: AuditLog[]) {
      this.auditLogList = auditLogList;
    },
    async fetchPagedAuditLogList(params: ActivityFind) {
      const url = `/api/activity?${stringify(params, {
        arrayFormat: "repeat",
      })}`;
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
    async fetchAuditLogList(params: ActivityFind) {
      const res = await this.fetchPagedAuditLogList(params);
      this.setAuditLogList(res.auditLogList);
      return res.auditLogList;
    },
  },
});
