package bncComponentControl

import (
	"strconv"
	"strings"

	"github.com/goshuirc/bnc/lib"
	"github.com/goshuirc/irc-go/ircmsg"
)

// Nick of the controller
const CONTROL_NICK = "*goshu"
const CONTROL_PREFIX = CONTROL_NICK + "!bnc@irc.goshu"

func Run(manager *ircbnc.Manager) {
	manager.Bus.Register(ircbnc.HookIrcRawName, onMessage)
}

func onMessage(hook interface{}) {
	event := hook.(*ircbnc.HookIrcRaw)
	if !event.FromClient {
		return
	}

	msg := event.Message
	listener := event.Listener

	if msg.Command != "PRIVMSG" || msg.Params[0] != CONTROL_NICK {
		return
	}

	// Stop the message from being sent upstream
	event.Halt = true

	parts := strings.Split(msg.Params[1], " ")
	command := strings.ToLower(parts[0])
	params := parts[1:]

	switch command {
	case "listnetworks":
		commandListNetworks(listener, params, msg)
	case "addnetwork":
		commandAddNetwork(listener, params, msg)
	case "connect":
		commandConnectNetwork(listener, params, msg)
	case "disconnect":
		commandDisconnectNetwork(listener, params, msg)
	}
}

func commandConnectNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	netName := listener.ServerConnection.Name
	if len(params) >= 1 {
		netName = params[0]
	}

	net, exists := listener.User.Networks[netName]
	if !exists {
		listener.Send(nil, CONTROL_PREFIX, "PRIVMSG", listener.ClientNick, "Network "+netName+" not found")
		return
	}

	net.Connect()
}

func commandDisconnectNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	netName := listener.ServerConnection.Name
	if len(params) >= 1 {
		netName = params[0]
	}

	net, exists := listener.User.Networks[netName]
	if !exists {
		listener.Send(nil, CONTROL_PREFIX, "PRIVMSG", listener.ClientNick, "Network "+netName+" not found")
		return
	}

	net.Disconnect()
}

func commandListNetworks(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	table := NewTable()
	table.SetHeader([]string{"Name", "Nick", "Connected", "Address"})

	for _, network := range listener.User.Networks {
		connected := "No"
		if network.Foo.HasRegistered {
			connected = "Yes"
		}

		address := network.Addresses[0].Host + ":"
		if network.Addresses[0].UseTLS {
			address += "+"
		}
		address += strconv.Itoa(network.Addresses[0].Port)

		name := network.Name
		if network == listener.ServerConnection {
			name = "*" + name
		}

		table.Append([]string{name, network.Nickname, connected, address})
	}

	table.RenderToListener(listener, CONTROL_PREFIX, "PRIVMSG")
}

func commandAddNetwork(listener *ircbnc.Listener, params []string, message ircmsg.IrcMessage) {
	sendUsage := func() {
		listener.Send(nil, CONTROL_PREFIX, "PRIVMSG", listener.ClientNick, "Usage: addnetwork name address [port] [password]")
		listener.Send(nil, CONTROL_PREFIX, "PRIVMSG", listener.ClientNick, "To use SSL/TLS, add + infront of the port number.")
	}

	if len(params) < 2 {
		sendUsage()
		return
	}

	netName := params[0]
	netAddress := params[1]
	netPort := 6667
	netTls := false
	netPassword := ""

	if len(params) >= 3 {
		portParam := params[2]
		if len(portParam) > 2 && portParam[:1] == "+" {
			netTls = true
			portParam = portParam[1:]
		}
		netPort, _ = strconv.Atoi(portParam)
	}

	if len(params) >= 4 {
		netPassword = params[3]
	}

	if netName == "" || netAddress == "" || netPort == 0 {
		sendUsage()
		return
	}

	connection := ircbnc.NewServerConnection()
	connection.User = listener.User
	connection.Name = netName
	connection.Password = netPassword

	newAddress := ircbnc.ServerConnectionAddress{
		Host:      netAddress,
		Port:      netPort,
		UseTLS:    netTls,
		VerifyTLS: false,
	}
	connection.Addresses = append(connection.Addresses, newAddress)
	listener.User.Networks[connection.Name] = connection

	err := listener.Manager.Ds.SaveConnection(connection)
	if err != nil {
		listener.Send(nil, CONTROL_PREFIX, "PRIVMSG", listener.ClientNick, "Could not save the new network")
	} else {
		listener.Send(nil, CONTROL_PREFIX, "PRIVMSG", listener.ClientNick, "New network saved")
	}
}
