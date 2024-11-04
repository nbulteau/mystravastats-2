export function calculateTrendLine(data: number[]): number[] {
    const n = data.length;
    const xSum = data.reduce((sum, _, index) => sum + index, 0);
    const ySum = data.reduce((sum, value) => sum + value, 0);
    const xySum = data.reduce((sum, value, index) => sum + index * value, 0);
    const xSquaredSum = data.reduce((sum, _, index) => sum + index * index, 0);
  
    const slope = (n * xySum - xSum * ySum) / (n * xSquaredSum - xSum * xSum);
    const intercept = (ySum - slope * xSum) / n;
  
    return data.map((_, index) => slope * index + intercept);
  }

    export function calculateAverageLine(data: number[]): number[] {
        const average = data.reduce((sum, value) => sum + value, 0) / data.length;

        return Array(data.length).fill(average);
    }