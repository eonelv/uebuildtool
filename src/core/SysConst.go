package core

type ObjectID uint64
type NAME_STRING [NAME_LENGTH]byte

const (
	TAG           uint16 = 0
	VERSION       uint16 = 1
	HEADER_LENGTH uint16 = 8
	NAME_LENGTH   uint16 = 255
)

const (
	SYSTEM_CHAN_ID      ObjectID = 100
	SYSTEM_USER_CHAN_ID ObjectID = 101

	SYSTEM_MAJIANG_ROOM_MGR_ID ObjectID = 100000
)
