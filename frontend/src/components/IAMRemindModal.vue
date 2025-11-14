<template>
  <BBModal
    :show="state.showModal"
    :title="$t('remind.role-expire.title')"
    @close="onClose"
  >
    <div class="w-120 max-w-[calc(100vw-10rem)]">
      <div class="flex flex-col gap-y-2">
        <p>{{ $t("remind.role-expire.content") }}</p>
        <ul class="list-disc textinfolabel ml-4">
          <li v-for="(data, i) in pendingExpireRoles" :key="i">
            {{ displayRoleTitle(data.role.name) }}:
            <span class="text-red-400">
              {{ data.expiration.format("YYYY-MM-DD HH:mm:ss") }}
            </span>
          </li>
        </ul>
      </div>
      <NCheckbox v-model:checked="state.checked" class="mt-6">
        <span class="textinfolabel">
          {{ $t("remind.role-expire.checkbox") }}
        </span>
      </NCheckbox>

      <div class="mt-7 flex justify-end gap-x-2">
        <NButton @click="onClose">
          {{ $t("common.dismiss") }}
        </NButton>
        <NButton v-if="!isInProjectPage" type="primary" @click="onClick">
          {{ $t("remind.role-expire.go-to-project") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="tsx" setup>
import dayjs from "dayjs";
import { orderBy } from "lodash-es";
import { NButton, NCheckbox } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { BBModal } from "@/bbkit";
import {
  useCurrentUserV1,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useRoleStore,
} from "@/store";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import {
  autoProjectRoute,
  bindingListInIAM,
  displayRoleTitle,
  useDynamicLocalStorage,
} from "@/utils";
import { convertFromExpr } from "@/utils/issue/cel";

const props = defineProps<{
  projectName: string;
}>();

const state = reactive<{
  showModal: boolean;
  checked: boolean;
}>({
  showModal: false,
  checked: false,
});

const route = useRoute();
const router = useRouter();
const me = useCurrentUserV1();
const iamRemindState = useDynamicLocalStorage<
  Map<string /* {project}.{pending expired roles} */, boolean /* show remind */>
>(
  computed(() => `bb.remind.iam.${me.value.name}`),
  new Map()
);
const projectStore = useProjectV1Store();
const roleStore = useRoleStore();
const projectIamPolicyStore = useProjectIamPolicyStore();

const project = computed(() =>
  projectStore.getProjectByName(props.projectName)
);

const isInProjectPage = computed(() => {
  return route.path.startsWith(`/${props.projectName}`);
});

const pendingExpireRoles = computed(
  (): {
    role: Role;
    expiration: dayjs.Dayjs;
  }[] => {
    const policy = projectIamPolicyStore.getProjectIamPolicy(props.projectName);
    const bindings = bindingListInIAM({
      policy,
      email: me.value.email,
      ignoreGroup: true,
    });

    const results: {
      role: Role;
      expiration: dayjs.Dayjs;
    }[] = [];

    for (const binding of bindings) {
      const role = roleStore.getRoleByName(binding.role);
      if (!role) {
        continue;
      }
      if (!binding.parsedExpr) {
        continue;
      }
      const conditionExpr = convertFromExpr(binding.parsedExpr);
      if (!conditionExpr.expiredTime) {
        continue;
      }
      const expiration = dayjs(conditionExpr.expiredTime);
      const now = dayjs();

      if (now.isBefore(expiration) && now.add(2, "days").isAfter(expiration)) {
        results.push({
          role,
          expiration,
        });
      }
    }

    return orderBy(
      results,
      [(data) => data.expiration.toDate().getTime()],
      "asc"
    );
  }
);

const key = computed(() => {
  if (pendingExpireRoles.value.length === 0) {
    return "";
  }
  return `${props.projectName}.${pendingExpireRoles.value.map((data) => data.role.name).join("&")}`;
});

watch(
  () => key.value,
  (key) => {
    if (!key) {
      return;
    }
    if (!iamRemindState.value.has(key)) {
      iamRemindState.value.set(key, true);
    }
    state.showModal = iamRemindState.value.get(key) ?? false;
  },
  { immediate: true }
);

const onClose = () => {
  if (state.checked) {
    iamRemindState.value.set(key.value, false);
  }
  state.showModal = false;
};

const onClick = () => {
  state.showModal = false;
  router.push({
    ...autoProjectRoute(router, project.value),
  });
};
</script>
