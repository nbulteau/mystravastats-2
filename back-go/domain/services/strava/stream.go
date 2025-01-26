package strava

type DistanceStream struct {
	// Define the fields of DistanceStream here
}

type TimeStream struct {
	// Define the fields of TimeStream here
}

type LatLngStream struct {
	// Define the fields of LatLngStream here
}

type CadenceStream struct {
	// Define the fields of CadenceStream here
}

type HeartRateStream struct {
	// Define the fields of HeartRateStream here
}

type MovingStream struct {
	// Define the fields of MovingStream here
}

type AltitudeStream struct {
	Data []float64 `json:"data"`
}

type PowerStream struct {
	// Define the fields of PowerStream here
}

type SmoothVelocityStream struct {
	// Define the fields of SmoothVelocityStream here
}

type SmoothGradeStream struct {
	// Define the fields of SmoothGradeStream here
}

type Stream struct {
	Distance       DistanceStream        `json:"distance"`
	Time           TimeStream            `json:"time"`
	LatLng         *LatLngStream         `json:"latlng,omitempty"`
	Cadence        *CadenceStream        `json:"cadence,omitempty"`
	HeartRate      *HeartRateStream      `json:"heartrate,omitempty"`
	Moving         *MovingStream         `json:"moving,omitempty"`
	Altitude       *AltitudeStream       `json:"altitude,omitempty"`
	Watts          *PowerStream          `json:"watts,omitempty"`
	VelocitySmooth *SmoothVelocityStream `json:"velocity_smooth,omitempty"`
	GradeSmooth    *SmoothGradeStream    `json:"grade_smooth,omitempty"`
}
