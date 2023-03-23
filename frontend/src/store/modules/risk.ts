import { ref } from "vue";
import { defineStore } from "pinia";
import { random } from "lodash-es";

import { riskServiceClient } from "@/grpcweb";
import { PresetRiskLevelList } from "@/types";
import {
  Risk,
  Risk_Source,
  risk_SourceToJSON,
} from "@/types/proto/v1/risk_service";
import { randomString } from "@/utils";
import DEMO_RULES from "@/components/CustomApproval/Settings/demo";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export const useRiskStore = defineStore("tab", () => {
  const riskList = ref<Risk[]>([]);

  const fetchRiskList = async () => {
    try {
      const response = await riskServiceClient.listRisks({
        pageSize: 100,
      });
      debugger;
      riskList.value = response.risks;

      // const list = generateMockRisks();
      // riskList.value.push(...list);
    } catch (err) {
      // debugger;
      console.error(err);
    }
    return riskList.value;
  };

  const upsertRisk = async (risk: Risk) => {
    await sleep(1000);
    const existedRisk = riskList.value.find((r) => r.name === risk.name);
    if (existedRisk) {
      // update
      // const updated = await riskServiceClient.updateRisk({
      //   risk,
      // });
      // Object.assign(existedRisk, updated);
      Object.assign(existedRisk, risk);
    } else {
      // create
      const created = await riskServiceClient.createRisk({
        risk,
      });
      debugger;
      Object.assign(risk, created);
      riskList.value.push(risk);
    }
  };

  const deleteRisk = async (risk: Risk) => {
    // await riskServiceClient.deleteRisk({
    //   name: risk.name,
    // });

    await sleep(500);
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

const _randomName = (m = 6, pmin = 5, pmax = 15) => {
  const n = random(1, m);
  const parts: string[] = [];
  for (let i = 0; i < n; i++) {
    const len = random(pmin, pmax);
    parts.push(randomString(len));
  }
  return parts.join(" ");
};

// const generateMockFlow = (
//   roles: ApprovalNode_RoleValue[]
// ): ApprovalTemplate => {
//   const uid = Math.random();
//   return ApprovalTemplate.fromJSON({
//     name: `approvalTemplates/${uid}`,
//     uid,
//     title: randomName(2, 3, 10),
//     description: randomName(10),
//     flow: ApprovalFlow.fromJSON({
//       steps: roles.map((role) => ({
//         type: ApprovalStep_Type.OR,
//         nodes: [
//           {
//             uid: Math.random(),
//             type: ApprovalNode_Type.ROLE,
//             roleValue: role,
//           },
//         ],
//       })),
//     }),
//   });
// };
// approvalTemplateList.value.push(
//   generateMockFlow([
//     ApprovalNode_RoleValue.PROJECT_OWNER,
//     ApprovalNode_RoleValue.DBA,
//   ]),
//   generateMockFlow([ApprovalNode_RoleValue.PROJECT_OWNER]),
//   generateMockFlow([ApprovalNode_RoleValue.DBA]),
//   generateMockFlow([ApprovalNode_RoleValue.WORKSPACE_OWNER]),
//   generateMockFlow([
//     ApprovalNode_RoleValue.PROJECT_OWNER,
//     ApprovalNode_RoleValue.DBA,
//     ApprovalNode_RoleValue.WORKSPACE_OWNER,
//   ])
// );
// approvalFlowList.value.sort((a, b) => {
//   if (a.type !== b.type) {
//     return a.type === "SYSTEM" ? -1 : 1;
//   }
//   return a.id < b.id ? -1 : 1;
// });

const generateMockRisk = (level: number, source: Risk_Source) => {
  const uid = String(Math.random());

  const exprs =
    source === Risk_Source.DDL
      ? DEMO_RULES.DDL
      : source === Risk_Source.DML
      ? DEMO_RULES.DML
      : DEMO_RULES.common;
  const demo = exprs[random(exprs.length - 1)];
  if (!demo) return undefined;
  const title = `${demo.key} - ${random(1000, 9999)}`;
  const expression = ParsedExpr.fromJSON({
    expr: demo.expr,
  });

  return Risk.fromJSON({
    uid,
    name: `risks/${risk_SourceToJSON(source)}-${Math.random()}`,
    level,
    title,
    source,
    expression,
    active: Math.random() < 0.8,
  });
};

const generateMockRisks = () => {
  const list: Risk[] = [];
  [Risk_Source.DDL, Risk_Source.DML, Risk_Source.CREATE_DATABASE].forEach(
    (source) => {
      PresetRiskLevelList.forEach(({ level }) => {
        const n = random(0, 5);
        for (let i = 0; i < n; i++) {
          const risk = generateMockRisk(level, source);
          if (risk) {
            list.push(risk);
          }
        }
      });
    }
  );
  return list;
};
