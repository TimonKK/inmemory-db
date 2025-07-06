package compute

type CommandId string

const (
	GetCommandId         CommandId = "GET"
	SetCommandId         CommandId = "SET"
	DeleteCommandId      CommandId = "DEL"
	ReplicationCommandId CommandId = "REPLICATION"
)

const (
	GetCommandArgsCount         = 1
	SetCommandArgsCount         = 2
	DeleteCommandArgsCount      = 1
	ReplicationCommandArgsCount = 1
)
