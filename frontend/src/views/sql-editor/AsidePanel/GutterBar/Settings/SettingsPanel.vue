<template>
  <Drawer v-model:show="showPanel" :close-on-esc="false" :mask-closable="false">
    <DrawerContent
      :title="$t('common.settings')"
      style="width: calc(100vw - 3rem)"
      :content-props="{
        bodyContentStyle: 'padding: 0',
      }"
    >
      <div class="w-full h-full flex">
        <div class="w-[20rem] self-stretch border-r border-block-border">
          11
        </div>
        <div class="w-full h-full overflow-y-auto relative">
          <InstanceForm :instance="undefined">
            <div class="p-2">
              <InstanceFormBody />
            </div>
            <InstanceFormButtons
              :allow-cancel="false"
              class="sticky bottom-0 p-2 bg-white border-t border-block-border"
            />
          </InstanceForm>
        </div>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import {
  InstanceForm,
  Form as InstanceFormBody,
  Buttons as InstanceFormButtons,
} from "@/components/InstanceForm";
import { Drawer, DrawerContent } from "@/components/v2";
import { useSettingsContext } from "./context";

const { showPanel } = useSettingsContext();
</script>
