package patch

import (
	"errors"
	"encoding/binary"
	"bytes"
	"io/ioutil"
	"strconv"
	"database/sql"
	"log"
	"crypto/sha512"
	"encoding/hex"
)

// ModifyNwcConfig takes an original config, applies needed patches to the URL and such,
// updates the checksum and returns either nil, error or a patched config w/o error.
func ModifyNwcConfig(originalConfig []byte, db *sql.DB, global Config, salt []byte) ([]byte, error) {
	if len(originalConfig) == 0 {
		return nil, errors.New("config seems to be empty. double check you uploaded a file")
	}

	if len(originalConfig) != 1024 {
		return nil, errors.New("invalid config size")
	}

	var config ConfigFormat
	configReadingBuf := bytes.NewBuffer(originalConfig)
	err := binary.Read(configReadingBuf, binary.BigEndian, &config)
	if err != nil {
		return nil, err
	}

	if bytes.Compare(config.Magic[:], ConfigMagic) != 0 {
		return nil, errors.New("invalid magic")
	}

	// Figure out mlid
	mlid := strconv.Itoa(int(config.FriendCode))
	if len(mlid) == 15 {
		// Chances are this has a 0 at the start.
		mlid = "0" + mlid
	}
	mlid = "w" + mlid

	// Go ahead and push generated data.
	mlchkid := RandStringBytesMaskImprSrc(32)
	mlchkidByte := sha512.Sum512(append(salt, []byte(mlchkid)...))
	mlchkidHash := hex.EncodeToString(mlchkidByte[:])

	passwd := RandStringBytesMaskImprSrc(16)
	passwdByte := sha512.Sum512(append(salt, []byte(passwd)...))
	passwdHash := hex.EncodeToString(passwdByte[:])

	stmt, err := db.Prepare("INSERT IGNORE INTO `accounts` (`mlid`,`mlchkid`, `passwd` ) VALUES (?, ?, ?)")
	if err != nil {
		log.Printf("Database error: %v", err)
		return nil, err
	}

	_, err = stmt.Exec(mlid, mlchkidHash, passwdHash)
	if err != nil {
		log.Printf("Database error: %v", err)
		return nil, err
	}

	// Alright, now it's time to patch.
	var newMailDomain [64]byte
	copy(newMailDomain[:], []byte("@" + global.SendGridDomain))
	config.MailDomain = newMailDomain

	// Copy changed credentials
	var newMlchkid [36]byte
	copy(newMlchkid[:], []byte(mlchkid))
	config.Mlchkid = newMlchkid

	var newPasswd [32]byte
	copy(newPasswd[:], []byte(passwd))
	config.Passwd = newPasswd

	// The following is extremely redundantly written. TODO: fix that?
	var newAccountURL [128]byte
	copy(newAccountURL[:], []byte(global.PatchBaseDomain+"/cgi-bin/account.cgi"))
	config.AccountURL = newAccountURL

	var newCheckURL [128]byte
	copy(newCheckURL[:], []byte(global.PatchBaseDomain+"/cgi-bin/check.cgi"))
	config.CheckURL = newCheckURL

	var newRecieveURL [128]byte
	copy(newRecieveURL[:], []byte(global.PatchBaseDomain+"/cgi-bin/receive.cgi"))
	config.ReceiveURL = newRecieveURL

	var newDeleteURL [128]byte
	copy(newDeleteURL[:], []byte(global.PatchBaseDomain+"/cgi-bin/delete.cgi"))
	config.DeleteURL = newDeleteURL

	var newSendURL [128]byte
	copy(newSendURL[:], []byte(global.PatchBaseDomain+"/cgi-bin/send.cgi"))
	config.SendURL = newSendURL

	// Enable title booting
	config.TitleBooting = 1

	// Read from struct to buffer
	fileBuf := new(bytes.Buffer)
	err = binary.Write(fileBuf, binary.BigEndian, config)
	if err != nil {
		return nil, err
	}
	patchedConfig, err := ioutil.ReadAll(fileBuf)
	if err != nil {
		return nil, err
	}

	var checksumInt uint32

	// Checksum.
	// We loop from 1020 to avoid current checksum.
	// Take every 4 bytes, add 'er up!
	for i := 0; i < 1020; i += 4 {
		addition := binary.BigEndian.Uint32(patchedConfig[i : i+4])
		checksumInt += addition
	}

	// Grab lower 32 bits of int
	var finalChecksum uint32
	finalChecksum = checksumInt & 0xFFFFFFFF
	binaryChecksum := make([]byte, 4)
	binary.BigEndian.PutUint32(binaryChecksum, finalChecksum)

	// Update patched config checksum
	copy(patchedConfig[1020:1024], binaryChecksum)
	return patchedConfig, nil
}
