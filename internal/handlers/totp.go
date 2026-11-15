package handlers

import (
	"bytes"
	"encoding/base64"
	"image/png"

	"github.com/pquerna/otp/totp"
)

func generateTOTPSetup(username string) (secret, otpauthURL, qrCode string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "AutoMarket",
		AccountName: username,
	})
	if err != nil {
		return "", "", "", err
	}

	image, err := key.Image(200, 200)
	if err != nil {
		return "", "", "", err
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, image); err != nil {
		return "", "", "", err
	}

	return key.Secret(), key.URL(), base64.StdEncoding.EncodeToString(pngData.Bytes()), nil
}
