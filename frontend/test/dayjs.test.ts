import moment from "moment";
import dayjs from "dayjs";

import utc from "dayjs/plugin/utc";
import LocalizedFormat from "dayjs/plugin/LocalizedFormat";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
import dayOfYear from "dayjs/plugin/dayOfYear";
import duration from "dayjs/plugin/duration";
import relativeTime from "dayjs/plugin/relativeTime";

dayjs.extend(utc);
dayjs.extend(LocalizedFormat);
dayjs.extend(isSameOrAfter);
dayjs.extend(dayOfYear);
dayjs.extend(duration);
dayjs.extend(relativeTime);

const TS = 60 * 1000;
const now = new Date();

test("UTC Local format", () => {
  expect(dayjs.utc().local().format("YYYYMMDDTHHmmss")).toEqual(
    moment.utc().local().format("YYYYMMDDTHHmmss")
  );
  expect(dayjs.utc(TS).local().format("HH:mm")).toEqual(
    moment.utc(TS).local().format("HH:mm")
  );
  expect(dayjs.utc(TS).local().format("MMM D HH:mm")).toEqual(
    moment.utc(TS).local().format("MMM D HH:mm")
  );
  expect(dayjs.utc(TS).local().format("MMM D")).toEqual(
    moment.utc(TS).local().format("MMM D")
  );
  expect(dayjs.utc(TS).local().format("MMM D YYYY")).toEqual(
    moment.utc(TS).local().format("MMM D YYYY")
  );
});

test("year", () => {
  expect(dayjs().year()).toEqual(moment().year());
});

test("dayOfYear", () => {
  expect(dayjs().dayOfYear()).toEqual(moment().dayOfYear());
});

test("diff", () => {
  expect(dayjs().diff(dayjs("2021-12-25"), "days")).toEqual(
    moment().diff(moment("2021-12-25"), "days")
  );
});

test("format LLL", () => {
  expect(dayjs(TS).format("LLL")).toEqual(moment(TS).format("LLL"));
});

test("isSameOrAfter", () => {
  // expect false
  expect(dayjs("2021-12-25").isSameOrAfter(now, "day")).toEqual(
    moment(moment("2021-12-25")).isSameOrAfter(now, "day")
  );
  // expect true
  expect(dayjs("2022-02-01").isSameOrAfter(now, "day")).toEqual(
    moment(moment("2022-02-01")).isSameOrAfter(now, "day")
  );
});

test("duration", () => {
  expect(dayjs.duration(TS).humanize()).toEqual(moment.duration(TS).humanize());
});
