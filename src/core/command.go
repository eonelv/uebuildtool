package core

type Command struct {
	Cmd       uint16
	Message   interface{}
	RetChan   chan *Command
	OtherInfo interface{}
}

type PackHeader struct {
	Tag     uint16 "TAG"
	Version uint16 "VERSION"
	Length  uint16
	Cmd     uint16
}
