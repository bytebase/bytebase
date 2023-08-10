import { defineStore } from "pinia";
import { ref } from "vue";
import { riskServiceClient } from "@/grpcweb";
import { Risk } from "@/types/proto/v1/risk_service";

export const useRiskStore = defineStore("risk", () => {
  const riskList = ref<Risk[]>([]);

  const fetchRiskList = async () => {
    const response = await riskServiceClient.listRisks({
      pageSize: 100,
    });
    riskList.value = response.risks;
    return riskList.value;
  };

  const upsertRisk = async (risk: Risk) => {
    const existedRisk = riskList.value.find((r) => r.name === risk.name);
    if (existedRisk) {
      // update
      const updated = await riskServiceClient.updateRisk({
        risk,
        updateMask: ["title", "level", "active", "condition"],
      });
      Object.assign(existedRisk, updated);
    } else {
      // create
      const created = await riskServiceClient.createRisk({
        risk,
      });
      Object.assign(risk, created);
      riskList.value.push(risk);
    }
  };

  const deleteRisk = async (risk: Risk) => {
    await riskServiceClient.deleteRisk({
      name: risk.name,
    });
    const index = riskList.value.findIndex((r) => r.name === risk.name);
    if (index >= 0) {
      riskList.value.splice(index, 1);
    }
  };

  return {
    riskList,
    fetchRiskList,
    upsertRisk,
    deleteRisk,
  };
});
