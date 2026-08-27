import { ref } from "vue";

type StoredStartPoint = { lat: number; lng: number };

const storageKey = "routes-last-location";

export function useRouteGeolocation() {
  const isLocating = ref(false);

  function getStoredStartPoint(): StoredStartPoint | null {
    try {
      const raw = localStorage.getItem(storageKey);
      if (!raw) return null;
      const parsed = JSON.parse(raw) as Partial<StoredStartPoint>;
      if (typeof parsed.lat !== "number" || typeof parsed.lng !== "number") return null;
      return { lat: parsed.lat, lng: parsed.lng };
    } catch {
      return null;
    }
  }

  function persistStartPoint(lat: number, lng: number): void {
    try {
      localStorage.setItem(storageKey, JSON.stringify({ lat, lng }));
    } catch {
      // Persistence is a best-effort convenience.
    }
  }

  async function locate(): Promise<StoredStartPoint> {
    if (isLocating.value) throw new Error("location request already in progress");
    if (!navigator.geolocation) throw new Error("geolocation is not available in this browser");
    const host = window.location.hostname;
    const isLocalhost = host === "localhost" || host === "127.0.0.1" || host === "::1";
    if (!window.isSecureContext && !isLocalhost) throw new Error("geolocation requires HTTPS outside localhost");

    isLocating.value = true;
    try {
      const position = await new Promise<GeolocationPosition>((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(resolve, reject, {
          enableHighAccuracy: false,
          timeout: 20_000,
          maximumAge: 10 * 60 * 1000,
        });
      });
      return { lat: position.coords.latitude, lng: position.coords.longitude };
    } catch (error) {
      if (!(error instanceof GeolocationPositionError)) throw error;
      if (error.code === error.PERMISSION_DENIED) throw new Error("permission denied", { cause: error });
      if (error.code === error.POSITION_UNAVAILABLE) throw new Error("position unavailable", { cause: error });
      if (error.code === error.TIMEOUT) throw new Error("timeout", { cause: error });
      throw new Error(error.message || "unknown error", { cause: error });
    } finally {
      isLocating.value = false;
    }
  }

  return { isLocating, locate, getStoredStartPoint, persistStartPoint };
}
