package dto

import "time"

type AthleteDto struct {
	BadgeTypeId           int       `json:"badge_type_id"`
	City                  string    `json:"city,omitempty"`
	Country               string    `json:"country,omitempty"`
	CreatedAt             time.Time `json:"created_at,omitempty"`
	Firstname             string    `json:"firstname,omitempty"`
	Id                    int64     `json:"id"`
	Lastname              string    `json:"lastname,omitempty"`
	Premium               bool      `json:"premium,omitempty"`
	Profile               string    `json:"profile,omitempty"`
	ProfileMedium         string    `json:"profile_medium,omitempty"`
	ResourceState         int       `json:"resource_state,omitempty"`
	Sex                   string    `json:"sex,omitempty"`
	State                 string    `json:"state,omitempty"`
	Summit                bool      `json:"summit,omitempty"`
	UpdatedAt             time.Time `json:"updated_at,omitempty"`
	Username              string    `json:"username,omitempty"`
	AthleteType           int       `json:"athlete_type,omitempty"`
	DatePreference        string    `json:"date_preference,omitempty"`
	FollowerCount         int       `json:"follower_count,omitempty"`
	FriendCount           int       `json:"friend_count,omitempty"`
	MeasurementPreference string    `json:"measurement_preference,omitempty"`
	MutualFriendCount     int       `json:"mutual_friend_count,omitempty"`
	Weight                int       `json:"weight,omitempty"`
}
