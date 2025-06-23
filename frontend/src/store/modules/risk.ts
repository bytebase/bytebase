import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { riskServiceClientConnect } from "@/grpcweb";
import type { Risk as OldRisk } from "@/types/proto/v1/risk_service";
import type { Risk } from "@/types/proto-es/v1/risk_service_pb";
import {
  CreateRiskRequestSchema,
  DeleteRiskRequestSchema,
  ListRisksRequestSchema,
  UpdateRiskRequestSchema,
} from "@/types/proto-es/v1/risk_service_pb";
import {
  convertOldRiskToNew,
  convertNewRiskToOld,
} from "@/utils/v1/risk-conversions";

export const useRiskStore = defineStore("risk", () => {
  // Internal state uses proto-es types
  const _riskList = ref<Risk[]>([]);
  
  // Computed property provides old proto type for compatibility
  const riskList = computed(() => {
    return _riskList.value.map(convertNewRiskToOld);
  });

  const fetchRiskList = async () => {
    const request = create(ListRisksRequestSchema, {
      pageSize: 100,
    });
    const response = await riskServiceClientConnect.listRisks(request);
    _riskList.value = response.risks;
    return riskList.value;
  };

  const upsertRisk = async (risk: OldRisk) => {
    const newRisk = convertOldRiskToNew(risk);
    const existedRisk = _riskList.value.find((r) => r.name === newRisk.name);
    if (existedRisk) {
      // update
      const request = create(UpdateRiskRequestSchema, {
        risk: newRisk,
        updateMask: { paths: ["title", "level", "active", "condition", "source"] },
      });
      const updated = await riskServiceClientConnect.updateRisk(request);
      Object.assign(existedRisk, updated);
    } else {
      // create
      const request = create(CreateRiskRequestSchema, {
        risk: newRisk,
      });
      const created = await riskServiceClientConnect.createRisk(request);
      _riskList.value.push(created);
    }
  };

  const deleteRisk = async (risk: OldRisk) => {
    const request = create(DeleteRiskRequestSchema, {
      name: risk.name,
    });
    await riskServiceClientConnect.deleteRisk(request);
    const index = _riskList.value.findIndex((r) => r.name === risk.name);
    if (index >= 0) {
      _riskList.value.splice(index, 1);
    }
  };

  return {
    riskList,
    fetchRiskList,
    upsertRisk,
    deleteRisk,
  };
});
