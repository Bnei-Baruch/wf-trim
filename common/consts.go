package common

import "os"

var (
	PORT      = os.Getenv("PORT")
	ACC_URL   = os.Getenv("ACC_URL")
	SKIP_AUTH = os.Getenv("SKIP_AUTH") == "true"
	LOG_PATH  = os.Getenv("LOG_PATH")
)
