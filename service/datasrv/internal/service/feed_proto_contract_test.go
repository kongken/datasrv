package service

import (
	"testing"

	feedsv1 "github.com/kongken/datasrv/pkg/proto/feeds/v1"
)

func TestFeedProtoContractsExposeAdminAndQueryMessages(t *testing.T) {
	adminReq := &feedsv1.CreateFeedSourceRequest{
		Source: &feedsv1.FeedSource{
			Url:         "https://example.com/feed.xml",
			DisplayName: "Example Feed",
			Enabled:     true,
		},
	}
	if adminReq.GetSource().GetUrl() == "" {
		t.Fatal("expected feed source url in admin request")
	}

	queryReq := &feedsv1.ListFeedContentsRequest{
		FeedSourceId: "feed-1",
		Page:         1,
		PageSize:     20,
	}
	if queryReq.GetFeedSourceId() == "" {
		t.Fatal("expected feed source id in query request")
	}
}
