package strava

type Athlete struct {
	AthleteType           *int     `json:"athlete_type,omitempty"`
	BadgeTypeId           *int     `json:"badge_type_id,omitempty"`
	Bikes                 []Bike   `json:"bikes,omitempty"`
	City                  *string  `json:"city,omitempty"`
	Clubs                 []any    `json:"clubs,omitempty"`
	Country               *string  `json:"country,omitempty"`
	CreatedAt             *string  `json:"created_at,omitempty"`
	DatePreference        *string  `json:"date_preference,omitempty"`
	Firstname             *string  `json:"firstname,omitempty"`
	Follower              *any     `json:"follower,omitempty"`
	FollowerCount         *int     `json:"follower_count,omitempty"`
	Friend                *any     `json:"friend,omitempty"`
	FriendCount           *int     `json:"friend_count,omitempty"`
	Ftp                   *any     `json:"ftp,omitempty"`
	Id                    int64    `json:"id"`
	Lastname              *string  `json:"lastname,omitempty"`
	MeasurementPreference *string  `json:"measurement_preference,omitempty"`
	MutualFriendCount     *int     `json:"mutual_friend_count,omitempty"`
	Premium               *bool    `json:"premium,omitempty"`
	Profile               *string  `json:"profile,omitempty"`
	ProfileMedium         *string  `json:"profile_medium,omitempty"`
	ResourceState         *int     `json:"resource_state,omitempty"`
	Sex                   *string  `json:"sex,omitempty"`
	Shoes                 []Shoe   `json:"shoes,omitempty"`
	State                 *string  `json:"state,omitempty"`
	Summit                *bool    `json:"summit,omitempty"`
	UpdatedAt             *string  `json:"updated_at,omitempty"`
	Username              *string  `json:"username,omitempty"`
	Weight                *float64 `json:"weight,omitempty"`
}
