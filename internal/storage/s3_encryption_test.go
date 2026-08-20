package storage

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestNormalizeServerSideEncryption(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		kmsKey  string
		want    string
		wantErr bool
	}{
		{name: "disabled", want: ""},
		{name: "s3 managed", mode: "AES256", want: "AES256"},
		{name: "KMS", mode: "aws:kms", kmsKey: "key-id", want: "aws:kms"},
		{name: "invalid mode", mode: "wrong", wantErr: true},
		{name: "KMS key without KMS mode", kmsKey: "key-id", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeServerSideEncryption(tt.mode, tt.kmsKey)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected configuration error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestPutObjectInputIncludesServerSideEncryption(t *testing.T) {
	client := &S3Client{
		bucket:               "recordings",
		serverSideEncryption: "aws:kms",
		kmsKeyID:             "arn:aws:kms:ap-south-1:123:key/example",
	}

	input := client.putObjectInput("recordings/org/call.ogg", strings.NewReader("audio"), "audio/ogg")
	if input.ServerSideEncryption != types.ServerSideEncryptionAwsKms {
		t.Fatalf("expected aws:kms encryption, got %q", input.ServerSideEncryption)
	}
	if aws.ToString(input.SSEKMSKeyId) != client.kmsKeyID {
		t.Fatalf("expected KMS key ID to be forwarded")
	}
}
