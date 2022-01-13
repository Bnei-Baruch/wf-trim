package common

import "os"

var (
	PORT      = os.Getenv("PORT")
	ACC_URL   = os.Getenv("ACC_URL")
	CDN_URL   = os.Getenv("CDN_URL")
	LINK_URL  = os.Getenv("LINK_URL")
	SKIP_AUTH = os.Getenv("SKIP_AUTH") == "true"
	LOG_PATH  = os.Getenv("LOG_PATH")
	WORK_DIR  = os.Getenv("WORK_DIR")
)
