package daytona

import (
	"context"
	"strings"
	"testing"
)

func TestDaytonaCopyDownloadsOneSandboxFile(t *testing.T) {
	original := downloadDaytonaSandboxFile
	t.Cleanup(func() { downloadDaytonaSandboxFile = original })

	var gotID, gotRemote, gotLocal string
	downloadDaytonaSandboxFile = func(_ context.Context, _ *daytonaLeaseBackend, id, remote, local string) error {
		gotID, gotRemote, gotLocal = id, remote, local
		return nil
	}

	backend := &daytonaLeaseBackend{}
	err := backend.Copy(context.Background(), CopyRequest{
		ID:          "cbx_0123456789ab",
		Source:      "SANDBOX:/home/daytona/work/.qa/evidence/offer.png",
		Destination: "/tmp/evidence/offer.png",
	})
	if err != nil {
		t.Fatalf("Copy err=%v", err)
	}
	if gotID != "cbx_0123456789ab" || gotRemote != "/home/daytona/work/.qa/evidence/offer.png" || gotLocal != "/tmp/evidence/offer.png" {
		t.Fatalf("download=(%q, %q, %q)", gotID, gotRemote, gotLocal)
	}
}

func TestDaytonaCopyRejectsUploadAndSymlinkFollowing(t *testing.T) {
	backend := &daytonaLeaseBackend{}

	for name, req := range map[string]CopyRequest{
		"upload": {
			ID:          "cbx_0123456789ab",
			Source:      "/tmp/evidence.png",
			Destination: "SANDBOX:/tmp/evidence.png",
		},
		"follow link": {
			ID:          "cbx_0123456789ab",
			Source:      "SANDBOX:/tmp/evidence.png",
			Destination: "/tmp/evidence.png",
			FollowLink:  true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := backend.Copy(context.Background(), req)
			if err == nil || !strings.Contains(err.Error(), "download") {
				t.Fatalf("Copy err=%v", err)
			}
		})
	}
}
