package domain

import "time"

// Pagination represents common pagination parameters
type Pagination struct {
	Page     int
	PageSize int
	Total    int
}

// DateRange represents a date range
type DateRange struct {
	Start time.Time
	End   time.Time
}

// Result represents a successful result with metadata
type Result struct {
	Data      interface{}
	Metadata  map[string]interface{}
	Timestamp time.Time
}

// NewResult creates a new result
func NewResult(data interface{}) *Result {
	return &Result{
		Data:      data,
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now().UTC(),
	}
}

// WithMetadata adds metadata to result
func (r *Result) WithMetadata(key string, value interface{}) *Result {
	r.Metadata[key] = value
	return r
}

// ActivityStatus represents the status of an activity
type ActivityStatus string

const (
	ActivityStatusActive   ActivityStatus = "ACTIVE"
	ActivityStatusArchived ActivityStatus = "ARCHIVED"
	ActivityStatusDeleted  ActivityStatus = "DELETED"
)

// ActivityType is the type of activity
type ActivityType string

const (
	ActivityTypeRide        ActivityType = "Ride"
	ActivityTypeRun         ActivityType = "Run"
	ActivityTypeHike        ActivityType = "Hike"
	ActivityTypeWalk        ActivityType = "Walk"
	ActivityTypeSwim        ActivityType = "Swim"
	ActivityTypeVirtualRide ActivityType = "VirtualRide"
	ActivityTypeEBikRide    ActivityType = "EBikRide"
)

// IsValid checks if activity type is valid
func (at ActivityType) IsValid() bool {
	switch at {
	case ActivityTypeRide, ActivityTypeRun, ActivityTypeHike,
		ActivityTypeWalk, ActivityTypeSwim, ActivityTypeVirtualRide,
		ActivityTypeEBikRide:
		return true
	default:
		return false
	}
}

// Period represents a time period
type Period string

const (
	PeriodDay   Period = "DAY"
	PeriodWeek  Period = "WEEK"
	PeriodMonth Period = "MONTH"
	PeriodYear  Period = "YEAR"
)

// IsValid checks if period is valid
func (p Period) IsValid() bool {
	switch p {
	case PeriodDay, PeriodWeek, PeriodMonth, PeriodYear:
		return true
	default:
		return false
	}
}
