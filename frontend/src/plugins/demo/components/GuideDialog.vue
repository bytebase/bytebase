<template>
  <teleport to="body">
    <div v-if="shouldShow" ref="coverWrapperRef" class="cover-wrapper"></div>
    <Transition>
      <div
        v-if="shouldShow"
        ref="dialogWrapperRef"
        class="guide-wrapper"
        :class="position"
      >
        <p class="flex flex-row justify-start items-center">
          <BBSpin v-if="loading" class="mr-1" />
          {{ props.title }}
        </p>
      </div>
    </Transition>
    <Transition>
      <div
        v-if="shouldShow"
        ref="highlightWrapperRef"
        class="highlight-wrapper"
      ></div>
    </Transition>
  </teleport>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import {
  getElementBounding,
  getElementMaxZIndex,
  getScrollParent,
  getStylePropertyValue,
  waitForTargetElement,
} from "../utils";

const props = defineProps({
  title: {
    type: String,
    default: "",
  },
  description: {
    type: String,
    default: "",
  },
  position: {
    type: String,
    default: "bottom",
  },
  targetElementSelector: {
    type: String,
    default: "",
  },
  loading: {
    type: Boolean,
    default: false,
  },
});

const shouldShow = ref(false);
const targetElementRef = ref<HTMLElement>();
const dialogWrapperRef = ref<HTMLElement>();
const highlightWrapperRef = ref<HTMLElement>();
const coverWrapperRef = ref<HTMLElement>();

const initialElements = async () => {
  const clonedProps = cloneDeep(props);
  const targetElement = await waitForTargetElement(
    clonedProps.targetElementSelector
  );
  if (!targetElement) {
    return;
  }

  targetElementRef.value = targetElement;
  targetElement.classList.add("bb-guide-target-element");

  const parentEl = getScrollParent(targetElementRef.value);
  parentEl.addEventListener("scroll", handleScrollableParentScroll);

  shouldShow.value = true;
  await updateDialogPosition();
};

onMounted(() => {
  nextTick(async () => {
    await initialElements();
  });
});

onUnmounted(() => {
  targetElementRef.value?.classList.remove("bb-guide-target-element");
  const parentEl = getScrollParent(targetElementRef.value);
  parentEl.removeEventListener("scroll", handleScrollableParentScroll);
});

watch([props], () => {
  nextTick(async () => {
    await initialElements();
  });
});

const handleScrollableParentScroll = () => {
  if (!targetElementRef.value) {
    return;
  }
  const parentEl = getScrollParent(targetElementRef.value);
  if (
    targetElementRef.value.offsetTop + 8 <
    parentEl.scrollTop + parentEl.offsetTop
  ) {
    shouldShow.value = false;
  } else {
    shouldShow.value = true;
  }
};

const updateDialogPosition = async () => {
  if (
    !coverWrapperRef.value ||
    !dialogWrapperRef.value ||
    !highlightWrapperRef.value
  ) {
    requestAnimationFrame(() => updateDialogPosition());
    return;
  }

  const targetElement = document.body.querySelector(
    props.targetElementSelector
  ) as HTMLElement;
  if (!targetElement) {
    shouldShow.value = false;
    return;
  }

  if (getStylePropertyValue(targetElement, "border-radius") !== "0px") {
    highlightWrapperRef.value.style.borderRadius = getStylePropertyValue(
      targetElement,
      "border-radius"
    );
  }

  targetElement.classList.add("bb-guide-target-element");
  const maxZIndex = getElementMaxZIndex(targetElement);
  coverWrapperRef.value.style.zIndex = `${Math.max(maxZIndex - 1, 0)}`;
  highlightWrapperRef.value.style.zIndex = `${maxZIndex}`;
  dialogWrapperRef.value.style.zIndex = `${maxZIndex + 1}`;

  const bounding = getElementBounding(targetElement);
  if (bounding.width === 0 || bounding.height === 0) {
    shouldShow.value = false;
    return;
  }

  if (props.position !== "center") {
    highlightWrapperRef.value.style.top = `${bounding.top}px`;
    highlightWrapperRef.value.style.left = `${bounding.left}px`;
    highlightWrapperRef.value.style.width = `${bounding.width}px`;
    highlightWrapperRef.value.style.height = `${bounding.height}px`;
  }

  const dialogBounding = getElementBounding(dialogWrapperRef.value);
  if (props.position === "bottom") {
    dialogWrapperRef.value.style.top = `${bounding.top + bounding.height}px`;
    dialogWrapperRef.value.style.left = `${bounding.left - 4}px`;
  } else if (props.position === "top") {
    dialogWrapperRef.value.style.top = `${
      bounding.top - dialogBounding.height - 8
    }px`;
    dialogWrapperRef.value.style.left = `${bounding.left - 4}px`;
  } else if (props.position === "left") {
    dialogWrapperRef.value.style.top = `${bounding.top - 8}px`;
    dialogWrapperRef.value.style.left = `${
      bounding.left - dialogBounding.width - 12
    }px`;
  } else if (props.position === "right") {
    dialogWrapperRef.value.style.top = `${bounding.top - 4}px`;
    dialogWrapperRef.value.style.left = `${bounding.left + bounding.width}px`;
  } else if (props.position === "bottom-right") {
    dialogWrapperRef.value.style.top = `${bounding.top + bounding.height}px`;
    dialogWrapperRef.value.style.left = `${
      bounding.left + bounding.width - dialogBounding.width
    }px`;
  } else if (props.position === "bottom-center") {
    dialogWrapperRef.value.style.top = `${bounding.top + bounding.height}px`;
    dialogWrapperRef.value.style.left = `${
      bounding.left + (bounding.width - dialogBounding.width) / 2
    }px`;
  }
  requestAnimationFrame(() => updateDialogPosition());
};
</script>

<style>
.bb-guide-target-element {
  z-index: 1;
}
</style>

<style scoped>
.guide-wrapper {
  @apply fixed -top-full -left-full bg-white w-64 m-1 my-2 p-3 px-4 text-base text-slate-900 rounded-lg text-left;
  z-index: 10000;
  box-shadow: 0 4px 36px rgb(33 33 33 / 40%);
}

.highlight-wrapper {
  @apply fixed pointer-events-none rounded-lg;
  box-shadow: 0 4px 36px rgb(33 33 33 / 40%),
    rgb(33 33 33 / 50%) 0px 0px 0px 100vw;
  z-index: 9999;
}

.cover-wrapper {
  @apply fixed inset-0 overflow-hidden;
  pointer-events: auto !important;
}

.v-enter-active,
.v-leave-active {
  transition: opacity 0.3s ease-in;
}

.v-enter-from,
.v-leave-to {
  opacity: 0;
}

.guide-wrapper.top {
  margin-top: -4px;
}
.guide-wrapper.top::before {
  content: "";
  position: absolute;
  bottom: -8px;
  left: 24px;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-top: 8px solid white;
}

.guide-wrapper.bottom {
  margin-top: 16px;
}
.guide-wrapper.bottom::before {
  content: "";
  position: absolute;
  top: -8px;
  left: 24px;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-bottom: 8px solid white;
}

.guide-wrapper.bottom-center {
  @apply w-auto text-base;
  margin-top: 12px;
}

.guide-wrapper.bottom-center::before {
  content: "";
  position: absolute;
  top: -8px;
  left: 50%;
  margin-left: -12px;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-bottom: 8px solid white;
}

.guide-wrapper.bottom-right {
  margin-top: 12px;
}

.guide-wrapper.bottom-right::before {
  content: "";
  position: absolute;
  top: -8px;
  right: 24px;
  border-left: 8px solid transparent;
  border-right: 8px solid transparent;
  border-bottom: 8px solid white;
}

.guide-wrapper.left {
  margin: 0;
  margin-top: -6px;
}
.guide-wrapper.left::before {
  content: "";
  position: absolute;
  top: 8px;
  right: -10px;
  border-top: 10px solid transparent;
  border-bottom: 10px solid transparent;
  border-left: 10px solid white;
}

.guide-wrapper.right {
  margin-top: -10px;
}
.guide-wrapper.right::before {
  content: "";
  position: absolute;
  top: 16px;
  left: -10px;
  border-top: 10px solid transparent;
  border-bottom: 10px solid transparent;
  border-right: 10px solid white;
}

.guide-wrapper.center {
  @apply w-auto top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 -mt-16 whitespace-nowrap;
}
</style>
