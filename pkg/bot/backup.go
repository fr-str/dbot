package dbot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"dbot/pkg/backup"
	"dbot/pkg/config"

	"github.com/fr-str/log"

	"github.com/bwmarrin/discordgo"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func backupMessage(d *DBot, m *discordgo.Message) error {
	if m.Author.Bot {
		return nil
	}

	var err error
	attachments, err := backupAttachment(d, m)
	if err != nil {
		return err
	}

	_, err = d.Backup.UpsertUser(d.Ctx, backup.UpsertUserParams{
		Username:  m.Author.Username,
		DiscordID: must(strconv.ParseInt(m.Author.ID, 10, 64)),
	})
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}

	err = d.Backup.InsertBackup(d.Ctx, backup.InsertBackupParams{
		MsgID:       must(strconv.ParseInt(m.ID, 10, 64)),
		AuthorID:    must(strconv.ParseInt(m.Author.ID, 10, 64)),
		ChannelID:   must(strconv.ParseInt(m.ChannelID, 10, 64)),
		Content:     m.Content,
		Attachments: attachments,
		CreatedAt:   time.Now(),
	})
	if err != nil {
		return fmt.Errorf("insert backup: %w", err)
	}

	return nil
}

func updateBackupMessage(d *DBot, m *discordgo.Message) error {
	err := d.Backup.UpdateBackupMsg(d.Ctx, backup.UpdateBackupMsgParams{
		MsgID:   must(strconv.ParseInt(m.ID, 10, 64)),
		Content: m.Content,
	})
	if err != nil {
		return fmt.Errorf("updateBackupMessage: %w", err)
	}

	return nil
}

func backupAttachment(d *DBot, m *discordgo.Message) (string, error) {
	if len(m.Attachments) == 0 {
		return "", nil
	}

	att := make([]string, len(m.Attachments))
	for i, a := range m.Attachments {
		resp, err := http.Get(a.URL)
		if err != nil {
			return "", fmt.Errorf("failed to get '%s': %w", a.URL, err)
		}
		defer resp.Body.Close()

		info, err := d.backupFile(a.Filename, m.GuildID, resp.Body)
		if err != nil {
			return "", fmt.Errorf("store attachment '%s': %w", a.URL, err)
		}

		err = d.Backup.InsertArtefact(d.Ctx, backup.InsertArtefactParams{
			Path:      info.Name,
			MediaType: a.ContentType,
			Hash:      info.Name,
			CreatedAt: time.Now(),
		})
		if err != nil {
			return "", fmt.Errorf("insert artefact: %w", err)
		}

		att[i] = info.Name
	}

	b, err := json.Marshal(att)
	if err != nil {
		return "", fmt.Errorf("encode attachments: %w", err)
	}

	return string(b), nil
}

type backupFile struct {
	Name string
	Size int64
}

func (d *DBot) backupFile(name string, gID string, file io.Reader) (backupFile, error) {
	var backupFile backupFile
	if err := os.MkdirAll(filepath.Join(config.BACKUP_DIR, gID), 0o755); err != nil {
		return backupFile, fmt.Errorf("failed to create dir: %w", err)
	}

	f, err := os.Create(filepath.Join(config.BACKUP_DIR, gID, fmt.Sprintf("%d-%s", time.Now().Unix(), filepath.Base(name))))
	if err != nil {
		return backupFile, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, file)
	if err != nil {
		return backupFile, fmt.Errorf("failed to copy file: %w", err)
	}

	log.Trace("backupFile", log.String("name", name), log.Int("size", n))
	backupFile.Name = f.Name()
	backupFile.Size = n

	return backupFile, nil
}
