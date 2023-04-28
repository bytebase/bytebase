<template>
  <div
    class="w-full min-w-[128px] flex flex-row justify-start items-start gap-4"
  >
    <div class="w-1/4 flex flex-col justify-start items-start gap-2">
      <span class="font-medium capitalize whitespace-nowrap">{{
        $t("slow-query.advise-index.current-index")
      }}</span>
      <span class="font-mono">{{ state.currentIndex }}</span>
    </div>
    <div class="w-3/4 flex flex-col justify-start items-start gap-2">
      <div>
        <span class="font-medium capitalize whitespace-nowrap">{{
          $t("slow-query.advise-index.suggestion")
        }}</span>
        <button class="ml-2 normal-link underline" @click="handleCreateIndex">
          {{ $t("slow-query.advise-index.create-index") }}
        </button>
      </div>
      <span class="w-full font-mono">{{ state.suggestion }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { ComposedSlowQueryLog } from "@/types";
import { reactive } from "vue";
import { watch } from "vue";
import dayjs from "dayjs";
import { useRouter } from "vue-router";

const props = defineProps<{
  slowQueryLog: ComposedSlowQueryLog;
}>();

interface LocalState {
  currentIndex: string;
  suggestion: string;
  createIndexStatement: string;
}

const router = useRouter();
const state = reactive<LocalState>({
  currentIndex: "",
  suggestion: "",
  createIndexStatement: "",
});
const log = computed(() => props.slowQueryLog.log);
const database = computed(() => props.slowQueryLog.database);
const sqlFingerprint = computed(
  () => log.value.statistics?.sqlFingerprint || ""
);

const handleCreateIndex = () => {
  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: database.value.projectId,
    mode: "normal",
    ghost: undefined,
  };
  query.databaseList = database.value.id;
  query.sql = state.createIndexStatement;
  query.name = generateIssueName();

  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
  router.push(routeInfo);
};

const generateIssueName = () => {
  const issueNameParts: string[] = [];
  issueNameParts.push(`[${database.value.name}]`);
  issueNameParts.push(`Create index`);
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);
  return issueNameParts.join(" ");
};

watch(
  () => props.slowQueryLog,
  async () => {
    // TODO(junyi): Do data fetching with database and sqlFingerprint.
    // Prevent eslint error.
    console.log(sqlFingerprint);

    state.currentIndex = "test index";
    state.suggestion = "Your suggestion";
    state.createIndexStatement = "CREATE INDEX balabala;";
  },
  {
    immediate: true,
  }
);
</script>
