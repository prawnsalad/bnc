// written by Daniel Oaks <daniel@danieloaks.net>
// released under the ISC license

package ircbnc

import "github.com/DanielOaks/girc-go/ircmsg"

// nickHandler handles the NICK command.
func nickHandler(listener *Listener, msg ircmsg.IrcMessage) bool {
	// always reject dodgy nicknames, makes things immensely easier
	nick, nickError := IrcName(msg.Params[0], false)
	if nickError != nil {
		listener.Send(nil, "", "422", listener.ClientNick, msg.Params[0], "Erroneus nickname")
		return false
	}

	// we ignore NICK messages during registration
	if !listener.Registered {
		listener.ClientNick = nick
		listener.regLocks["NICK"] = true
		return false
	}
	//TODO(dan): Handle NICK messages when connected to servers.
	listener.Send(nil, "", "ERROR", "We're supposed to handle NICK changes here!")
	return true
}

// userHandler handles the USER command.
func userHandler(listener *Listener, msg ircmsg.IrcMessage) bool {
	// we ignore USER messages entirely
	if !listener.Registered {
		listener.regLocks["USER"] = true
	}
	return false
}

// passHandler handles the PASS command.
func passHandler(listener *Listener, msg ircmsg.IrcMessage) bool {
	//TODO(dan): Handle PASS messages.
	return false
}