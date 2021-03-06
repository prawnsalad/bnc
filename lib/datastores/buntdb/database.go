// Copyright (c) 2012-2014 Jeremy Latt
// Copyright (c) 2016-2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package bncDataStoreBuntdb

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/tidwall/buntdb"
)

const (
	// 'version' of the database schema
	keySchemaVersion = "db.version"
	// latest schema of the db
	latestDbSchema = "1"
	// key for the primary salt used by the ircd
	KeySalt = "crypto.salt"

	// KeyUserInfo stores the general info of a specific user in our database
	KeyUserInfo = "user.info %s"
	// KeyUserPermissions stores the permissions that the given user has access to
	KeyUserPermissions = "user.permissions %s"

	KeyServerConnectionInfo      = "user.server.info %s %s"
	KeyServerConnectionAddresses = "user.server.addresses %s %s"
	KeyServerConnectionBuffers   = "user.server.channels %s %s"
)

// these are types used to store information in / retrieve information from the database

// UserInfo stores information about the user in our database
type UserInfo struct {
	ID                  string
	Name                string `json:"username"`
	Role                string
	EncodedSalt         string `json:"salt"`
	EncodedPasswordHash string `json:"hash"`
	DefaultNick         string `json:"default-nick"`
	DefaultNickFallback string `json:"default-nick-fallback"`
	DefaultUsername     string `json:"default-username"`
	DefaultRealname     string `json:"default-realname"`
}

// UserPermissions is a list of permissions the user has access to
type UserPermissions []string

// ServerConnectionMapping maps ServerConnection to its JSON structure
type ServerConnectionMapping struct {
	Name             string
	Enabled          bool
	ConnectPassword  string `json:"connect-password"`
	Nickname         string
	NicknameFallback string
	Username         string
	Realname         string
}

// ServerConnectionAddressMapping maps ServerConnectionAddress to its JSON structure
type ServerConnectionAddressMapping struct {
	Host      string
	Port      int
	UseTLS    bool `json:"use-tls"`
	VerifyTLS bool `json:"verify-tls"`
}

// ServerConnectionBufferMapping maps ServerConnectionBuffer to its JSON structure
type ServerConnectionBufferMapping struct {
	Channel  bool
	Name     string
	Key      string
	UseKey   bool  `json:"use_key"`
	LastSeen int64 `json:"last_seen"`
}

// InitDB creates the database.
func InitDB(path string) {
	// prepare kvstore db
	//TODO(dan): fail if already exists instead? don't want to overwrite good data
	os.Remove(path)
	store, err := buntdb.Open(path)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to open datastore: %s", err.Error()))
	}
	defer store.Close()

	err = store.Update(func(tx *buntdb.Tx) error {
		// set base db salt
		salt := NewSalt()
		encodedSalt := base64.StdEncoding.EncodeToString(salt)
		if err != nil {
			log.Fatal("Could not generate cryptographically-secure salt for the database:", err.Error())
		}
		tx.Set(KeySalt, encodedSalt, nil)

		// set schema version
		tx.Set(keySchemaVersion, latestDbSchema, nil)
		return nil
	})

	if err != nil {
		log.Fatal("Could not save datastore:", err.Error())
	}
}

// UpgradeDB upgrades the datastore to the latest schema.
func UpgradeDB(path string) {
	store, err := buntdb.Open(path)
	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to open datastore: %s", err.Error()))
	}
	defer store.Close()

	err = store.Update(func(tx *buntdb.Tx) error {
		version, _ := tx.Get(keySchemaVersion)

		// datastore upgrading code here
		fmt.Println("db version is", version, "but no upgrading code is written yet")

		return nil
	})
	if err != nil {
		log.Fatal("Could not update datastore:", err.Error())
	}

	return
}
