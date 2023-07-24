<!--
 * @Author: SIOOOO gesiyuan01@gmail.com
 * @Date: 2023-07-19 23:38:48
 * @LastEditors: SIOOOO gesiyuan01@gmail.com
 * @LastEditTime: 2023-07-20 12:01:06
 * @FilePath: /bytebase/frontend/src/components/SQLReview/components/RuleEngineIcon.vue
 * @Description: 这是默认设置,请设置`customMade`, 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
-->
<template>
  <img class="w-auto h-4" :src="icon" />
</template>

<script lang="ts" setup>
import { SchemaRuleEngineType } from "@/types";
import { computed } from "vue";

const props = defineProps<{
  engine: SchemaRuleEngineType;
}>();

const icon = computed(() => {
  if (props.engine === "ORACLE") {
    return new URL("../../../assets/db-oracle.svg", import.meta.url).href;
  } else if (props.engine === "MSSQL") {
    return new URL("../../../assets/db-mssql.svg", import.meta.url).href;
  }
  return new URL(
    `../../../assets/db-${props.engine.toLowerCase()}.png`,
    import.meta.url
  ).href;
});
</script>
