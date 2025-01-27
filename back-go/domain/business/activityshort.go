package business

type ActivityShort struct {
	Id   int64
	Name string
	Type ActivityType
}

func NewActivityShort(id int64, name string, activityType string) ActivityShort {
	return ActivityShort{
		Id:   id,
		Name: name,
		Type: ActivityTypes[activityType],
	}
}
