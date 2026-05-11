package fleet

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// CommandType is a cloud-to-edge command kind.
type CommandType string

const (
	CommandTypeRestart        CommandType = "restart"
	CommandTypeUpdateConfig   CommandType = "update_config"
	CommandTypeExecuteSkill   CommandType = "execute_skill"
	CommandTypeFirmwareUpdate CommandType = "firmware_update"
)

// CommandStatus tracks command progress through the edge delivery lifecycle.
type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "pending"
	CommandStatusDelivered CommandStatus = "delivered"
	CommandStatusExecuted  CommandStatus = "executed"
	CommandStatusFailed    CommandStatus = "failed"
)

var (
	ErrEmptyNodeID        = errors.New("node_id is required")
	ErrInvalidCommandType = errors.New("invalid command type")
	ErrInvalidStatus      = errors.New("invalid command status")
	ErrCommandNotFound    = errors.New("command not found")
)

// Command is a command queued by the Fleet Manager for an edge node.
type Command struct {
	ID          string         `json:"id"`
	NodeID      string         `json:"node_id"`
	Type        CommandType    `json:"type"`
	Payload     map[string]any `json:"payload,omitempty"`
	Status      CommandStatus  `json:"status"`
	Error       string         `json:"error,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	DeliveredAt *time.Time     `json:"delivered_at,omitempty"`
	ExecutedAt  *time.Time     `json:"executed_at,omitempty"`
}

// Dispatcher stores per-node command queues and command status.
type Dispatcher struct {
	mu       sync.RWMutex
	nextID   uint64
	queues   map[string][]string
	commands map[string]*Command
}

// NewDispatcher creates an in-memory command dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		queues:   make(map[string][]string),
		commands: make(map[string]*Command),
	}
}

// Dispatch queues a command for delivery to a specific node.
func (d *Dispatcher) Dispatch(nodeID string, commandType CommandType, payload map[string]any) (Command, error) {
	if nodeID == "" {
		return Command{}, ErrEmptyNodeID
	}
	if !validCommandType(commandType) {
		return Command{}, ErrInvalidCommandType
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.nextID++
	command := &Command{
		ID:        fmt.Sprintf("cmd-%d", d.nextID),
		NodeID:    nodeID,
		Type:      commandType,
		Payload:   clonePayload(payload),
		Status:    CommandStatusPending,
		CreatedAt: time.Now(),
	}
	d.commands[command.ID] = command
	d.queues[nodeID] = append(d.queues[nodeID], command.ID)

	return cloneCommand(command), nil
}

// DeliverPending marks queued commands as delivered and returns them.
func (d *Dispatcher) DeliverPending(nodeID string) []Command {
	d.mu.Lock()
	defer d.mu.Unlock()

	ids := d.queues[nodeID]
	delete(d.queues, nodeID)

	now := time.Now()
	commands := make([]Command, 0, len(ids))
	for _, id := range ids {
		command, ok := d.commands[id]
		if !ok || command.Status != CommandStatusPending {
			continue
		}
		command.Status = CommandStatusDelivered
		command.DeliveredAt = &now
		commands = append(commands, cloneCommand(command))
	}
	return commands
}

// UpdateStatus records the latest edge-reported command status.
func (d *Dispatcher) UpdateStatus(commandID string, status CommandStatus, message string) error {
	if !validCommandStatus(status) {
		return ErrInvalidStatus
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	command, ok := d.commands[commandID]
	if !ok {
		return ErrCommandNotFound
	}

	command.Status = status
	switch status {
	case CommandStatusDelivered:
		if command.DeliveredAt == nil {
			now := time.Now()
			command.DeliveredAt = &now
		}
	case CommandStatusExecuted:
		now := time.Now()
		command.ExecutedAt = &now
		command.Error = ""
	case CommandStatusFailed:
		now := time.Now()
		command.ExecutedAt = &now
		command.Error = message
	}
	return nil
}

// Command returns a command by ID.
func (d *Dispatcher) Command(commandID string) (Command, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	command, ok := d.commands[commandID]
	if !ok {
		return Command{}, false
	}
	return cloneCommand(command), true
}

func validCommandType(commandType CommandType) bool {
	switch commandType {
	case CommandTypeRestart, CommandTypeUpdateConfig, CommandTypeExecuteSkill, CommandTypeFirmwareUpdate:
		return true
	default:
		return false
	}
}

func validCommandStatus(status CommandStatus) bool {
	switch status {
	case CommandStatusPending, CommandStatusDelivered, CommandStatusExecuted, CommandStatusFailed:
		return true
	default:
		return false
	}
}

func cloneCommand(command *Command) Command {
	copied := *command
	copied.Payload = clonePayload(command.Payload)
	copied.DeliveredAt = cloneTime(command.DeliveredAt)
	copied.ExecutedAt = cloneTime(command.ExecutedAt)
	return copied
}

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	copied := make(map[string]any, len(payload))
	for key, value := range payload {
		copied[key] = value
	}
	return copied
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
