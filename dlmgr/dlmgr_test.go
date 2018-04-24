package dlmgr

import "testing"

func TestDownload(t *testing.T) {
	d := NewDownloader("/cygdrive/d/cygwin64/tmp")
	d.Start()
	d.Download(&ContentDL{
		Name:    "captive",
		URL:     "http://www.cloudfi.cn/captive.html",
		Hashsum: "41ba060eb1c0898e0a4a0cca36a8ca91",
	}, nil)
	select {}
}
