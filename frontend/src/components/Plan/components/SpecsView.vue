<template>
  <div class="w-full flex-1 flex pt-2 pb-4">
    <SpecDetailView :key="selectedSpec.id" />
  </div>
</template>

<script setup lang="ts">
import { type Ref } from "vue";
import { useCurrentProjectV1 } from "@/store";
import { type Plan_Spec } from "@/types/proto/v1/plan_service";
import { usePlanContext } from "../logic/context";
import { providePlanSQLCheckContext } from "./SQLCheckSection";
import SpecDetailView from "./SpecDetailView";
import { providePlanSpecContext } from "./SpecDetailView/context";

const { project } = useCurrentProjectV1();
const { isCreating, plan } = usePlanContext();

const { selectedSpec } = providePlanSpecContext({
  isCreating,
  plan,
});

providePlanSQLCheckContext({
  project,
  plan,
  selectedSpec: selectedSpec as Ref<Plan_Spec>,
});
</script>
