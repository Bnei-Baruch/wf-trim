package api

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Bnei-Baruch/wf-trim/common"
	"github.com/gabriel-vasile/mimetype"
	"gopkg.in/vansante/go-ffprobe.v2"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Upload struct {
	Filename  string      `json:"file_name"`
	Extension string      `json:"extension,omitempty"`
	Sha1      string      `json:"sha1"`
	Size      int64       `json:"size"`
	Mimetype  string      `json:"type"`
	Url       string      `json:"url"`
	MediaInfo interface{} `json:"media_info"`
}

type Status struct {
	Status string `json:"status"`
	Out    string `json:"stdout"`
	Result string `json:"link"`
	Get    string `json:"download"`
}

func (s *Status) PutExec(endpoint string, p string) error {

	cmd := exec.Command("/opt/wfexec/"+endpoint+".sh", p)
	cmd.Dir = "/opt/wfexec/"
	out, err := cmd.CombinedOutput()

	if err != nil {
		s.Out = err.Error()
		return err
	}

	s.Out = string(out)
	json.Unmarshal(out, &s.Result)

	return nil
}

func (s *Status) newTrimExec(uid string, audio string, video string, sstart string, send string) error {

	fn, err := getNewFile(uid, audio, video)
	if err != nil {
		s.Out = err.Error()
		return err
	}

	inp := hmsParse(sstart)
	oup := hmsParse(send) - inp
	ss := strconv.Itoa(inp)
	tt := strconv.Itoa(oup)

	n := strings.Split(fn, ".")[0]
	e := strings.Split(fn, ".")[1]

	ifn := getInputFileName(fn, uid)
	ofn := n + "_" + sstart + "-" + send + "." + e
	s.Result = common.LINK_URL + ofn
	s.Get = common.GET_URL + ofn

	// Maybe someone already did trim with exact data
	if isExists(common.DATA_DIR + "/" + ofn) {
		return nil
	}

	var codec, args, input []string
	input = []string{"-y", "-ss", ss, "-i", common.SRC_DIR + "/" + ifn, "-to", tt}
	output := []string{"-f", "mp4", common.DATA_DIR + "/" + ofn}
	decoder := strings.Split("-init_hw_device qsv=hw -filter_hw_device hw -hwaccel qsv -hwaccel_output_format qsv", " ")

	if video == "hd" {
		codec = strings.Split("-c:v h264_qsv -profile:v high -preset veryfast -b:v 1000k -c:a aac", " ")
		input = append(decoder, input...)
	} else if video == "nhd" {
		codec = strings.Split("-c:v h264_qsv -profile:v main -preset veryfast -b:v 450k -c:a aac", " ")
		input = append(decoder, input...)
	} else if video == "fhd" {
		codec = strings.Split("-c:v h264_qsv -profile:v main -preset veryfast -b:v 2500k -c:a aac", " ")
		input = append(decoder, input...)
	} else if e == "m4a" {
		codec = strings.Split("-c:a aac -b:a 64k", " ")
	} else {
		err = errors.New("wrong options for demux")
		s.Out = "Wrong options for demux"
		return err
	}

	args = append(input, codec...)
	args = append(args, output...)

	out, err := exec.Command("ffmpeg", args...).CombinedOutput()

	if err != nil {
		s.Out = string(out)
		return err
	}

	s.Out = uid

	return nil
}

func (s *Status) oldTrimExec(uid string, sstart string, send string) error {

	fn, err := getFile(uid)
	if err != nil {
		s.Out = err.Error()
		return err
	}

	inp := hmsParse(sstart)
	oup := hmsParse(send) - inp
	ss := strconv.Itoa(inp)
	tt := strconv.Itoa(oup)

	n := strings.Split(fn, ".")[0]
	e := strings.Split(fn, ".")[1]
	h := strings.Split(n, "_")
	hd := h[len(h)-1]

	ifn := getInputFileName(fn, uid)
	ofn := n + "_" + sstart + "-" + send + "." + e
	s.Result = common.LINK_URL + ofn
	s.Get = common.GET_URL + ofn

	// Maybe someone already did trim with exact data
	if isExists(common.DATA_DIR + "/" + ofn) {
		return nil
	}

	var codec, args, input []string
	input = []string{"-y", "-ss", ss, "-i", common.SRC_DIR + "/" + ifn, "-to", tt}
	output := []string{"-f", e, common.DATA_DIR + "/" + ofn}
	decoder := strings.Split("-init_hw_device qsv=hw -filter_hw_device hw -hwaccel qsv -hwaccel_output_format qsv", " ")

	if hd == "hd" {
		codec = strings.Split("-c:v h264_qsv -profile:v high -preset veryfast -b:v 1000k -c:a aac", " ")
		input = append(decoder, input...)
	} else if e == "mp3" {
		codec = strings.Split("-c:a mp3 -ar 44100 -write_xing 0", " ")
	} else {
		codec = strings.Split("-c:v h264_qsv -profile:v main -preset veryfast -b:v 450k -c:a aac", " ")
		input = append(decoder, input...)
	}

	args = append(input, codec...)
	args = append(args, output...)

	out, err := exec.Command("ffmpeg", args...).CombinedOutput()

	if err != nil {
		s.Out = string(out)
		return err
	}

	s.Out = uid

	return nil
}

func hmsParse(hms string) int {
	hms = strings.Replace(hms, "h", ":", -1)
	hms = strings.Replace(hms, "m", ":", -1)
	hms = strings.Replace(hms, "s", "", -1)
	t := strings.Split(hms, ":")
	var h, m, s int

	switch l := len(t); l {
	case 3:
		_, _ = fmt.Sscanf(hms, "%d:%d:%d", &h, &m, &s)
	case 2:
		_, _ = fmt.Sscanf(hms, "%d:%d", &m, &s)
	case 1:
		_, _ = fmt.Sscanf(hms, "%d", &s)
	}

	return h*3600 + m*60 + s
}

func getNewFile(uid string, audio string, video string) (filename string, err error) {

	resp, err := http.Get(common.HLS_URL + "/get/" + uid + ".mp4?audio=" + audio + "&video=" + video)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	location := resp.Request.URL.String()
	parts := strings.Split(location, "/")
	filename = parts[len(parts)-1]
	ifn := getInputFileName(filename, uid)
	tmpFile := common.SRC_DIR + "/" + audio + "_" + ifn + ".part"
	finalFile := common.SRC_DIR + "/" + audio + "_" + ifn

	// Wait for finish if someone downloads wright now
	for {
		if !isExists(tmpFile) {
			break
		}
		println("Waiting")
		time.Sleep(2 * time.Second)
	}

	// Do not download twice same file
	if isExists(finalFile) {
		return filename, nil
	}

	out, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	err = os.Rename(tmpFile, finalFile)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func getFile(uid string) (filename string, err error) {

	resp, err := http.Get(common.CDN_URL + "/" + uid)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	location := resp.Request.URL.String()
	parts := strings.Split(location, "/")
	filename = parts[len(parts)-1]
	ifn := getInputFileName(filename, uid)
	tmpFile := common.SRC_DIR + "/" + ifn + ".part"
	finalFile := common.SRC_DIR + "/" + ifn

	// Wait for finish if someone downloads wright now
	for {
		if !isExists(tmpFile) {
			break
		}
		println("Waiting")
		time.Sleep(2 * time.Second)
	}

	// Do not download twice same file
	if isExists(finalFile) {
		return filename, nil
	}

	out, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	err = os.Rename(tmpFile, finalFile)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func getInputFileName(filename string, uid string) string {
	name := ""

	n := strings.Split(filename, ".")[0]
	e := strings.Split(filename, ".")[1]
	s := strings.Split(n, "_")
	video := s[len(s)-1]

	switch video {
	case "hd":
		name = uid + "_hd.mp4"
	case "nhd":
		name = uid + "_nhd.mp4"
	case "fhd":
		name = uid + "_fhd.mp4"
	default:
		name = uid + "." + e
	}

	return name
}

func isExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Status) GetStatus(endpoint string, id string, key string, value string) error {

	cmdArguments := []string{id, key, value}
	cmd := exec.Command("/opt/wfexec/get_"+endpoint+".sh", cmdArguments...)
	cmd.Dir = "/opt/wfexec/"
	out, err := cmd.CombinedOutput()

	if err != nil {
		s.Out = err.Error()
		return err
	}

	s.Out = string(out)
	json.Unmarshal(out, &s.Result)

	return nil
}

func (u *Upload) UploadProps(filepath string, ep string) error {

	f, err := os.Open(filepath)
	if err != nil {
		return err
	}

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	u.Size = fi.Size()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	u.Sha1 = hex.EncodeToString(h.Sum(nil))

	if ep == "insert" {
		newpath := "/backup/tmp/insert/" + u.Sha1
		err = os.Rename(u.Url, newpath)
		if err != nil {
			return err
		}
		u.Url = newpath
	}

	if ep == "products" {
		newpath := "/backup/files/upload/" + u.Sha1
		err = os.Rename(u.Url, newpath)
		if err != nil {
			return err
		}
		u.Url = newpath

		mt, err := mimetype.DetectFile(newpath)
		if err != nil {
			return err
		}

		u.Mimetype = mt.String()

		if u.Mimetype == "application/octet-stream" {
			u.Extension = "srt"
		} else {
			u.Extension = strings.Trim(mt.Extension(), ".")
		}

		ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelFn()

		data, err := ffprobe.ProbeURL(ctx, newpath)
		if err == nil {
			u.MediaInfo = data
		}
	}

	return nil
}
