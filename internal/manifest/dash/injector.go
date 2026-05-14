package dash

import (
	"fmt"
	"strings"

	"github.com/anupam-chopra/prism/internal/model"
)

func Inject(manifest []byte, asset *model.Asset) []byte {
	if asset.DRMType == "" {
		return manifest
	}

	contentProtection := contentProtectionElement(asset)
	if contentProtection == "" {
		return manifest
	}

	xmlText := string(manifest)
	if asset.DRMType == "widevine" {
		xmlText = strings.Replace(xmlText, "<MPD ", `<MPD xmlns:cenc="urn:mpeg:cenc:2013" `, 1)
	}

	lines := strings.Split(xmlText, "\n")
	output := make([]string, 0, len(lines)+4)
	for _, line := range lines {
		output = append(output, line)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<AdaptationSet") && strings.HasSuffix(trimmed, ">") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			output = append(output, indent+"  "+contentProtection)
		}
	}

	return []byte(strings.Join(output, "\n"))
}

func contentProtectionElement(asset *model.Asset) string {
	switch asset.DRMType {
	case "widevine":
		return fmt.Sprintf(`<ContentProtection schemeIdUri="urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed" cenc:default_KID="%s"/>`, asset.KeyID)
	case "fairplay":
		return `<ContentProtection schemeIdUri="urn:uuid:94ce86fb-07ff-4f43-adb8-93d2fa968ca2"/>`
	case "playready":
		return `<ContentProtection schemeIdUri="urn:uuid:9a04f079-9840-4286-ab92-e65be0885f95"/>`
	default:
		return ""
	}
}
