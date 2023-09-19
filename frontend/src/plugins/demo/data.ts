import { merge } from "lodash-es";
import { validateStepData } from "./guide";
import { DemoData, GuideData } from "./types";

const demoDataCache = new Map<string, DemoData>();

const fetchJSONData = async (path: string) => {
  const res = await fetch("/demo" + path);
  const data = await res.json();
  return data;
};

export const fetchGuideDataWithName = async (guideName: string) => {
  const recorderData = await fetchJSONData(`/${guideName}/recorder.json`);
  const guideRawData = await fetchJSONData(`/${guideName}/guide.json`);
  const guideData = merge(recorderData, guideRawData) as GuideData;
  guideData.steps = guideData.steps.filter((s) => validateStepData(s));
  return guideData;
};

export const fetchDemoDataWithName = async (demoName: string) => {
  if (demoDataCache.has(demoName)) {
    return demoDataCache.get(demoName) as DemoData;
  }

  const demoData = (await fetchJSONData(`/${demoName}/demo.json`)) as DemoData;
  demoDataCache.set(demoName, demoData);

  return demoData;
};
