package dbot

import (
	"encoding/json"
	"fmt"

	"dbot/pkg/dbg"
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
	// name := filepath.Join(dwMeta.GID, dwMeta.DownloadFor, dwMeta.Name)

	// TODO: implement
	// info, err := d.storeMediaInMinIOAsMP4(name, dwMeta.URL, dwMeta.GID)
	// if err != nil {
	// 	return fmt.Errorf("downloadAsync: %w", err)
	// }
	//
	// _, err = d.Store.AddPlaylistEntry(d.Ctx, store.AddPlaylistEntryParams{
	// 	PlaylistID: dwMeta.PlaylistID,
	// 	YoutubeUrl: dwMeta.URL,
	// 	MinioUrl:   linkFromMinioUploadInfo(info.Key),
	// 	Name:       dwMeta.Name,
	// })
	// if err != nil {
	// 	return fmt.Errorf("downloadAsync: %w", err)
	// }
	return nil
}
