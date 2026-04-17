import { afterEach, describe, expect, it, vi } from "vitest";
import {
  calculateYtdAverageLine,
  extractPeriodEntries,
  isoWeekNumber,
  parseActivityDate,
  rollingAverage,
  weeksInIsoYear,
  weekLabel,
} from "@/utils/charts";

describe("charts utils", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("maps numeric week key to ISO-style week label", () => {
    // GIVEN
    const rawWeek = "7";

    // WHEN
    const label = weekLabel(rawWeek);

    // THEN
    expect(label).toBe("W07");
  });

  it("extracts period entries from structured API objects", () => {
    // GIVEN
    const source = [
      { periodKey: "01", value: 12.3, activityCount: 4 },
      { periodKey: "02", value: 0, activityCount: 0 },
    ];

    // WHEN
    const entries = extractPeriodEntries(source);

    // THEN
    expect(entries).toEqual([
      { key: "01", value: 12.3, activityCount: 4 },
      { key: "02", value: 0, activityCount: 0 },
    ]);
  });

  it("keeps backward compatibility with legacy map payloads", () => {
    // GIVEN
    const source = [{ "01": 12.3 }, { "02": 0 }];

    // WHEN
    const entries = extractPeriodEntries(source);

    // THEN
    expect(entries).toEqual([
      { key: "01", value: 12.3, activityCount: 0 },
      { key: "02", value: 0, activityCount: 0 },
    ]);
  });

  it("computes YTD average for current year weeks only", () => {
    // GIVEN
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-17T10:00:00Z"));
    const values = [...Array(16).fill(10), ...Array(36).fill(100)];

    // WHEN
    const ytdAverageLine = calculateYtdAverageLine(values, "2026", "WEEKS");

    // THEN
    expect(ytdAverageLine).toHaveLength(52);
    expect(ytdAverageLine.every((value) => value === 10)).toBe(true);
  });

  it("uses full-year average for past years", () => {
    // GIVEN
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-04-17T10:00:00Z"));
    const values = [10, 20, 30, 40];

    // WHEN
    const averageLine = calculateYtdAverageLine(values, "2025", "MONTHS");

    // THEN
    expect(averageLine).toEqual([25, 25, 25, 25]);
  });

  it("computes ISO week number from date", () => {
    // GIVEN
    const date = new Date("2026-01-01T12:00:00Z");

    // WHEN
    const week = isoWeekNumber(date);

    // THEN
    expect(week).toBe(1);
  });

  it("returns 52 or 53 ISO weeks depending on year", () => {
    // GIVEN
    const yearWith52Weeks = 2025;
    const yearWith53Weeks = 2026;

    // WHEN
    const weeks52 = weeksInIsoYear(yearWith52Weeks);
    const weeks53 = weeksInIsoYear(yearWith53Weeks);

    // THEN
    expect(weeks52).toBe(52);
    expect(weeks53).toBe(53);
  });

  it("parses plain activity date strings", () => {
    // GIVEN
    const rawDate = "2026-04-17";

    // WHEN
    const parsed = parseActivityDate(rawDate);

    // THEN
    expect(parsed).not.toBeNull();
    expect(parsed?.getFullYear()).toBe(2026);
  });

  it("computes rolling average while skipping null values", () => {
    // GIVEN
    const values = [10, null, 20, 30, null];

    // WHEN
    const average = rollingAverage(values, 3);

    // THEN
    expect(average).toEqual([10, 10, 15, 25, 25]);
  });
});
