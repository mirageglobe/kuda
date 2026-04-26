package network

type parserState int

const (
	stateText parserState = iota
	stateIAC
	stateDO
	stateDONT
	stateWILL
	stateWONT
	stateSB
	stateSBIAC
)

// telnetParser holds per-connection telnet state machine state.
type telnetParser struct {
	client  *Client
	state   parserState
	sbBuf   []byte
	textBuf []byte
}

func (p *telnetParser) flushText() {
	if len(p.textBuf) > 0 {
		p.client.events <- Event{Type: EventText, Data: p.textBuf}
		p.textBuf = nil
	}
}

func (p *telnetParser) handleByte(b byte) {
	switch p.state {
	case stateText:
		if b == IAC {
			p.flushText()
			p.state = stateIAC
		} else {
			p.textBuf = append(p.textBuf, b)
			if len(p.textBuf) >= 512 { // flush large blocks early
				p.flushText()
			}
		}

	case stateIAC:
		switch b {
		case IAC:
			p.textBuf = append(p.textBuf, IAC)
			p.state = stateText
		case GA:
			p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{GA}}
			p.state = stateText
		case DO:
			p.state = stateDO
		case DONT:
			p.state = stateDONT
		case WILL:
			p.state = stateWILL
		case WONT:
			p.state = stateWONT
		case SB:
			p.state = stateSB
			p.sbBuf = nil
		default:
			p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{b}}
			p.state = stateText
		}

	case stateDO:
		p.handleNegotiation(DO, b)
		p.state = stateText

	case stateDONT:
		p.handleNegotiation(DONT, b)
		p.state = stateText

	case stateWILL:
		p.handleNegotiation(WILL, b)
		p.state = stateText

	case stateWONT:
		p.handleNegotiation(WONT, b)
		p.state = stateText

	case stateSB:
		if b == IAC {
			p.state = stateSBIAC
		} else {
			p.sbBuf = append(p.sbBuf, b)
		}

	case stateSBIAC:
		if b == SE {
			p.handleSubnegotiation(p.sbBuf)
			p.state = stateText
		} else if b == IAC {
			p.sbBuf = append(p.sbBuf, IAC)
			p.state = stateSB
		} else {
			// malformed SB sequence — absorb and continue
			p.sbBuf = append(p.sbBuf, IAC, b)
			p.state = stateSB
		}
	}
}

func (p *telnetParser) handleNegotiation(cmd, option byte) {
	// handle supported options explicitly
	switch cmd {
	case WILL:
		if option == TelnetOptionEcho {
			p.client.Write([]byte{IAC, DO, option}) //nolint:errcheck
			p.client.echoActive.Store(true)
			p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{cmd, option}}
			return
		}
		if option == TelnetOptionMCCP {
			p.client.Write([]byte{IAC, DO, option}) //nolint:errcheck
			return
		}
		if option == TelnetOptionGMCP {
			p.client.Write([]byte{IAC, DO, option}) //nolint:errcheck
			p.client.gmcpActive.Store(true)
			// Initial handshake
			p.client.Write([]byte{IAC, SB, TelnetOptionGMCP})
			p.client.Write([]byte(`Core.Hello {"client": "kuda", "version": "0.1.0"}`))
			p.client.Write([]byte{IAC, SE})

			p.client.Write([]byte{IAC, SB, TelnetOptionGMCP})
			p.client.Write([]byte(`Core.Supports.Set ["Char 1", "Char.Vitals 1", "Room 1", "Comm 1"]`))
			p.client.Write([]byte{IAC, SE})
			return
		}
		p.client.Write([]byte{IAC, DONT, option}) //nolint:errcheck
	case WONT:
		if option == TelnetOptionEcho {
			p.client.Write([]byte{IAC, DONT, option}) //nolint:errcheck
			p.client.echoActive.Store(false)
			p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{cmd, option}}
			return
		}
	case DO:
		if option == TelnetOptionTTYPE {
			p.client.Write([]byte{IAC, WILL, option}) //nolint:errcheck
			return
		}
		p.client.Write([]byte{IAC, WONT, option}) //nolint:errcheck
	}
	p.client.events <- Event{Type: EventTelnetCommand, Data: []byte{cmd, option}}
}

func (p *telnetParser) handleSubnegotiation(data []byte) {
	if len(data) == 0 {
		return
	}
	switch data[0] {
	case TelnetOptionGMCP:
		p.client.events <- Event{Type: EventGMCP, Data: data[1:]}
	case TelnetOptionMCCP:
		p.flushText()
		p.client.pendingMCCP = true
		p.client.mccpActive.Store(true)
	case TelnetOptionTTYPE:
		if len(data) > 1 && data[1] == 1 { // SEND
			// reply with IS KUDA
			p.client.Write([]byte{IAC, SB, TelnetOptionTTYPE, 0}) // IS
			p.client.Write([]byte("KUDA"))                        // terminal name
			p.client.Write([]byte{IAC, SE})                       // end
		}
	}
}
