package common

import "os"

var (
	PORT      = os.Getenv("PORT")
	ACC_URL   = os.Getenv("ACC_URL")
	CDN_URL   = os.Getenv("CDN_URL")
	HLS_URL   = os.Getenv("HLS_URL")
	LINK_URL  = os.Getenv("LINK_URL")
	GET_URL   = os.Getenv("GET_URL")
	SKIP_AUTH = os.Getenv("SKIP_AUTH") == "true"
	LOG_PATH  = os.Getenv("LOG_PATH")
	SRC_DIR   = os.Getenv("WORK_DIR")
	DATA_DIR  = os.Getenv("DATA_DIR")
)
