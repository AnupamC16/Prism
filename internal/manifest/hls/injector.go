package hls

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/anupam-chopra/prism/internal/model"
)

func Inject(manifest []byte, asset *model.Asset) []byte {
	if asset.DRMType == "" {
		return manifest
	}

	tag := keyTag(asset)
	if tag == "" {
		return manifest
	}

	const marker = "#EXT-X-INDEPENDENT-SEGMENTS"
	return []byte(strings.Replace(string(manifest), marker+"\n", marker+"\n"+tag+"\n", 1))
}

func keyTag(asset *model.Asset) string {
	keyIDBase64 := base64.StdEncoding.EncodeToString([]byte(asset.KeyID))

	switch asset.DRMType {
	case "widevine":
		return fmt.Sprintf(`#EXT-X-KEY:METHOD=SAMPLE-AES,URI="data:text/plain;base64,%s",KEYFORMAT="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed",KEYFORMATVERSIONS="1"`, keyIDBase64)
	case "fairplay":
		return fmt.Sprintf(`#EXT-X-KEY:METHOD=SAMPLE-AES,URI="skd://fairplay/%s",KEYFORMAT="com.apple.streamingkeydelivery",KEYFORMATVERSIONS="1"`, asset.KeyID)
	case "playready":
		return fmt.Sprintf(`#EXT-X-KEY:METHOD=SAMPLE-AES,URI="data:text/plain;charset=UTF-16;base64,%s",KEYFORMAT="com.microsoft.playready",KEYFORMATVERSIONS="1"`, keyIDBase64)
	default:
		return ""
	}
}
