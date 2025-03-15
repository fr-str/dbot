package ffmpeg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/fr-str/log"
)

var probeCMD = []string{
	"-v", "error",
	"-show_streams",
	"-print_format", "json",
}

type (
	Streams []Stream
	Stream  struct {
		CodecName      string `json:"codec_name"`
		Profile        string `json:"profile"`
		CodecType      string `json:"codec_type"`
		CodecTagString string `json:"codec_tag_string"`
		Width          int    `json:"width,omitempty"`
		Height         int    `json:"height,omitempty"`
		CodedWidth     int    `json:"coded_width,omitempty"`
		CodedHeight    int    `json:"coded_height,omitempty"`
		PixFmt         string `json:"pix_fmt,omitempty"`
		Level          int    `json:"level,omitempty"`
		ColorRange     string `json:"color_range,omitempty"`
		StartTime      string `json:"start_time"`
		DurationTs     int    `json:"duration_ts"`
		Duration       string `json:"duration"`
		BitRate        string `json:"bit_rate"`

		SampleFmt      string `json:"sample_fmt,omitempty"`
		SampleRate     string `json:"sample_rate,omitempty"`
		Channels       int    `json:"channels,omitempty"`
		ChannelLayout  string `json:"channel_layout,omitempty"`
		BitsPerSample  int    `json:"bits_per_sample,omitempty"`
		InitialPadding int    `json:"initial_padding,omitempty"`
	}
)

func (f *Streams) UnmarshalJSON(src []byte) error {
	var tmp struct {
		Streams json.RawMessage `json:"streams"`
	}
	err := json.Unmarshal(src, &tmp)
	if err != nil {
		return err
	}

	type alias Streams
	var dupa alias
	err = json.Unmarshal(tmp.Streams, &dupa)
	if err != nil {
		return fmt.Errorf("dupa2: %w", err)
	}
	*f = Streams(dupa)

	return nil
}

func Probe(path string) (Streams, error) {
	cmd := exec.Command("ffprobe", append(probeCMD, path)...)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var meta Streams
	log.Trace("Probe", log.String("cmd", cmd.String()), log.String("link", path))
	err := cmd.Run()
	if err != nil {
		b, _ := io.ReadAll(stderr)
		return meta, errors.New(string(b))
	}

	err = json.NewDecoder(stdout).Decode(&meta)
	if err != nil {
		return meta, err
	}

	return meta, nil
}
