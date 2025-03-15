package dbot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"dbot/pkg/cache"
	"dbot/pkg/config"
	"dbot/pkg/db"
	"dbot/pkg/dbg"
	jobrunner "dbot/pkg/job_runner"
	miniocli "dbot/pkg/minio"
	"dbot/pkg/player"
	"dbot/pkg/store"
	"dbot/pkg/ytdlp"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
	"github.com/minio/minio-go/v7"
)

type DBot struct {
	// only used to create new Player instances
	_c *cache.Queries

	Ctx   context.Context
	Store *store.Queries
	MinIO miniocli.Minio

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
	return strings.ToLower(normalizeReplacer.Replace(s))
}

func startJobRunner(ctx context.Context, db *store.Queries) jobrunner.Runner {
	runner := jobrunner.NewRunner(ctx, db)
	go runner.Loop()
	return runner
}

func Start(ctx context.Context, sess *discordgo.Session, dbstore *store.Queries, minIO miniocli.Minio) {
	audioCache, err := db.ConnectAudioCache(ctx, "audio-cache.db", "")
	if err != nil {
		panic(err)
	}

	d := DBot{
		_c:          audioCache,
		Ctx:         ctx,
		Session:     sess,
		MusicPlayer: player.NewPlayer(audioCache),
		Store:       dbstore,
		MinIO:       minIO,
	}

	runner := startJobRunner(ctx, dbstore)
	runner.RegisterJob(DownloadJob, d.downloadAsync)

	// Listeners must be registered befor we open connection
	d.RegisterEventListiners()

	d.StartScheduler()
	go d.interfaceLoop()

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
			if errors.Is(Err.Err, ytdlp.ErrFailedToDownload) {
				msg = "could not download video"
			}

			_, err = d.ChannelMessageSend(chID, msg)
			if err != nil {
				log.Error(err.Error())
				continue
			}

		}
	}()
}

const (
	musicChannel = "music"
	errorChannel = "error"
	adminChannel = "admin"
)

func uniqueVideoName(name string) string {
	fileName, ext, found := strings.Cut(name, ".")
	if !found {
		// assume mp4
		newName := fmt.Sprintf("%s.mp4", name)
		// newName := fmt.Sprintf("%s_%s.mp4", name, uuid.NewString())
		log.Trace("uniqueVideoName", log.Any("newName", newName))
		return newName
	}

	// newName := fmt.Sprintf("%s_%s.%s", fileName, uuid.NewString(), ext)
	newName := fmt.Sprintf("%s.%s", fileName, ext)
	log.Trace("uniqueVideoName", log.Any("newName", newName))
	return newName
}

type SaveSoundParams struct {
	GID     string
	Aliases string `opt:"aliases"`
	Link    string `opt:"link"`
	// after unmarshal only ID field will be populated
	// if attachment was provided
	Att *discordgo.MessageAttachment `opt:"file"`
}

func (d *DBot) SaveSound(params SaveSoundParams) error {
	dbg.Assert(len(params.GID) != 0)

	params.Aliases = normalize(params.Aliases)
	dbg.Assert(len(params.Link) != 0)
	log.Trace("SaveSound", log.Any("params.Link", params.Link))

	aliases := strings.Split(params.Aliases, ",")
	dbg.Assert(len(aliases) > 0)

	info, err := d.storeMediaInMinIO(aliases[0], params.Link, params.GID)
	if err != nil {
		return fmt.Errorf("failed to save attachment '%s': %w", params.Link, err)
	}

	link := linkFromMinioUploadInfo(filepath.Join(params.GID, "sounds", info.Key))
	log.Trace("SaveSound", log.Any("link", link))

	sound, err := d.Store.AddSound(d.Ctx, store.AddSoundParams{
		Gid:       params.GID,
		Url:       link,
		Aliases:   aliases,
		OriginUrl: params.Link,
	})
	if err != nil {
		return err
	}
	log.Trace("SaveSound", log.JSON(sound))

	return nil
}

func linkFromMinioUploadInfo(key string) string {
	dbg.Assert(len(key) > 0, "")
	return fmt.Sprintf("%s,%s", config.MINIO_DBOT_BUCKET_NAME, key)
}

func (d *DBot) storeMediaInMinIO(name, url, gID string) (minio.UploadInfo, error) {
	file, err := d.downloadAsMP4(url)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("storeMediaInMinIO: %w", err)
	}
	defer file.Close()

	err = d.MinIO.CreateFolderStructure(d.Ctx, gID)
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("storeMediaInMinIO: %w", err)
	}

	defer file.body.Close()
	info, err := d.MinIO.PutObject(d.Ctx,
		config.MINIO_DBOT_BUCKET_NAME,
		uniqueVideoName(name),
		file.body,
		file.size,
		minio.PutObjectOptions{
			ContentType: file.contentType,
		})
	if err != nil {
		return minio.UploadInfo{}, fmt.Errorf("failed to put in minio, '%s': %w", name, err)
	}
	log.Trace("uploadAttachmentToMinIO", log.JSON(info))

	return info, nil
}

type file struct {
	body        io.ReadCloser
	size        int64
	contentType string
	ogFile      string
	ffmpegFile  string
}

func (f file) Close() {
	if f.ogFile == "" && f.ffmpegFile == "" {
		f.body.Close()
		return
	}
	dbg.Assert(len(f.ogFile) > 0)
	dbg.Assert(len(f.ffmpegFile) > 0)
	err := os.Remove(f.ogFile)
	if err != nil {
		log.Error("failed to delete ogFile", log.Err(err))
	}

	err = os.Remove(f.ffmpegFile)
	if err != nil {
		log.Error("failed to delete ffmpegFile", log.Err(err))
	}
}

// file.body has to be closed after use
func (d *DBot) downloadAsMP4(url string) (file, error) {
	if strings.Contains(url, "dodupy.dev") {
		resp, err := http.Get(url)
		if err != nil {
			return file{}, fmt.Errorf("failed to get '%s': %w", url, err)
		}

		return file{
			body:        resp.Body,
			size:        resp.ContentLength,
			contentType: resp.Header.Get("content-type"),
		}, nil
	}

	vi, err := d.DownloadVideo(url)
	if err != nil {
		return file{}, fmt.Errorf("'%s': %w", url, err)
	}

	f, err := d.convertToMP4(vi.Filepath)
	if err != nil {
		return file{}, fmt.Errorf("convertToMP4 failed '%s': %w", vi.Filepath, err)
	}

	// size is only for optimizing transport to minio
	// and i don't care enought to get the size here
	return file{body: f, size: -1, ogFile: vi.Filepath, ffmpegFile: f.Name()}, nil
}

// os.File has to be closed manualy
func (d *DBot) convertToMP4(file string) (*os.File, error) {
	name := strings.ReplaceAll(file, filepath.Ext(file), "")
	mp4Path := filepath.Join(config.FFMPEG_TRANSCODE_PATH, fmt.Sprintf("edit.%s.mp4", filepath.Base(name)))
	// ffmpeg -i input.mp4 -map 0 -crf 18 -preset slow -b:a 96k -movflags +faststart -pix_fmt yuv420p out.mp4
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-i", file,
		"-c:v", "libx264",
		"-crf", "18",
		"-preset", "slow",
		"-map", "0",
		"-movflags", "+faststart",
		"-pix_fmt", "yuv420p",
		mp4Path,
	)

	log.Info("convertToMP4", log.String("cmd", cmd.String()))
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("cmd.Start failed: %w", err)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(mp4Path)
	if err != nil {
		return nil, err
	}

	return f, nil
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
	dbg.Assert(len(params.ChName) > 0)
	dbg.Assert(len(params.Gid) > 0)
	dbg.Assert(len(params.Chid) > 0)
	dbg.Assert(len(params.Type) > 0)
	ch, err := d.Store.MapChannel(d.Ctx, params)
	if err != nil {
		return store.Channel{}, dbotErr("failed to save: %w", err)
	}

	return ch, nil
}

func (d *DBot) getLinkFromSoundKey(key string) (string, error) {
	bucket, key, found := strings.Cut(key, ",")
	if !found {
		log.Warn(fmt.Sprintf("could not find separator in link '%s'", key))
		return "", errors.New(fmt.Sprintf("could not find separator in link '%s'", key))
	}

	url, err := d.MinIO.PresignedGetObject(d.Ctx, bucket, key, 5*time.Hour, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
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
		err := d.playFromYTPlaylist(url)
		if err != nil {
			return fmt.Errorf("failed load playlist: %w", err)
		}
	default:
		d.MusicPlayer.Add(url)
	}

	return nil
}

func (d *DBot) searchAndPlay(url string) error {
	url = fmt.Sprintf(`ytsearch:"%s"`, url)
	d.MusicPlayer.Add(url)
	return nil
}

func (d *DBot) playFromYTPlaylist(url string) error {
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
		d.MusicPlayer.Add(info.Entries[i].URL)
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
		dbg.Assert(err == nil, err)

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

	var topErr error
	for _, elem := range list {
		link, err := d.getLinkFromSoundKey(elem.MinioUrl)
		if err != nil {
			topErr = errors.Join(topErr, err)
			continue
		}
		d.MusicPlayer.Add(link)
	}
	if topErr != nil {
		log.Error("failed getting Link from Key", log.Err(topErr))
	}

	return nil
}
