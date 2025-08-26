export function formatTime(time: number): string {
    const hours = Math.floor((time ?? 0) / 3600);
    const minutes = Math.floor(((time ?? 0) % 3600) / 60);
    const seconds = (time ?? 0) % 60;

    if (hours === 0) {
        return `${minutes}m ${seconds}s`; // Customize the formatting as needed
    }

    return `${hours}h ${minutes}m ${seconds}s`; // Customize the formatting as needed
}

const options: Intl.DateTimeFormatOptions = {
    weekday: "short",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
};

export function formatDate(value: string): string {
    const date = new Date(value);
    return new Intl.DateTimeFormat(navigator.language, options).format(date);
}

export function formatSpeedWithUnit(speed: number, activityType: string): string {
    const formatedSpeed = formatSpeed(speed, activityType);

    if (activityType.endsWith("Run"))  {
        return `${formatedSpeed}/km`;
    } else {
        return `${formatedSpeed} km/h`;
    }
}

export function formatSpeed(speed: number, activityType: string): string {
    if (activityType.endsWith("Run"))  {
      return `${formatSeconds(1000 / speed)}`;
    } else {
      return `${(speed * 3.6).toFixed(2)}`;
    }
  }

/**
 * Format seconds to minutes and seconds
 */
function formatSeconds(seconds: number): string {
    let min = Math.floor((seconds % 3600) / 60);
    let sec = Math.floor(seconds % 60);
    const hnd = Math.floor((seconds - min * 60 - sec) * 100 + 0.5);

    if (hnd === 100) {
        sec++;
        if (sec === 60) {
            sec = 0;
            min++;
        }
    }

    return `${min}'${sec < 10 ? "0" : ""}${sec}`;
}