package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
	"github.com/kongken/datasrv/service/datasrv/internal/dao"
)

type IssueCommentStore interface {
	SaveComments(ctx context.Context, repo string, issueID int64, issueNumber int32, comments []dao.IssueComment) error
	LoadComments(ctx context.Context, repo string, issueID int64, issueNumber int32) ([]dao.IssueComment, error)
}

type S3IssueCommentStore struct {
	client    *s3.Client
	bucket    string
	keyPrefix string
}

func NewIssueCommentStore(cfg conf.IssueCommentStorageConfig) (IssueCommentStore, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "s3":
		return newS3IssueCommentStore(cfg)
	default:
		return nil, fmt.Errorf("unsupported issue comment storage provider %q", cfg.Provider)
	}
}

func newS3IssueCommentStore(cfg conf.IssueCommentStorageConfig) (*S3IssueCommentStore, error) {
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, fmt.Errorf("issue comment storage bucket is required")
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "us-east-1"
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	if strings.TrimSpace(cfg.AccessKeyID) != "" || strings.TrimSpace(cfg.SecretAccessKey) != "" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	return &S3IssueCommentStore{
		client:    client,
		bucket:    cfg.Bucket,
		keyPrefix: strings.Trim(strings.TrimSpace(cfg.KeyPrefix), "/"),
	}, nil
}

func (s *S3IssueCommentStore) SaveComments(ctx context.Context, repo string, issueID int64, issueNumber int32, comments []dao.IssueComment) error {
	body, err := json.Marshal(comments)
	if err != nil {
		return fmt.Errorf("marshal issue comments: %w", err)
	}

	key := s.objectKey(repo, issueID, issueNumber)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put comments object: %w", err)
	}
	return nil
}

func (s *S3IssueCommentStore) LoadComments(ctx context.Context, repo string, issueID int64, issueNumber int32) ([]dao.IssueComment, error) {
	key := s.objectKey(repo, issueID, issueNumber)
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get comments object: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read comments object: %w", err)
	}

	var comments []dao.IssueComment
	if err := json.Unmarshal(body, &comments); err != nil {
		return nil, fmt.Errorf("unmarshal comments object: %w", err)
	}
	return comments, nil
}

func (s *S3IssueCommentStore) objectKey(repo string, issueID int64, issueNumber int32) string {
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	key := fmt.Sprintf("github-issue-comments/%s/%d-%d.json", repo, issueID, issueNumber)
	if s.keyPrefix == "" {
		return key
	}
	return s.keyPrefix + "/" + key
}
