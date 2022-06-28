import { merge } from "lodash-es";
import { GuideData } from "./types";

const fetchJSONData = async (path: string) => {
  const res = await fetch("/static/guide" + path);
  const data = await res.json();
  return data;
};

export const fetchGuideDataWithName = async (guideName: string) => {
  const recorderData = await fetchJSONData(`/recorder/${guideName}.json`);
  const guideTempData = await fetchJSONData(`/${guideName}.json`);
  const guideData = merge(recorderData, guideTempData) as GuideData;
  guideData.steps = guideData.steps.filter(
    (s) => s.type === "click" || s.type === "change"
  );
  return guideData;
};
