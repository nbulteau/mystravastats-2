import { describe, expect, it } from "vitest";
import { formatIsoDateDayFirst } from "@/utils/formatters";

describe("formatIsoDateDayFirst", () => {
  it("formats an ISO calendar date without timezone conversion", () => {
    expect(formatIsoDateDayFirst("2017-07-09")).toBe("09-07-2017");
    expect(formatIsoDateDayFirst("2017-07-09T09:30:00+02:00")).toBe("09-07-2017");
  });

  it("keeps an unknown date representation unchanged", () => {
    expect(formatIsoDateDayFirst("09/07/2017")).toBe("09/07/2017");
  });
});
