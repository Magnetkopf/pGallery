package cli

import (
	"github.com/Magnetkopf/pGallery/internal/web"
)

type WebUIArgs = web.ServerArgs

func WebUI(args WebUIArgs) {
	web.Start(args)
}
