package dbot

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"dbot/pkg/store"
)

const (
	DownloadJob = "download"
)

type DownloadAsyncMeta struct {
	PlaylistID int64
	URL        string
	GID        string
	Name       string
}

func (d *DBot) downloadAsync(meta string) error {
	var dwMeta DownloadAsyncMeta
	err := json.Unmarshal([]byte(meta), &dwMeta)
	if err != nil {
		return fmt.Errorf("downloadAsync: %w", err)
	}

	info, err := d.storeMediaInMinIO(dwMeta.Name, dwMeta.URL, dwMeta.GID)
	if err != nil {
		return fmt.Errorf("downloadAsync: %w", err)
	}

	_, err = d.Store.AddPlaylistEntry(d.Ctx, store.AddPlaylistEntryParams{
		PlaylistID: dwMeta.PlaylistID,
		YoutubeUrl: dwMeta.URL,
		MinioUrl:   linkFromMinioUploadInfo(filepath.Join(dwMeta.GID, "videos", info.Key)),
		Name:       dwMeta.Name,
	})
	if err != nil {
		return fmt.Errorf("downloadAsync: %w", err)
	}
	return nil
}
