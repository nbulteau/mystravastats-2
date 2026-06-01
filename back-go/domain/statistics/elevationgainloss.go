package statistics

import "math"

type elevationGainLoss struct {
	gain float64
	loss float64
}

type elevationGainLossPrefix struct {
	gains  []float64
	losses []float64
}

func newElevationGainLossPrefix(altitudes []float64, dataSize int) elevationGainLossPrefix {
	if dataSize > len(altitudes) {
		dataSize = len(altitudes)
	}
	if dataSize <= 0 {
		return elevationGainLossPrefix{}
	}

	gains := make([]float64, dataSize)
	losses := make([]float64, dataSize)
	for i := 1; i < dataSize; i++ {
		gains[i] = gains[i-1]
		losses[i] = losses[i-1]

		if !isFiniteAltitude(altitudes[i]) || !isFiniteAltitude(altitudes[i-1]) {
			continue
		}

		delta := altitudes[i] - altitudes[i-1]
		if delta > 0 {
			gains[i] += delta
		} else if delta < 0 {
			losses[i] += math.Abs(delta)
		}
	}

	return elevationGainLossPrefix{gains: gains, losses: losses}
}

func (prefix elevationGainLossPrefix) between(idxStart, idxEnd int) elevationGainLoss {
	if len(prefix.gains) == 0 || len(prefix.losses) == 0 {
		return elevationGainLoss{}
	}
	if idxStart < 0 {
		idxStart = 0
	}
	if idxEnd < 0 {
		idxEnd = 0
	}
	if idxStart >= len(prefix.gains) {
		idxStart = len(prefix.gains) - 1
	}
	if idxEnd >= len(prefix.gains) {
		idxEnd = len(prefix.gains) - 1
	}
	if idxEnd < idxStart {
		idxEnd = idxStart
	}

	return elevationGainLoss{
		gain: prefix.gains[idxEnd] - prefix.gains[idxStart],
		loss: prefix.losses[idxEnd] - prefix.losses[idxStart],
	}
}

func (prefix elevationGainLossPrefix) betweenPtrs(idxStart, idxEnd int) (*float64, *float64) {
	if len(prefix.gains) == 0 || len(prefix.losses) == 0 {
		return nil, nil
	}

	elevation := prefix.between(idxStart, idxEnd)
	return float64Ptr(elevation.gain), float64Ptr(elevation.loss)
}

func float64Ptr(value float64) *float64 {
	copy := value
	return &copy
}

func isFiniteAltitude(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
