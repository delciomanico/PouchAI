package heartbeats

import (
	"testing"
	"time"
)

func TestDeadReportsOnlyStaleWorkers(t *testing.T) {
	tr := NewTracker()
	tr.Touch("fresh")
	tr.Touch("stale")

	time.Sleep(30 * time.Millisecond)
	tr.Touch("fresh") // renova só este

	dead := tr.Dead(20 * time.Millisecond)
	if len(dead) != 1 || dead[0] != "stale" {
		t.Fatalf("Dead = %v, want [stale]", dead)
	}
}

func TestDeadForgetsReportedWorkers(t *testing.T) {
	tr := NewTracker()
	tr.Touch("worker-1")

	time.Sleep(20 * time.Millisecond)

	first := tr.Dead(10 * time.Millisecond)
	if len(first) != 1 || first[0] != "worker-1" {
		t.Fatalf("first Dead() = %v, want [worker-1]", first)
	}

	second := tr.Dead(10 * time.Millisecond)
	if len(second) != 0 {
		t.Fatalf("second Dead() = %v, want [] (already reported once)", second)
	}
}

func TestDeadWithNoWorkers(t *testing.T) {
	tr := NewTracker()
	if dead := tr.Dead(time.Second); len(dead) != 0 {
		t.Fatalf("Dead() on empty tracker = %v, want []", dead)
	}
}
