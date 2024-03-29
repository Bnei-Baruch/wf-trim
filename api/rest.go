package api

import (
	"bytes"
	"github.com/Bnei-Baruch/wf-trim/common"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"
)

func getUploadPath(ep string) string {

	switch ep {
	case "insert":
		return "/backup/tmp/insert/"
	case "jobs":
		return "/backup/jobs/"
	case "products":
		return "/backup/files/upload/"
	case "aricha":
		return "/backup/aricha/"
	case "aklada":
		return "/backup/tmp/akladot/"
	case "gibuy":
		return "/tmp/"
	case "carbon":
		return "/backup/tmp/carbon/"
	case "dgima":
		return "/backup/dgima/"
	case "proxy":
		return "/backup/tmp/proxy/"
	case "youtube":
		return "/backup/tmp/youtube/"
	case "coder":
		return "/backup/tmp/coder/"
	case "muxer":
		return "/backup/tmp/muxer/"
	default:
		return "/backup/tmp/upload/"
	}
}

func (a *App) handleUpload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	endpoint := vars["ep"]
	var u Upload

	uploadpath := getUploadPath(endpoint)

	if _, err := os.Stat(uploadpath); os.IsNotExist(err) {
		os.MkdirAll(uploadpath, 0755)
	}

	var n int
	var err error

	// define pointers for the multipart reader and its parts
	var mr *multipart.Reader
	var part *multipart.Part

	//log.Println("File Upload Endpoint Hit")

	if mr, err = r.MultipartReader(); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// buffer to be used for reading bytes from files
	chunk := make([]byte, 10485760)

	// continue looping through all parts, *multipart.Reader.NextPart() will
	// return an End of File when all parts have been read.
	for {
		// variables used in this loop only
		// tempfile: filehandler for the temporary file
		// filesize: how many bytes where written to the tempfile
		// uploaded: boolean to flip when the end of a part is reached
		var tempfile *os.File
		var filesize int
		var uploaded bool

		if part, err = mr.NextPart(); err != nil {
			if err != io.EOF {
				respondWithError(w, http.StatusInternalServerError, err.Error())
			} else {
				respondWithJSON(w, http.StatusOK, u)
			}
			return
		}
		// at this point the filename and the mimetype is known
		//log.Printf("Uploaded filename: %s", part.FileName())
		//log.Printf("Uploaded mimetype: %s", part.Header)

		u.Filename = part.FileName()
		u.Mimetype = part.Header.Get("Content-Type")
		u.Url = uploadpath + u.Filename

		tempfile, err = ioutil.TempFile(uploadpath, part.FileName()+".*")
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer tempfile.Close()
		//defer os.Remove(tempfile.Name())

		// continue reading until the whole file is upload or an error is reached
		for !uploaded {
			if n, err = part.Read(chunk); err != nil {
				if err != io.EOF {
					respondWithError(w, http.StatusInternalServerError, err.Error())
					return
				}
				uploaded = true
			}

			if n, err = tempfile.Write(chunk[:n]); err != nil {
				respondWithError(w, http.StatusInternalServerError, err.Error())
				return
			}
			filesize += n
		}

		// once uploaded something can be done with the file, the last defer
		// statement will remove the file after the function returns so any
		// errors during upload won't hit this, but at least the tempfile is
		// cleaned up

		os.Rename(tempfile.Name(), u.Url)
		u.UploadProps(u.Url, endpoint)
	}
}

func (a *App) handleDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	file := vars["file"]
	dlBytes, err := ioutil.ReadFile(common.DATA_DIR + "/" + file)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	mime := http.DetectContentType(dlBytes)
	fileSize := len(string(dlBytes))
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", "attachment; filename="+file+"")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Content-Length", strconv.Itoa(fileSize))
	w.Header().Set("Content-Control", "private, no-transform, no-store, must-revalidate")
	http.ServeContent(w, r, file, time.Now(), bytes.NewReader(dlBytes))
}

func (a *App) putJson(w http.ResponseWriter, r *http.Request) {
	var s Status
	vars := mux.Vars(r)
	endpoint := vars["ep"]

	b, _ := ioutil.ReadAll(r.Body)

	err := s.PutExec(endpoint, string(b))

	defer r.Body.Close()

	if err != nil {
		s.Status = "error"
	} else {
		s.Status = "ok"
	}

	respondWithJSON(w, http.StatusOK, s)
}

func (a *App) statusJson(w http.ResponseWriter, r *http.Request) {
	var s Status
	vars := mux.Vars(r)
	endpoint := vars["ep"]
	id := r.FormValue("id")
	key := r.FormValue("key")
	value := r.FormValue("value")

	err := s.GetStatus(endpoint, id, key, value)

	if err != nil {
		s.Status = "error"
	} else {
		s.Status = "ok"
	}

	respondWithJSON(w, http.StatusOK, s)
}

func (a *App) trimExec(w http.ResponseWriter, r *http.Request) {
	var s Status

	uid := r.FormValue("uid")
	sstart := r.FormValue("sstart")
	send := r.FormValue("send")
	audio := r.FormValue("audio")
	video := r.FormValue("video")
	var err error

	if audio != "" {
		//HLS Trim
		err = s.newTrimExec(uid, audio, video, sstart, send)
	} else {
		//Files Trim
		err = s.oldTrimExec(uid, sstart, send)
	}

	if err != nil {
		s.Status = "error"
	} else {
		s.Status = "ok"
	}

	respondWithJSON(w, http.StatusOK, s)
}

func (a *App) getFilesList(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	ep := vars["ep"]
	var list []string

	files, err := ioutil.ReadDir("/" + ep)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Not found")
	}

	for _, f := range files {
		if f.Size() > 1024*1024 {
			list = append(list, f.Name())
		}
	}

	respondWithJSON(w, http.StatusOK, list)
}
