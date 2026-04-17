import { afterEach, describe, expect, it, vi } from "vitest";
import {
  calculateYtdAverageLine,
  extractPeriodEntries,
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

  it("extracts period entries from API objects", () => {
    // GIVEN
    const source = [{ "01": 12.3 }, { "02": 0 }];

    // WHEN
    const entries = extractPeriodEntries(source);

    // THEN
    expect(entries).toEqual([
      { key: "01", value: 12.3 },
      { key: "02", value: 0 },
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
});
