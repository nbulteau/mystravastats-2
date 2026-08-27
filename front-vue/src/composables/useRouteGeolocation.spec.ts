import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useRouteGeolocation } from "@/composables/useRouteGeolocation";

class TestGeolocationError extends Error {
  static readonly PERMISSION_DENIED = 1;
  static readonly POSITION_UNAVAILABLE = 2;
  static readonly TIMEOUT = 3;
  readonly PERMISSION_DENIED = 1;
  readonly POSITION_UNAVAILABLE = 2;
  readonly TIMEOUT = 3;

  constructor(readonly code: number, message: string) {
    super(message);
  }
}

describe("useRouteGeolocation", () => {
  const values = new Map<string, string>();

  beforeEach(() => {
    values.clear();
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
    });
    vi.stubGlobal("window", { location: { hostname: "localhost" }, isSecureContext: false });
    vi.stubGlobal("GeolocationPositionError", TestGeolocationError);
  });

  afterEach(() => vi.unstubAllGlobals());

  it("persists and restores the last valid start point", () => {
    const geolocation = useRouteGeolocation();
    geolocation.persistStartPoint(48.13, -1.63);
    expect(geolocation.getStoredStartPoint()).toEqual({ lat: 48.13, lng: -1.63 });
    values.set("routes-last-location", "not-json");
    expect(geolocation.getStoredStartPoint()).toBeNull();
  });

  it("resolves the browser position and resets the progress state", async () => {
    const getCurrentPosition = vi.fn((resolve: PositionCallback) => resolve({
      coords: { latitude: 48.13, longitude: -1.63 },
    } as GeolocationPosition));
    vi.stubGlobal("navigator", { geolocation: { getCurrentPosition } });
    const geolocation = useRouteGeolocation();

    await expect(geolocation.locate()).resolves.toEqual({ lat: 48.13, lng: -1.63 });
    expect(geolocation.isLocating.value).toBe(false);
    expect(getCurrentPosition).toHaveBeenCalledWith(expect.any(Function), expect.any(Function), {
      enableHighAccuracy: false,
      timeout: 20_000,
      maximumAge: 600_000,
    });
  });

  it("translates browser errors and enforces a secure non-local context", async () => {
    vi.stubGlobal("navigator", {
      geolocation: {
        getCurrentPosition: (_resolve: PositionCallback, reject: PositionErrorCallback) =>
          reject(new TestGeolocationError(TestGeolocationError.PERMISSION_DENIED, "denied") as GeolocationPositionError),
      },
    });
    await expect(useRouteGeolocation().locate()).rejects.toThrow("permission denied");

    vi.stubGlobal("window", { location: { hostname: "example.test" }, isSecureContext: false });
    await expect(useRouteGeolocation().locate()).rejects.toThrow("geolocation requires HTTPS");
  });
});
