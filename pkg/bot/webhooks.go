package dbot

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

const DbotHook = "dbot/hook"

func (b *DBot) GetWebHook(ctx context.Context, chID, name, avatarURL string) (*discordgo.Webhook, error) {
	hooks, err := b.ChannelWebhooks(chID)
	if err != nil {
		return nil, err
	}

	for _, hook := range hooks {
		if hook.Name == name {
			return hook, nil
		}
	}

	if avatarURL == "" {
		return b.WebhookCreate(chID, name, "")
	}

	avatar, err := imageData(avatarURL, "png")
	if err != nil {
		return nil, err
	}
	return b.WebhookCreate(chID, name, avatar)
}

func imageData(url, imageFormat string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	encoded := &bytes.Buffer{}
	n := int(resp.ContentLength)
	encoded.Grow((n + 2 - ((n + 2) % 3)) / 3 * 4)

	enc := base64.NewEncoder(base64.StdEncoding, encoded)
	_, err = enc.Write(data)
	if err != nil {
		return "", err
	}
	enc.Close()
	return fmt.Sprintf("data:image/%s;base64,%s", imageFormat, encoded.String()), nil
}
