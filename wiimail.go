package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/Disconnect24/Mail-GO/utilities"
	"github.com/nfnt/resize"

	"image"
	// We use jpeg to actually send to the Wii.
	"image/jpeg"

	// We don't actually use the following formats for encoding,
	// they're here for image format detection.
	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const CRLF = "\r\n"

func FormulateMail(from string, to string, subject string, body string, potentialImage []byte) (string, error) {
	boundary := utilities.GenerateBoundary()

	// Set up headers and set up first boundary with body.
	// The body could be empty: that's fine, it'll have no value
	// (compared to nil) and the Wii will ignore that section.
	mailContent := fmt.Sprint("From: ", from, CRLF,
		"Subject: ", subject, CRLF,
		"To: ", to, CRLF,
		"MIME-Version: 1.0", CRLF,
		`Content-Type: MULTIPART/mixed; BOUNDARY="`, boundary, `"`, CRLF,
		CRLF,
		"--", boundary, CRLF,
		"Content-Type: TEXT/plain; CHARSET=utf-8", CRLF,
		"Content-Description: wiimail", CRLF,
		CRLF,
	)

	normalMailFormat := fmt.Sprint(mailContent,
		body,
		strings.Repeat(CRLF, 3),
		"--", boundary, "--")

	// If there's an attachment, we need to factor that in.
	// Otherwise we're done.
	if potentialImage == nil {
		return normalMailFormat, nil
	}

	// The image library interprets known file types automatically.
	givenImg, _, err := image.Decode(bytes.NewReader(potentialImage))

	// The Wii has a max image size of 8192x8192px.
	// If any dimension exceeds that, scale to fit.
	outputImg := resize.Thumbnail(8192, 8192, givenImg, resize.Lanczos3)

	// Encode image as JPEG for the Wii to handle.
	var outputImgWriter bytes.Buffer
	err = jpeg.Encode(bufio.NewWriter(&outputImgWriter), outputImg, nil)
	if err != nil {
		log.Printf("Error transforming image: %v", err)
		return genError(mailContent, body, boundary), err
	}

	outputImgBytes, err := ioutil.ReadAll(bufio.NewReader(&outputImgWriter))
	if err != nil {
		log.Printf("Error transforming image: %v", err)
		return genError(mailContent, body, boundary), err
	}

	// The Wii's mailbox is roughly 7.3mb.
	// We'll cap any generated image at 7mb.
	if len(outputImgBytes) > 7*1024*1024 {
		return genError(mailContent, body, boundary), nil
	}

	encodedImage := base64.StdEncoding.EncodeToString(outputImgBytes)

	var splitEncoding string
	// 76 is a widely accepted base64 newline max char standard for mail.
	for {
		if len(encodedImage) >= 76 {
			// Separate the next 73.
			splitEncoding += encodedImage[:76] + CRLF
			encodedImage = encodedImage[76:]
		} else {
			// To the end.
			splitEncoding += encodedImage[:]
			break
		}
	}

	return fmt.Sprint(mailContent,
		body,
		strings.Repeat(CRLF, 3),
		"--", boundary, CRLF,
		// Now we can put our image data.
		"Content-Type: IMAGE/jpeg; name=converted.jpeg", CRLF,
		"Content-Transfer-Encoding: BASE64", CRLF,
		"Content-Disposition: attachment; filename=converted.jpeg", CRLF,
		CRLF,
		splitEncoding, CRLF,
		CRLF,
		"--", boundary, "--",
	), nil
}

func genError(mailContent string, body string, boundary string) string {
	return fmt.Sprint(mailContent,
		body,
		CRLF,
		"---",
		CRLF,
		"An error occurred processing the attached image.", CRLF,
		"For more information, ask the sender to forward this mail to support@riiconnect24.net.",
		strings.Repeat(CRLF, 3),
		"--", boundary, "--")
}
