import { createSSRApp, h } from "vue";
import { renderToString } from "@vue/server-renderer";
import { describe, expect, it, vi } from "vitest";
import ActivityDetailHero from "@/components/activity-detail/ActivityDetailHero.vue";

const baseProps = {
  activityName: "Morning Ride",
  activityTypeLabel: "Ride",
  activityDateLabel: "27 August 2026",
  commute: true,
  activityVersion: "corrected" as const,
  activityVersionLabel: "Corrected data",
  effortCountLabel: "3 efforts",
  canSelectCorrectedVersion: true,
  stravaActivityUrl: "https://www.strava.com/activities/42",
};

describe("ActivityDetailHero", () => {
  it("renders its activity metadata and safe Strava link", async () => {
    const html = await renderToString(createSSRApp({
      render: () => h(ActivityDetailHero, baseProps),
    }));

    expect(html).toContain("Morning Ride");
    expect(html).toContain("Commute");
    expect(html).toContain("3 efforts");
    expect(html).toContain('rel="noopener noreferrer"');
    expect(html).toContain("https://www.strava.com/activities/42");
  });

  it("emits version changes from both controls", () => {
    const emit = vi.fn();
    const vnode = h(ActivityDetailHero, { ...baseProps, onVersionChange: emit });
    expect(vnode.props?.onVersionChange).toBe(emit);
    expect(ActivityDetailHero.emits).toContain("versionChange");
  });
});
