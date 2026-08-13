package domain

import (
	"testing"
	"time"
)

func TestTableNames(t *testing.T) {
	if (Job{}).TableName() != "jobs" {
		t.Fatal("job table name")
	}
	if (DeadLetter{}).TableName() != "dead_letters" {
		t.Fatal("dead letter table name")
	}
	if (JobHistory{}).TableName() != "job_history" {
		t.Fatal("history table name")
	}
}

func TestJobStatusConstants(t *testing.T) {
	expected := []string{JobStatusPending, JobStatusRunning, JobStatusSuccess, JobStatusFailed, JobStatusDead}
	want := []string{"pending", "running", "success", "failed", "dead"}
	for i, e := range expected {
		if e != want[i] {
			t.Fatalf("status[%d] = %q, want %q", i, e, want[i])
		}
	}
}

func TestEntityValues(t *testing.T) {
	now := time.Now()
	j := Job{ID: 1, Type: "email", Status: JobStatusPending, Priority: 5, NextRunAt: now}
	if j.ID != 1 || j.Type != "email" || j.Priority != 5 || !j.NextRunAt.Equal(now) {
		t.Fatal("job value mismatch")
	}
	d := DeadLetter{ID: 2, JobID: 1, Resolved: false, FailedAt: now}
	if d.ID != 2 || d.JobID != 1 || d.Resolved {
		t.Fatal("dead letter value mismatch")
	}
	h := JobHistory{ID: 3, JobID: 1, Status: "success", Duration: 100}
	if h.ID != 3 || h.Duration != 100 || h.Status != "success" {
		t.Fatal("history value mismatch")
	}
}
