package api

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
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
	Status string      `json:"status"`
	Out    string      `json:"stdout"`
	Result interface{} `json:"jsonst"`
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

func (s *Status) trimExec(uid string, sstart string, send string) error {

	fn, err := getFile(uid)
	if err != nil {
		s.Out = err.Error()
		return err
	}

	inp := hmsParse(sstart)
	oup := hmsParse(send) - inp
	ss := strconv.Itoa(inp)
	tt := strconv.Itoa(oup)

	cmdArguments := []string{fn, ss, tt}
	cmd := exec.Command(common.WORK_DIR+"/exec.sh", cmdArguments...)
	cmd.Dir = common.WORK_DIR
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.Out = err.Error()
		return err
	}

	s.Out = string(out)
	json.Unmarshal(out, &s.Result)

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

func getFile(uid string) (filename string, err error) {

	resp, err := http.Get(common.CDN_URL + "/" + uid)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	location := resp.Request.URL.String()
	parts := strings.Split(location, "/")
	filename = parts[len(parts)-1]

	out, err := os.Create(common.WORK_DIR + "/" + filename)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return filename, nil
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
