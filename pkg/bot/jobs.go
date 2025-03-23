package dbot

import (
	"encoding/json"
	"fmt"

	"dbot/pkg/dbg"
	"dbot/pkg/store"
)

const (
	DownloadJob = "download"
)

type DownloadAsyncMeta struct {
	PlaylistID  int64
	URL         string
	GID         string
	Name        string
	DownloadFor string
}

func (d *DBot) downloadAsync(meta string) error {
	var dwMeta DownloadAsyncMeta
	err := json.Unmarshal([]byte(meta), &dwMeta)
	if err != nil {
		return fmt.Errorf("downloadAsync: %w", err)
	}

	dbg.Assert(len(dwMeta.GID) > 0)
	dbg.Assert(len(dwMeta.URL) > 0)
	dbg.Assert(len(dwMeta.Name) > 0)

	f, err := d.downloadAsMP4(d.Ctx, dwMeta.URL)
	if err != nil {
		return fmt.Errorf("downloadAsync: %w", err)
	}
	defer f.Close()

	bf, err := d.backupFile(backupFileParams{
		Name: dwMeta.Name,
		GID:  dwMeta.GID,
		Dirs: dwMeta.DownloadFor,
		File: f.body,
	})
	if err != nil {
		return fmt.Errorf("downloadAsync: %w", err)
	}

	_, err = d.Store.AddPlaylistEntry(d.Ctx, store.AddPlaylistEntryParams{
		PlaylistID: dwMeta.PlaylistID,
		YoutubeUrl: dwMeta.URL,
		Filepath:   bf.Name,
		Name:       dwMeta.Name,
	})
	if err != nil {
		return fmt.Errorf("downloadAsync: %w", err)
	}
	return nil
}
