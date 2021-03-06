package bncComponentLogger

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"

	_ "github.com/mattn/go-sqlite3"
)

const TYPE_MESSAGE = 1
const TYPE_ACTION = 2
const TYPE_NOTICE = 3

type SqliteMessage struct {
	ts          int32
	user        string
	network     string
	buffer      string
	from        string
	messageType int
	line        string
}

type SqliteMessageDatastore struct {
	dbPath       string
	db           *sql.DB
	messageQueue chan SqliteMessage
}

func (ds *SqliteMessageDatastore) SupportsStore() bool {
	return true
}
func (ds *SqliteMessageDatastore) SupportsRetrieve() bool {
	return true
}
func (ds *SqliteMessageDatastore) SupportsSearch() bool {
	return false
}
func NewSqliteMessageDatastore(config map[string]string) *SqliteMessageDatastore {
	ds := &SqliteMessageDatastore{}

	ds.dbPath = config["database"]
	db, err := sql.Open("sqlite3", ds.dbPath)
	if err != nil {
		log.Fatal(err)
	}

	ds.db = db

	// Create the tables if needed
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS messages (uid TEXT, netid TEXT, ts INT, buffer TEXT, fromNick TEXT, type INT, line TEXT)")
	if err != nil {
		log.Fatal("Error creates messages sqlite database:", err.Error())
	}

	// Start the queue to insert messages
	ds.messageQueue = make(chan SqliteMessage)
	go ds.messageWriter()

	return ds
}

func (ds *SqliteMessageDatastore) messageWriter() {
	storeStmt, err := ds.db.Prepare("INSERT INTO messages (uid, netid, ts, buffer, fromNick, type, line) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err.Error())
	}
	for {
		message, isOK := <-ds.messageQueue
		if !isOK {
			break
		}

		storeStmt.Exec(
			message.user,
			message.network,
			message.ts,
			strings.ToLower(message.buffer),
			message.from,
			message.messageType,
			message.line,
		)
	}
}

func (ds *SqliteMessageDatastore) Store(event *ircbnc.HookIrcRaw) {
	from, buffer, messageType, line := extractMessageParts(event)
	if line == "" {
		return
	}

	ds.messageQueue <- SqliteMessage{
		ts:          int32(time.Now().UTC().Unix()),
		user:        event.User.ID,
		network:     event.Server.Name,
		buffer:      buffer,
		from:        from,
		messageType: messageType,
		line:        line,
	}
}
func (ds *SqliteMessageDatastore) GetFromTime(userID string, networkID string, buffer string, from time.Time, num int) []*ircmsg.IrcMessage {
	messages := []*ircmsg.IrcMessage{}

	sql := "SELECT ts, fromNick, type, line, buffer FROM messages WHERE uid = ? AND netid = ? AND buffer = ? AND ts > ? ORDER BY ts DESC LIMIT ?"
	rows, err := ds.db.Query(sql, userID, networkID, strings.ToLower(buffer), int32(from.UTC().Unix()), num)
	if err != nil {
		log.Println("GetBeforeTime() error: " + err.Error())
		return messages
	}
	for rows.Next() {
		m := rowToIrcMessage(rows)
		messages = append(messages, m)
	}

	// Reverse the messages so they're in order
	for i := 0; i < len(messages)/2; i++ {
		j := len(messages) - i - 1
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages
}
func (ds *SqliteMessageDatastore) GetBeforeTime(userID string, networkID string, buffer string, from time.Time, num int) []*ircmsg.IrcMessage {
	messages := []*ircmsg.IrcMessage{}

	sql := "SELECT ts, fromNick, type, line, buffer FROM messages WHERE uid = ? AND netid = ? AND buffer = ? AND ts < ? ORDER BY ts DESC LIMIT ?"
	rows, err := ds.db.Query(sql, userID, networkID, strings.ToLower(buffer), int32(from.UTC().Unix()), num)
	if err != nil {
		log.Println("GetBeforeTime() error: " + err.Error())
		return messages
	}
	for rows.Next() {
		m := rowToIrcMessage(rows)
		messages = append(messages, m)
	}

	// Reverse the messages so they're in order
	for i := 0; i < len(messages)/2; i++ {
		j := len(messages) - i - 1
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages
}
func (ds *SqliteMessageDatastore) Search(string, string, string, time.Time, time.Time, int) []*ircmsg.IrcMessage {
	return []*ircmsg.IrcMessage{}
}

func rowToIrcMessage(rows *sql.Rows) *ircmsg.IrcMessage {
	var ts int32
	var from string
	var messageType int
	var line string
	var buffer string
	rows.Scan(&ts, &from, &messageType, &line, &buffer)

	v := ircmsg.TagValue{}
	v.Value = time.Unix(int64(ts), 0).UTC().Format(time.RFC3339)
	v.HasValue = true
	mTags := make(map[string]ircmsg.TagValue)
	mTags["time"] = v

	mPrefix := from
	mCommand := "PRIVMSG"
	mParams := []string{
		buffer,
		line,
	}

	if messageType == TYPE_ACTION {
		mParams[1] = "\x01" + mParams[1]
	} else if messageType == TYPE_NOTICE {
		mCommand = "NOTICE"
	}

	m := ircmsg.MakeMessage(&mTags, mPrefix, mCommand, mParams...)
	return &m
}

func extractMessageParts(event *ircbnc.HookIrcRaw) (string, string, int, string) {
	messageType := TYPE_MESSAGE
	from := ""
	buffer := ""
	line := ""

	message := event.Message
	server := event.Server

	prefixNick, _, _ := ircbnc.SplitMask(message.Prefix)

	if event.FromServer {
		switch message.Command {
		case "PRIVMSG":
			line = message.Params[1]
			if strings.HasPrefix(line, "\x01ACTION") {
				messageType = TYPE_ACTION
				line = line[1:]
			} else if !strings.HasPrefix(line, "\x01") {
				messageType = TYPE_MESSAGE
			} else {
				return "", "", 0, ""
			}

			if message.Params[0] == server.Foo.Nick {
				buffer = prefixNick
				from = prefixNick
			} else {
				buffer = message.Params[0]
				from = prefixNick
			}

		case "NOTICE":
			line = message.Params[1]
			if !strings.HasPrefix(line, "\x01") {
				messageType = TYPE_NOTICE
			} else {
				return "", "", 0, ""
			}

			if message.Params[0] == server.Foo.Nick {
				buffer = prefixNick
				from = prefixNick
			} else {
				buffer = message.Params[0]
				from = prefixNick
			}
		}
	} else if event.FromClient && event.Listener.ServerConnection != nil {
		switch message.Command {
		case "PRIVMSG":
			line = message.Params[1]
			if strings.HasPrefix(line, "\x01ACTION") {
				messageType = TYPE_ACTION
				line = line[1:]
			} else if !strings.HasPrefix(line, "\x01") {
				messageType = TYPE_MESSAGE
			} else {
				return "", "", 0, ""
			}

			buffer = message.Params[0]
			from = event.Listener.ServerConnection.Nickname

		case "NOTICE":
			line = message.Params[1]
			if !strings.HasPrefix(line, "\x01") {
				messageType = TYPE_NOTICE
			} else {
				return "", "", 0, ""
			}

			buffer = message.Params[0]
			from = event.Listener.ServerConnection.Nickname
		}
	}

	from = strings.ToLower(from)
	buffer = strings.ToLower(buffer)

	return from, buffer, messageType, line
}
