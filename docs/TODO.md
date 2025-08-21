# Backlog

## 1. **Heart Rate Analysis**

- **Heart Rate Zones**:
  - **How to Calculate**: Define heart rate zones (e.g., warm-up, fat burn, cardio, peak) and calculate the time spent in each zone.
  - **Implementation**: Iterate through heart rate readings and count the duration spent in each zone.

     ```kotlin
     val zones = mapOf("Warm-up" to 100..120, "Fat Burn" to 121..140, "Cardio" to 141..160, "Peak" to 161..180)
     val timeInZones = mutableMapOf<String, Int>()
     heartRateReadings.forEach { hr ->
         zones.forEach { (zone, range) ->
             if (hr in range) {
                 timeInZones[zone] = timeInZones.getOrDefault(zone, 0) + 1
             }
         }
     }
     ```

### 2. **Advanced Performance Metrics**

- **Running Power**:
  - **How to Calculate**: Use data from compatible devices that measure running power.
  - **Implementation**: Fetch running power data from the device's API or file format.
- **Cycling Efficiency**:
  - **How to Calculate**: Calculate power-to-weight ratio (PWR) using power output and the athlete's weight.
  - **Implementation**: Fetch power data and athlete's weight, then perform the calculation.

     ```kotlin
     val pwr = powerOutput / athleteWeight
     ```

### 3. **Route Analysis**

- **Popular Routes**:
  - **How to Implement**: Identify frequently used routes by analyzing GPS data.
  - **Implementation**: Cluster GPS coordinates to identify popular routes.
- **Route Difficulty**:
  - **How to Calculate**: Analyze elevation gain, distance, and other factors to determine difficulty.
  - **Implementation**: Use algorithms to calculate difficulty based on route characteristics.

     ```kotlin
     val difficulty = calculateRouteDifficulty(elevationGain, distance)
     ```

### 4. **Calorie Burn**

- **Calorie Burn**:
  - **How to Calculate**: Estimate calorie burn using activity type, duration, and intensity.
  - **Implementation**: Use formulas or APIs to calculate calorie burn.

     ```kotlin
     val caloriesBurned = calculateCaloriesBurned(activityType, duration, intensity)
     ```
