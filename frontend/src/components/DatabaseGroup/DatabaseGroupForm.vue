<template>
  <div>
    <ExprEditor
      :expr="state.expr"
      :allow-admin="true"
      :allow-high-level-factors="false"
    />
  </div>
</template>

<script lang="ts" setup>
import { ConditionGroupExpr, resolveCELExpr, wrapAsGroup } from "@/plugins/cel";
import { reactive } from "vue";

import { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import ExprEditor from "./common/ExprEditor";
import { ResourceType } from "./common/ExprEditor/context";

type LocalState = {
  expr: ConditionGroupExpr;
  resourceType: ResourceType;
};

const state = reactive<LocalState>({
  expr: wrapAsGroup(resolveCELExpr(CELExpr.fromJSON({}))),
  resourceType: "DATABASE_GROUP",
});
</script>
