package main

import (
	"encoding/base64"
	"fmt"
	"github.com/Disconnect24/lilliput"
	"log"
	"strings"
)

const CRLF = "\r\n"

func FormulateMail(from string, to string, subject string, body string, potentialImage []byte) (string, error) {
	boundary := GenerateBoundary()

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

	decoder, err := lilliput.NewDecoder(potentialImage)
	if err != nil {
		// It's not valid for whatever reason. Ignore it.
		return normalMailFormat, nil
	}
	defer decoder.Close()

	// Buffer for image to return.
	// The Wii's receive mailbox is 7.3MB roughly.
	// We're going to have it be 7MB max output.
	outputImg := make([]byte, 7*1024*1024)

	// The Wii has a max image size of
	// 8192x8192px. If any dimension
	// exceeds that, stretch to fit.
	ops := lilliput.NewImageOps(8192)
	defer ops.Close()

	header, err := decoder.Header()
	if err != nil {
		log.Printf("error decoding image header: %v", err)
		return normalMailFormat, nil
	}

	opts := &lilliput.ImageOptions{
		FileType:             ".jpeg",
		Width:                header.Width(),
		Height:               header.Height(),
		ResizeMethod:         lilliput.ImageOpsResize,
		NormalizeOrientation: true,
		EncodeOptions:        map[int]int{lilliput.JpegQuality: 85},
	}

	// Actually resize image.
	outputImg, err = ops.Transform(decoder, opts, outputImg)
	if err != nil {
		log.Printf("Error transforming image: %v", err)
		// Inform the user an error occurred.
		return fmt.Sprint(mailContent,
			body,
			CRLF,
			"---",
			CRLF,
			"An error occurred processing the attached image.", CRLF,
			"For more information, ask the sender to forward this mail to support@riiconnect24.net.",
			strings.Repeat(CRLF, 3),
			"--", boundary, "--"), nil
	}

	encodedImage := base64.StdEncoding.EncodeToString(outputImg)

	var splitEncoding string

	// 76 is a widely accepted base64 newline max char standard for mail.
	for {
		// If we have 73 characters or less, carry on.
		if len(encodedImage) >= 76 {
			// Otherwise, separate the next 73.
			splitEncoding += encodedImage[:76] + CRLF
			// Chop off what was just done for next loop.
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
