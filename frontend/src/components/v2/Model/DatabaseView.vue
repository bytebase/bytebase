<template>
  <div class="flex flex-row justify-start items-center gap-x-1" v-bind="$attrs">
    <FeatureBadge
      :feature="PlanFeature.FEATURE_DATABASE_GROUPS"
      class="mr-2"
      :instance="getInstanceResource(database)"
    />
    <InstanceV1EngineIcon :instance="getInstanceResource(database)" />
    <EnvironmentV1Name
      text-class="text-control-light"
      :environment="getDatabaseEnvironment(database)"
      :plain="true"
      :show-icon="false"
      :link="false"
    />
    <DatabaseV1Name
      :database="database"
      :plain="true"
      :show-icon="false"
      :link="false"
    />
  </div>
</template>

<script lang="ts" setup>
import { FeatureBadge } from "@/components/FeatureGuard";
import { DatabaseV1Name, EnvironmentV1Name } from "@/components/v2";
import { useDatabaseV1ByName } from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { getDatabaseEnvironment, getInstanceResource } from "@/utils";
import { InstanceV1EngineIcon } from "./Instance";

const props = defineProps<{
  database: string;
}>();

const { database } = useDatabaseV1ByName(props.database);
</script>
