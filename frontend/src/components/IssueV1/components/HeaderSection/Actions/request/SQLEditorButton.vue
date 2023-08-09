<template>
  <NButton type="primary" size="large" @click="gotoSQLEditor">
    <template #icon>
      <heroicons-solid:terminal class="w-5 h-5" />
    </template>
    {{ $t("sql-editor.self") }}
  </NButton>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";

const gotoSQLEditor = async () => {
  alert("gogogo sql editor");
  // const grantRequest = issue.value.payload.grantRequest as GrantRequestPayload;
  // const conditionExpression = await convertFromCELString(
  //   grantRequest.condition.expression
  // );
  // if (
  //   conditionExpression.databaseResources !== undefined &&
  //   conditionExpression.databaseResources.length > 0
  // ) {
  //   const databaseResourceName = conditionExpression.databaseResources[0]
  //     .databaseName as string;
  //   const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
  //     databaseResourceName
  //   );
  //   if (db.uid !== String(UNKNOWN_ID)) {
  //     const slug = connectionV1Slug(db.instanceEntity, db);
  //     const url = router.resolve({
  //       name: "sql-editor.detail",
  //       params: {
  //         connectionSlug: slug,
  //       },
  //     });
  //     window.open(url.fullPath, "__BLANK");
  //     return;
  //   }
  // }
  // const url = router.resolve({
  //   name: "sql-editor.home",
  // });
  // window.open(url.fullPath, "__BLANK");
};
</script>
