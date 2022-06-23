import initGuideListeners from "./listener";

// initial guide listeners when window loaded
window.addEventListener(
  "load",
  () => {
    initGuideListeners();
  },
  {
    once: true,
  }
);

export default null;
