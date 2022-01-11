package cmd

import (
	"github.com/Bnei-Baruch/wf-trim/api"
	"github.com/Bnei-Baruch/wf-trim/common"
)

func Init() {
	a := api.App{}
	a.InitAuthClient()
	a.InitServer()
	a.Run(":" + common.PORT)
}
