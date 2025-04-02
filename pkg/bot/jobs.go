package dbot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"dbot/pkg/config"
	"dbot/pkg/dbg"
	"dbot/pkg/store"
)

const (
	DownloadJob = "download"
	BackupJob   = "backup"
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
	defer f.File.Close()

	bf, err := d.backupFile(BackupFileParams{
		Name: dwMeta.Name,
		GID:  dwMeta.GID,
		Dirs: dwMeta.DownloadFor,
		File: f.File,
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

func (d *DBot) backupJob(meta string) error {
	var bf BackupFileParams
	err := json.Unmarshal([]byte(meta), &bf)
	if err != nil {
		return fmt.Errorf("backupFile: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := filepath.Join(config.TMP_PATH, "backup")
	os.MkdirAll(dir, 0o755)
	ctx = context.WithValue(ctx, config.DirKey, dir)

	f, err := d.downloadAsMP4(ctx, bf.OriginUrl)
	if err != nil {
		return fmt.Errorf("backupFile: %w", err)
	}
	defer f.File.Close()
	bf.File = f.File
	bf.Name = f.Name

	_, err = d.backupArtefact(ctx, bf)
	if err != nil {
		return fmt.Errorf("backupFile: %w", err)
	}

	return nil
}
