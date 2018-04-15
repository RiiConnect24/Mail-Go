package patch

import (
	"errors"
	"encoding/binary"
	"bytes"
	"io/ioutil"
	"strconv"
	"database/sql"
	"log"
	"golang.org/x/crypto/bcrypt"
)

// ModifyNwcConfig takes an original config, applies needed patches to the URL and such,
// updates the checksum and returns either nil, error or a patched config w/o error.
func ModifyNwcConfig(originalConfig []byte, db *sql.DB, global Config) ([]byte, error) {
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

	// Go ahead and push read data.
	mlchkid := RandStringBytesMaskImprSrc(32)
	passwd := RandStringBytesMaskImprSrc(16)

	mlchkidByte, err := bcrypt.GenerateFromPassword([]byte(mlchkid), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Bcrypt error: %v", err)
		return nil, err
	}

	passwdByte, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Bcrypt error: %v", err)
		return nil, err
	}


	stmt, err := db.Prepare("INSERT IGNORE INTO `accounts` (`mlid`,`mlchkid`, `passwd` ) VALUES (?, ?, ?)")
	if err != nil {
		log.Printf("Database error: %v", err)
		return nil, err
	}

	_, err = stmt.Exec(mlid, mlchkidByte, passwdByte)
	if err != nil {
		log.Printf("Database error: %v", err)
		return nil, err
	}

	// Alright, now it's time to patch.
	copy(config.MailDomain[:], []byte(global.SendGridDomain))

	// Copy changed credentials
	copy(config.Mlchkid[:], []byte(mlchkid))
	copy(config.Passwd[:], []byte(passwd))

	// The following is very redundantly written. TODO: fix that?
	copy(config.AccountURL[:128], []byte(global.PatchBaseDomain + "/cgi-bin/account.cgi"))
	copy(config.CheckURL[:128], []byte(global.PatchBaseDomain + "/cgi-bin/check.cgi"))
	copy(config.ReceiveURL[:128], []byte(global.PatchBaseDomain + "/cgi-bin/receive.cgi"))
	copy(config.DeleteURL[:128], []byte(global.PatchBaseDomain + "/cgi-bin/delete.cgi"))
	copy(config.SendURL[:128], []byte(global.PatchBaseDomain + "/cgi-bin/send.cgi"))

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
