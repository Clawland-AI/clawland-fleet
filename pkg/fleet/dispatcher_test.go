package fleet

import (
	"errors"
	"testing"
)

func TestDispatcherQueuesCommandsPerNode(t *testing.T) {
	dispatcher := NewDispatcher()

	first, err := dispatcher.Dispatch("node-1", CommandTypeRestart, map[string]any{"reason": "maintenance"})
	if err != nil {
		t.Fatalf("dispatch first command: %v", err)
	}
	second, err := dispatcher.Dispatch("node-2", CommandTypeUpdateConfig, map[string]any{"interval": "30s"})
	if err != nil {
		t.Fatalf("dispatch second command: %v", err)
	}

	if first.ID == "" {
		t.Fatal("expected command ID to be set")
	}
	if first.Status != CommandStatusPending {
		t.Fatalf("expected first command status %q, got %q", CommandStatusPending, first.Status)
	}

	delivered := dispatcher.DeliverPending("node-1")
	if len(delivered) != 1 {
		t.Fatalf("expected 1 command for node-1, got %d", len(delivered))
	}
	if delivered[0].ID != first.ID {
		t.Fatalf("expected delivered command %q, got %q", first.ID, delivered[0].ID)
	}
	if delivered[0].Status != CommandStatusDelivered {
		t.Fatalf("expected delivered status %q, got %q", CommandStatusDelivered, delivered[0].Status)
	}
	if delivered[0].DeliveredAt == nil {
		t.Fatal("expected delivered timestamp to be set")
	}

	if again := dispatcher.DeliverPending("node-1"); len(again) != 0 {
		t.Fatalf("expected node-1 queue to be empty after delivery, got %d", len(again))
	}

	remaining := dispatcher.DeliverPending("node-2")
	if len(remaining) != 1 || remaining[0].ID != second.ID {
		t.Fatalf("expected node-2 command %q to remain queued, got %#v", second.ID, remaining)
	}
}

func TestDispatcherRejectsUnsupportedCommandType(t *testing.T) {
	dispatcher := NewDispatcher()

	_, err := dispatcher.Dispatch("node-1", CommandType("shutdown"), nil)

	if !errors.Is(err, ErrInvalidCommandType) {
		t.Fatalf("expected ErrInvalidCommandType, got %v", err)
	}
}

func TestDispatcherTracksExecutedAndFailedStatuses(t *testing.T) {
	dispatcher := NewDispatcher()
	command, err := dispatcher.Dispatch("node-1", CommandTypeExecuteSkill, map[string]any{"skill": "inspect"})
	if err != nil {
		t.Fatalf("dispatch command: %v", err)
	}

	dispatcher.DeliverPending("node-1")
	if err := dispatcher.UpdateStatus(command.ID, CommandStatusExecuted, ""); err != nil {
		t.Fatalf("update executed status: %v", err)
	}
	executed, ok := dispatcher.Command(command.ID)
	if !ok {
		t.Fatalf("expected command %q to exist", command.ID)
	}
	if executed.Status != CommandStatusExecuted {
		t.Fatalf("expected status %q, got %q", CommandStatusExecuted, executed.Status)
	}
	if executed.ExecutedAt == nil {
		t.Fatal("expected executed timestamp to be set")
	}

	failed, err := dispatcher.Dispatch("node-1", CommandTypeFirmwareUpdate, nil)
	if err != nil {
		t.Fatalf("dispatch failed command: %v", err)
	}
	if err := dispatcher.UpdateStatus(failed.ID, CommandStatusFailed, "firmware checksum mismatch"); err != nil {
		t.Fatalf("update failed status: %v", err)
	}
	got, ok := dispatcher.Command(failed.ID)
	if !ok {
		t.Fatalf("expected failed command %q to exist", failed.ID)
	}
	if got.Status != CommandStatusFailed {
		t.Fatalf("expected status %q, got %q", CommandStatusFailed, got.Status)
	}
	if got.Error != "firmware checksum mismatch" {
		t.Fatalf("expected failure error to be recorded, got %q", got.Error)
	}
}
