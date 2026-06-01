import { describe, expect, it } from "vitest";
import {
  normalizeAthletePerformanceSettings,
  resolveManualFtpForDate,
} from "@/models/athlete-performance-settings.model";

describe("athlete performance settings", () => {
  it("normalizes FTP history by date and positive values", () => {
    const settings = normalizeAthletePerformanceSettings({
      ftpHistory: [
        { effectiveFrom: "2026-02-01", ftp: 170 },
        { effectiveFrom: "bad-date", ftp: 999 },
        { effectiveFrom: "2026-01-01", ftp: 160 },
        { effectiveFrom: "2026-02-01", ftp: 175 },
        { effectiveFrom: "2026-03-01", ftp: -1 },
      ],
      weightKg: 72.5,
    });

    expect(settings).toEqual({
      ftpHistory: [
        { effectiveFrom: "2026-01-01", ftp: 160 },
        { effectiveFrom: "2026-02-01", ftp: 175 },
      ],
      weightKg: 72.5,
    });
  });

  it("resolves the latest manual FTP applicable to the activity date", () => {
    const resolved = resolveManualFtpForDate(
      {
        ftpHistory: [
          { effectiveFrom: "2026-01-01", ftp: 160 },
          { effectiveFrom: "2026-06-01", ftp: 180 },
        ],
        weightKg: null,
      },
      "2026-05-31T11:14:00Z",
    );

    expect(resolved).toEqual({ effectiveFrom: "2026-01-01", ftp: 160 });
  });
});
