package daytona

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var downloadDaytonaSandboxFile = func(ctx context.Context, backend *daytonaLeaseBackend, id, remotePath, localPath string) error {
	toolboxClient, err := newDaytonaToolboxClient(backend.cfg)
	if err != nil {
		return err
	}
	apiClient, err := newDaytonaClient(backend.cfg, backend.rt)
	if err != nil {
		return err
	}
	apiSandbox, leaseID, err := resolveDaytonaSandbox(ctx, apiClient, backend.cfg, id)
	if err != nil {
		return err
	}
	if err := requireExactDaytonaClaim(leaseID, apiSandbox); err != nil {
		return err
	}
	if !daytonaStateReady(daytonaSandboxState(apiSandbox)) {
		if _, err := apiClient.StartSandbox(ctx, apiSandbox.GetId()); err != nil {
			return daytonaError("start sandbox", err)
		}
		if _, err := waitForDaytonaReady(ctx, apiClient, apiSandbox.GetId(), 5*time.Minute); err != nil {
			return err
		}
	}
	sandbox, err := toolboxClient.Get(ctx, apiSandbox.GetId())
	if err != nil {
		return daytonaError("get sandbox", err)
	}
	stream, err := sandbox.FileSystem.DownloadFileStream(ctx, remotePath)
	if err != nil {
		return daytonaError("download file", err)
	}
	defer stream.Close()

	if info, statErr := os.Stat(localPath); statErr == nil && info.IsDir() {
		localPath = filepath.Join(localPath, filepath.Base(remotePath))
	}
	parent := filepath.Dir(localPath)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create download directory: %w", err)
	}
	temp, err := os.CreateTemp(parent, ".crabbox-download-*")
	if err != nil {
		return fmt.Errorf("create temporary download: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return fmt.Errorf("secure temporary download: %w", err)
	}
	if _, err := io.Copy(temp, stream); err != nil {
		temp.Close()
		return fmt.Errorf("download file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close downloaded file: %w", err)
	}
	if err := os.Rename(tempPath, localPath); err != nil {
		return fmt.Errorf("save downloaded file: %w", err)
	}
	return nil
}

func (b *daytonaLeaseBackend) Copy(ctx context.Context, req CopyRequest) error {
	if req.FollowLink {
		return exit(2, "daytona native copy supports file download only; -L is not supported")
	}
	sourcePrefix, remotePath, sourceHasSeparator := strings.Cut(req.Source, ":")
	if !sourceHasSeparator || !strings.EqualFold(strings.TrimSpace(sourcePrefix), "SANDBOX") || strings.TrimSpace(remotePath) == "" {
		return exit(2, "daytona native copy supports file download only; source must use SANDBOX:PATH")
	}
	if destinationPrefix, _, ok := strings.Cut(req.Destination, ":"); ok && strings.EqualFold(strings.TrimSpace(destinationPrefix), "SANDBOX") {
		return exit(2, "daytona native copy supports file download only")
	}
	return downloadDaytonaSandboxFile(ctx, b, req.ID, remotePath, req.Destination)
}
