package service

import (
	"context"
	"testing"
	"time"

	issuesv1 "github.com/kongken/datasrv/pkg/proto/issues/v1"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

func TestIssueQueryGRPCServer_ListIssues(t *testing.T) {
	store := newFakeSyncStore()
	now := time.Now().UTC()
	_, _ = store.UpsertIssues(context.Background(), "o/r", []dao.SyncedIssue{
		{Repo: "o/r", IssueID: 1, Number: 1, Title: "one", State: "open", Author: "alice", UpdatedAt: now, AISummary: "one summary"},
		{Repo: "o/r", IssueID: 2, Number: 2, Title: "two", State: "closed", Author: "bob", UpdatedAt: now},
		{Repo: "o/r", IssueID: 3, Number: 3, Title: "three", State: "open", Author: "carol", UpdatedAt: now},
	})

	srv := NewIssueQueryGRPCServer(store, nil)
	resp, err := srv.ListIssues(context.Background(), &issuesv1.ListIssuesRequest{Repo: "o/r", State: "open", Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}
	if len(resp.Issues) != 1 {
		t.Fatalf("issues len = %d, want 1", len(resp.Issues))
	}
	if !resp.HasNext {
		t.Fatalf("has_next = false, want true")
	}
	if resp.Issues[0].State != "open" {
		t.Fatalf("state = %q, want open", resp.Issues[0].State)
	}
	if resp.Issues[0].GetAiSummary() != "one summary" {
		t.Fatalf("ai_summary = %q, want one summary", resp.Issues[0].GetAiSummary())
	}
}

func TestIssueQueryGRPCServer_GetIssue(t *testing.T) {
	store := newFakeSyncStore()
	now := time.Now().UTC()
	_, _ = store.UpsertIssues(context.Background(), "o/r", []dao.SyncedIssue{
		{Repo: "o/r", IssueID: 10, Number: 100, Title: "hello", State: "open", Author: "alice", UpdatedAt: now, AISummary: "short summary"},
	})

	srv := NewIssueQueryGRPCServer(store, nil)
	resp, err := srv.GetIssue(context.Background(), &issuesv1.GetIssueRequest{Repo: "o/r", Selector: &issuesv1.GetIssueRequest_Number{Number: 100}})
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}
	if resp.GetIssue().GetTitle() != "hello" {
		t.Fatalf("title = %q, want hello", resp.GetIssue().GetTitle())
	}
	if resp.GetIssue().GetAiSummary() != "short summary" {
		t.Fatalf("ai_summary = %q, want short summary", resp.GetIssue().GetAiSummary())
	}
}

func TestIssueQueryGRPCServer_GetIssueValidation(t *testing.T) {
	srv := NewIssueQueryGRPCServer(newFakeSyncStore(), nil)
	if _, err := srv.GetIssue(context.Background(), &issuesv1.GetIssueRequest{Repo: "o/r"}); err == nil {
		t.Fatalf("GetIssue() should fail when selector missing")
	}
	if _, err := srv.ListIssues(context.Background(), &issuesv1.ListIssuesRequest{}); err == nil {
		t.Fatalf("ListIssues() should fail when repo missing")
	}
}

func TestIssueQueryGRPCServer_GetIssueLoadsComments(t *testing.T) {
	store := newFakeSyncStore()
	commentStore := newFakeIssueCommentStore()
	now := time.Now().UTC()
	_, _ = store.UpsertIssues(context.Background(), "o/r", []dao.SyncedIssue{
		{
			Repo:      "o/r",
			IssueID:   10,
			Number:    100,
			Title:     "hello",
			State:     "open",
			Author:    "alice",
			UpdatedAt: now,
			Comments:  1,
		},
	})
	commentStore.saved["o/r/10-100.json"] = []dao.IssueComment{{
		ID:        1,
		Body:      "reply",
		UserLogin: "bob",
		CreatedAt: now,
	}}

	srv := NewIssueQueryGRPCServer(store, commentStore)
	resp, err := srv.GetIssue(context.Background(), &issuesv1.GetIssueRequest{Repo: "o/r", Selector: &issuesv1.GetIssueRequest_Number{Number: 100}})
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}
	if len(resp.GetIssue().GetCommentsDetail()) != 1 {
		t.Fatalf("comments_detail len = %d, want 1", len(resp.GetIssue().GetCommentsDetail()))
	}
	if resp.GetIssue().GetCommentsDetail()[0].GetBody() != "reply" {
		t.Fatalf("comment body = %q, want reply", resp.GetIssue().GetCommentsDetail()[0].GetBody())
	}
}
