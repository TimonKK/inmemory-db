package compute

type CommandId string

const (
	GetCommandId    CommandId = "GET"
	SetCommandId    CommandId = "SET"
	DeleteCommandId CommandId = "DEL"
)

const (
	GetCommandArgsCount    = 1
	SetCommandArgsCount    = 2
	DeleteCommandArgsCount = 1
)
