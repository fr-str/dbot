package dbot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"dbot/pkg/backup"
	"dbot/pkg/cache"
	"dbot/pkg/config"
	"dbot/pkg/db"
	. "dbot/pkg/dbg"
	"dbot/pkg/ffmpeg"
	jobrunner "dbot/pkg/job_runner"
	"dbot/pkg/player"
	"dbot/pkg/store"
	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
)

type DBot struct {
	// only used to create new Player instances
	_c *cache.Queries

	Ctx    context.Context
	Store  *store.Queries
	Backup *backup.Queries

	*discordgo.Session
	ytdlp.YTDLP
	// MusicPlayer is for plaing stuff from YT and others
	MusicPlayer *player.Player
}

func dbotErr(msg string, vars ...any) error {
	return fmt.Errorf("dbot: "+msg+": ", vars...)
}

var normalizeReplacer = strings.NewReplacer(
	"ą", "a",
	"ć", "c",
	"ę", "e",
	"ł", "l",
	"ż", "z",
	"ź", "z",
	"ó", "o",
	"ś", "s",
	"ń", "n",
	" ", "",
	"\n", "",
)

func normalize(s string) string {
	return normalizeReplacer.Replace(strings.ToLower(s))
}

func startJobRunner(ctx context.Context, db *store.Queries) jobrunner.Runner {
	runner := jobrunner.NewRunner(ctx, db)
	go runner.Loop()
	return runner
}

func Start(ctx context.Context, sess *discordgo.Session, dbstore *store.Queries) *DBot {
	audioCache, err := db.ConnectAudioCache(ctx, "audio-cache.db", "")
	if err != nil {
		panic(err)
	}
	backupDB, err := db.ConnectBackup(ctx, "backup.db", "")
	if err != nil {
		panic(err)
	}

	// sess.LogLevel = 3
	d := DBot{
		_c:          audioCache,
		Ctx:         ctx,
		Session:     sess,
		MusicPlayer: player.NewPlayer(audioCache),
		Store:       dbstore,
		Backup:      backupDB,
	}

	runner := startJobRunner(ctx, dbstore)
	runner.RegisterJob(DownloadJob, d.downloadAsync)
	runner.RegisterJob(BackupJob, d.backupJob)

	// Listeners must be registered befor we open connection
	d.RegisterEventListiners()

	d.StartScheduler()
	d.interfaceLoop()

	err = sess.Open()
	if err != nil {
		panic(err)
	}

	for _, v := range cmds {
		_, err := sess.ApplicationCommandCreate(sess.State.User.ID, config.GUILD_ID, v)
		if err != nil {
			panic(err)
		}
	}

	go func() {
		for Err := range d.MusicPlayer.ErrChan {
			log.Error(Err.Err.Error())
			ch, err := d.Store.GetChannel(ctx, store.GetChannelParams{
				Gid:  Err.GID,
				Type: musicChannel,
			})
			if err != nil {
				log.Error(err.Error())
				continue
			}

			chID := ch.Chid
			errChan, err := d.Store.GetChannel(ctx, store.GetChannelParams{
				Gid:  ch.Gid,
				Type: errorChannel,
			})
			log.Trace("errChan", log.JSON(errChan), log.Err(err))
			if err == nil && errChan.Chid != "" && errChan.Chid != "0" {
				chID = errChan.Chid
			}

			msg := Err.Err.Error()
			switch {
			case errors.Is(Err.Err, ytdlp.ErrFailedToDownload):
				msg = "could not download video"
			case errors.Is(Err.Err, ffmpeg.ErrFfmpegError):
				msg = "failed converting to mp4"
			}

			_, err = d.ChannelMessageSend(chID, msg)
			if err != nil {
				log.Error(err.Error())
				continue
			}

		}
	}()
	return &d
}

func (d *DBot) connectVoice(gID, uID string) error {
	// skip if we are already connected
	if d.MusicPlayer.VC != nil {
		return nil
	}

	channel, err := d.getUserVC(d.Session, gID, uID)
	if err != nil {
		return fmt.Errorf("failed to find User VC: %w", err)
	}

	vc, err := d.ChannelVoiceJoin(gID, channel.ChannelID, false, false)
	if err != nil {
		return fmt.Errorf("failed to join VC: %w", err)
	}

	// vc.LogLevel = 3
	d.MusicPlayer.VC = vc

	return nil
}

const (
	musicChannel = "music"
	errorChannel = "error"
	adminChannel = "admin"
)

type SaveSoundParams struct {
	GID     string
	Aliases string `opt:"aliases"`
	Link    string `opt:"link"`
	// after unmarshal only ID field will be populated
	// if attachment was provided
	Att *discordgo.MessageAttachment `opt:"file"`
}

func (d *DBot) SaveSound(ctx context.Context, params SaveSoundParams) error {
	Assert(len(params.GID) != 0)

	params.Aliases = normalize(params.Aliases)
	Assert(len(params.Link) != 0)
	log.Trace("SaveSound", log.Any("params.Link", params.Link))

	aliases := strings.Split(params.Aliases, ",")
	Assert(len(aliases) > 0)

	f, err := d.downloadAsMP4(ctx, params.Link)
	if err != nil {
		return fmt.Errorf("failed to download file '%s': %w", params.Link, err)
	}

	name := fmt.Sprintf("%s.mp4", aliases[0])
	log.Trace("SaveSound", log.Any("link", name))
	file, err := d.backupFile(BackupFileParams{
		Name:        name,
		GID:         params.GID,
		Dirs:        "sounds",
		PrependTime: false,
		File:        f.File,
	})
	if err != nil {
		return fmt.Errorf("failed to backup file '%s': %w", params.Link, err)
	}

	sound, err := d.Store.AddSound(d.Ctx, store.AddSoundParams{
		Gid:       params.GID,
		Url:       file.Name,
		Aliases:   aliases,
		OriginUrl: params.Link,
	})
	if err != nil {
		return err
	}
	log.Trace("SaveSound", log.JSON(sound))

	return nil
}

type MP4File struct {
	Filepath string
	Name     string
	File     *os.File
}

func (d *DBot) downloadAsMP4(ctx context.Context, url string) (MP4File, error) {
	vi, err := d.DownloadVideo(ctx, url)
	if err != nil {
		return MP4File{}, fmt.Errorf("'%s': %w", url, err)
	}

	f, err := ffmpeg.ConvertToMP4(ctx, vi.Filepath)
	if err != nil {
		return MP4File{}, fmt.Errorf("convertToMP4 failed '%s': %w", vi.Filepath, err)
	}

	return MP4File{
		Filepath: vi.Filepath,
		Name:     vi.Filepath,
		File:     f,
	}, nil
}

func (d *DBot) Ready(s *discordgo.Session, e *discordgo.Ready) {
	log.Info("ready")
}

// returns users VoiceCHannel if user is connected to one
// errors if guild does not exist and if user is not in voice channel
func (d *DBot) getUserVC(s *discordgo.Session, gID string, uID string) (*discordgo.VoiceState, error) {
	g, err := s.State.Guild(gID)
	if err != nil {
		return nil, err
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID != uID {
			continue
		}
		return vs, nil
	}

	return nil, fmt.Errorf("user <@%s> in guild '%s' is not in voice channel", uID, gID)
}

func (d *DBot) mapChannel(params store.MapChannelParams) (store.Channel, error) {
	Assert(len(params.ChName) > 0)
	Assert(len(params.Gid) > 0)
	Assert(len(params.Chid) > 0)
	Assert(len(params.Type) > 0)
	ch, err := d.Store.MapChannel(d.Ctx, params)
	if err != nil {
		return store.Channel{}, dbotErr("failed to save: %w", err)
	}

	return ch, nil
}

func (d *DBot) wypierdalajZVC(gID string) error {
	if d.MusicPlayer.VC != nil {
		err := d.MusicPlayer.VC.Disconnect()
		if err != nil {
			d.MusicPlayer.ErrChan <- player.Err{
				GID: gID,
				Err: err,
			}
		}
	}
	d.MusicPlayer.Close()

	d.MusicPlayer = player.NewPlayer(d._c)
	return nil
}

func isValidUrl(s string) bool {
	u, err := url.Parse(s)
	log.Trace("isValidUrl", log.JSON(u))
	return err == nil && u.Host != ""
}

func (d *DBot) play(gID, uID string, url string) error {
	err := d.connectVoice(gID, uID)
	if err != nil {
		return err
	}

	switch {
	case !isValidUrl(url):
		err := d.searchAndPlay(url)
		if err != nil {
			return fmt.Errorf("failed searching: %w", err)
		}

	case strings.Contains(url, "/playlist"):
		err := d.playFromYTPlaylist(gID, url)
		if err != nil {
			return fmt.Errorf("failed load playlist: %w", err)
		}

	default:
		art, err := d.Backup.GetArtefact(d.Ctx, url)
		if err != nil {
			go saveInPlayHistory(d, url, gID)
		} else {
			log.Trace("playHistory: artefact already exists")
			url = art.Path
		}

		d.MusicPlayer.Add(url)
	}

	return nil
}

func saveInPlayHistory(d *DBot, url string, gID string) {
	log.Trace("saveInPlayHistory", log.String("url", url), log.String("gid", gID))
	meta := BackupFileParams{
		OriginUrl:   url,
		GID:         gID,
		Dirs:        "play_history",
		PrependTime: false,
		File:        nil,
	}

	jsonMeta, err := json.Marshal(meta)
	if err != nil {
		log.Error("lol dupa", log.Err(err), log.String("meta", fmt.Sprintf("%+v", meta)))
		return
	}

	_, err = d.Store.Enqueue(d.Ctx, store.EnqueueParams{
		Meta:    string(jsonMeta),
		JobType: BackupJob,
	})
	if err != nil {
		log.Error("lol dupa", log.Err(err), log.String("meta", fmt.Sprintf("%+v", meta)))
	}
}

func (d *DBot) searchAndPlay(url string) error {
	url = fmt.Sprintf(`ytsearch:"%s"`, url)
	d.MusicPlayer.Add(url)
	return nil
}

func (d *DBot) playFromYTPlaylist(gID, url string) error {
	info, err := d.PlaylistInfo(url)
	if err != nil {
		return fmt.Errorf("failed getting playlist info: %w", err)
	}

	log.Trace("adding tracks from playlist", log.Int("len", len(info.Entries)))
	for i := range info.Entries {
		if info.Entries[i].Duration == nil {
			log.Trace("skipping due to null duration, probably deleted vid",
				log.String("title", info.Entries[i].Title),
				log.String("url", info.Entries[i].URL),
			)
			continue
		}
		u := info.Entries[i].URL
		art, err := d.Backup.GetArtefact(d.Ctx, url)
		if err != nil {
			go saveInPlayHistory(d, url, gID)
		} else {
			u = art.Path
		}
		d.MusicPlayer.Add(u)
	}

	return nil
}

func (d *DBot) savePlaylistFromYT(name, url, gID string) error {
	info, err := d.PlaylistInfo(url)
	if err != nil {
		return fmt.Errorf("failed getting playlist info: %w", err)
	}
	if len(info.Entries) == 0 {
		return fmt.Errorf("failed to get video list from playlist")
	}

	playlist, err := d.Store.CreatePlaylist(d.Ctx, store.CreatePlaylistParams{
		GuildID: gID,
		Name:    name,
		YoutubeUrl: sql.NullString{
			String: info.WebpageURL,
			Valid:  true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed creating playlist: %w", err)
	}

	for i := range info.Entries {
		if info.Entries[i].Duration == nil {
			log.Trace("skipping due to null duration, probably deleted vid",
				log.String("title", info.Entries[i].Title),
				log.String("url", info.Entries[i].URL),
			)
			continue
		}

		meta := DownloadAsyncMeta{
			PlaylistID:  playlist.ID,
			URL:         info.Entries[i].URL,
			GID:         gID,
			Name:        info.Entries[i].Title,
			DownloadFor: "playlist_vids",
		}

		b, err := json.Marshal(meta)
		Assert(err == nil, err)

		_, err = d.Store.Enqueue(d.Ctx, store.EnqueueParams{
			Meta:      string(b),
			FailCount: 0,
			Status:    "new",
			JobType:   DownloadJob,
		})
		if err != nil {
			log.Error("lol dupa", log.Err(err), log.String("meta", fmt.Sprintf("%+v", meta)))
			continue
		}
	}

	return nil
}

func (d *DBot) loadPlaylistFromDB(name string, gID string) error {
	playlist, err := d.Store.GetPlaylist(d.Ctx, store.GetPlaylistParams{
		GuildID: gID,
		Name:    name,
	})
	if err != nil {
		return fmt.Errorf("could not find playlist: %w", err)
	}

	list, err := d.Store.ListPlaylistEntries(d.Ctx, playlist.ID)
	if err != nil {
		return fmt.Errorf("could not find playlist: %w", err)
	}

	for _, elem := range list {
		d.MusicPlayer.Add(elem.Filepath)
	}

	return nil
}
